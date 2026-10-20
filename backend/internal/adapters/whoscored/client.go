package whoscored

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
	"github.com/gocolly/colly/v2"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

var (
	ErrBotChallenge      = errors.New("bot challenge detected")
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
)

type Client interface {
	FetchLeagueStatistics(ctx context.Context, leagueCode string) ([]model.PlayerStatisticsRow, error)
}

// leagueTournamentIDs mapea cada código de liga del catálogo al tournamentId
// que WhoScored usa en su feed de estadísticas.
var leagueTournamentIDs = map[string]int64{
	"PL":  2,
	"BL1": 3,
	"PD":  4,
	"SA":  5,
	"FL1": 22,
}

const userAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"

const (
	defaultBaseURL      = "https://www.whoscored.com"
	defaultRequestDelay = 4 * time.Second
)

// feedSubcategories son los dos feeds por liga: el ofensivo trae las métricas
// de ataque y el defensivo las defensivas; el merge por playerId los une.
var feedSubcategories = []string{"offensive", "defensive"}

type WhoscoredClient struct {
	baseURL      string
	requestDelay time.Duration
}

type ClientOption func(*WhoscoredClient)

func WithBaseURL(u string) ClientOption {
	return func(c *WhoscoredClient) {
		c.baseURL = strings.TrimRight(u, "/")
	}
}

func WithRequestDelay(d time.Duration) ClientOption {
	return func(c *WhoscoredClient) {
		c.requestDelay = d
	}
}

func NewClient(opts ...ClientOption) *WhoscoredClient {
	c := &WhoscoredClient{
		baseURL:      defaultBaseURL,
		requestDelay: defaultRequestDelay,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// FetchLeagueStatistics descarga las estadísticas de la liga indicada desde
// los feeds públicos de WhoScored. El primer pedido resuelve el handshake de
// Cloudflare (301 que setea la cookie __cf_bm) y los siguientes van directo.
func (c *WhoscoredClient) FetchLeagueStatistics(ctx context.Context, leagueCode string) ([]model.PlayerStatisticsRow, error) {
	tournamentID, ok := leagueTournamentIDs[leagueCode]
	if !ok {
		return nil, fmt.Errorf("unsupported league code %s", leagueCode)
	}

	collector, err := c.newCollector(ctx)
	if err != nil {
		return nil, err
	}

	state := &scrapeState{}
	collector.OnResponse(state.onResponse)

	for _, subcategory := range feedSubcategories {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if err := collector.Visit(c.feedURL(tournamentID, subcategory)); err != nil {
			return nil, fmt.Errorf("visit statistics feed %s: %w", subcategory, err)
		}
		collector.Wait()
		if state.fetchErr != nil {
			return nil, state.fetchErr
		}
	}

	return mergeFeeds(state.offensive, state.defensive), nil
}

// scrapeState acumula las filas de cada feed a medida que el collector visita
// las URLs. Los callbacks corren en la goroutine de Visit, así que no hay
// acceso concurrente.
type scrapeState struct {
	fetchErr  error
	offensive []playerTableStat
	defensive []playerTableStat
}

func (s *scrapeState) onResponse(res *colly.Response) {
	if res.StatusCode == http.StatusTooManyRequests {
		s.fetchErr = ErrRateLimitExceeded
		return
	}
	if res.StatusCode == http.StatusForbidden {
		s.fetchErr = ErrBotChallenge
		return
	}
	if res.StatusCode != http.StatusOK {
		s.fetchErr = fmt.Errorf("unexpected status code %d", res.StatusCode)
		return
	}
	if !strings.Contains(res.Request.URL.Path, "/statisticsfeed/") {
		return
	}
	if looksLikeChallenge(res.Body) {
		s.fetchErr = ErrBotChallenge
		return
	}
	rows, err := parseFeedPage(res.Body)
	if err != nil {
		s.fetchErr = err
		return
	}
	if res.Request.URL.Query().Get("subcategory") == "offensive" {
		s.offensive = rows
		return
	}
	s.defensive = rows
}

func (c *WhoscoredClient) newCollector(ctx context.Context) (*colly.Collector, error) {
	tlsClient, err := tls_client.NewHttpClient(nil,
		tls_client.WithClientProfile(profiles.Chrome_131),
		tls_client.WithTimeoutSeconds(30),
		tls_client.WithForceHttp1(),
	)
	if err != nil {
		return nil, fmt.Errorf("create tls client: %w", err)
	}

	collector := colly.NewCollector(
		colly.AllowedDomains("www.whoscored.com", "127.0.0.1", "localhost"),
		colly.ParseHTTPErrorResponse(),
		colly.StdlibContext(ctx),
		colly.UserAgent(userAgent),
	)
	collector.WithTransport(&tlsClientRoundTripper{dialTLS: tlsClient.GetTLSDialer()})
	collector.SetRequestTimeout(30 * time.Second)
	if err := collector.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 1,
		RandomDelay: c.requestDelay,
	}); err != nil {
		return nil, fmt.Errorf("configure request rate limit: %w", err)
	}
	return collector, nil
}

// tlsClientRoundTripper habla HTTP/1.1 sobre conexiones TLS con el ClientHello
// de Chrome 131 que produce tls-client. Cloudflare distingue a los clientes
// por ese fingerprint y bloquea al net/http estándar.
type tlsClientRoundTripper struct {
	dialTLS tls_client.TLSDialerFunc
}

func (t *tlsClientRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	addr := req.URL.Host
	if !strings.Contains(addr, ":") {
		addr = net.JoinHostPort(req.URL.Hostname(), "443")
	}

	conn, err := t.dialTLS(req.Context(), "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("tls dial: %w", err)
	}
	defer func() { _ = conn.Close() }()

	if writeErr := req.Write(conn); writeErr != nil {
		return nil, fmt.Errorf("write request: %w", writeErr)
	}

	resp, err := http.ReadResponse(bufio.NewReader(conn), req)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	return resp, nil
}

func (c *WhoscoredClient) feedURL(tournamentID int64, subcategory string) string {
	params := url.Values{}
	params.Set("category", "summary")
	params.Set("subcategory", subcategory)
	params.Set("statsAccumulationType", "0")
	params.Set("isCurrent", "true")
	params.Set("playerId", "")
	params.Set("teamIds", "")
	params.Set("matchId", "")
	params.Set("stageId", "")
	params.Set("tournamentOptions", "2")
	params.Set("sortBy", "Rating")
	params.Set("sortAscending", "")
	params.Set("age", "")
	params.Set("ageComparisonType", "")
	params.Set("appearances", "")
	params.Set("appearancesComparisonType", "")
	params.Set("field", "Overall")
	params.Set("nationality", "")
	params.Set("positionOptions", "")
	params.Set("timeOfTheGame", "")
	params.Set("timezoneOffset", "")
	params.Set("tournamentId", strconv.FormatInt(tournamentID, 10))
	params.Set("useStatisticsFilter", "true")
	params.Set("formationId", "")
	params.Set("includeZeroMinSubs", "false")
	params.Set("page", "1")
	return fmt.Sprintf("%s/statisticsfeed/1/GetPlayerStatistics?%s", c.baseURL, params.Encode())
}

func looksLikeChallenge(body []byte) bool {
	lower := strings.ToLower(string(body))
	return strings.Contains(lower, "incap_ses") || strings.Contains(lower, "captcha")
}
