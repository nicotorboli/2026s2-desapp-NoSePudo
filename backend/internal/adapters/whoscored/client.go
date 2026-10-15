package whoscored

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

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

// leagueConfig mapea cada código de liga del catálogo a los identificadores
// que WhoScored usa en sus URLs.
type leagueConfig struct {
	regionSlug   string
	regionID     int64
	tournamentID int64
}

var leagueConfigs = map[string]leagueConfig{
	"PL":  {regionID: 252, tournamentID: 2, regionSlug: "England-Premier-League"},
	"BL1": {regionID: 81, tournamentID: 3, regionSlug: "Germany-Bundesliga"},
	"PD":  {regionID: 206, tournamentID: 4, regionSlug: "Spain-LaLiga"},
	"SA":  {regionID: 108, tournamentID: 5, regionSlug: "Italy-Serie-A"},
	"FL1": {regionID: 74, tournamentID: 22, regionSlug: "France-Ligue-1"},
}

const userAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"

const (
	defaultBaseURL      = "https://www.whoscored.com"
	defaultRequestDelay = 4 * time.Second
)

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

// FetchLeagueStatistics descarga las estadísticas de la liga indicada.
// Primero visita la página regional para adquirir las cookies de sesión del
// proveedor y después recorre las páginas del feed de estadísticas.
func (c *WhoscoredClient) FetchLeagueStatistics(ctx context.Context, leagueCode string) ([]model.PlayerStatisticsRow, error) {
	league, ok := leagueConfigs[leagueCode]
	if !ok {
		return nil, fmt.Errorf("unsupported league code %s", leagueCode)
	}

	collector, err := c.newCollector(ctx)
	if err != nil {
		return nil, err
	}

	state := &scrapeState{totalPages: 1}
	collector.OnResponse(state.onResponse)

	seedURL := fmt.Sprintf("%s/Regions/%d/Tournaments/%d/%s", c.baseURL, league.regionID, league.tournamentID, league.regionSlug)
	if err := collector.Visit(seedURL); err != nil {
		return nil, fmt.Errorf("seed visit: %w", err)
	}
	collector.Wait()
	if state.fetchErr != nil {
		return nil, state.fetchErr
	}

	for page := 1; page <= state.totalPages; page++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if err := collector.Visit(c.feedURL(league, page)); err != nil {
			return nil, fmt.Errorf("visit statistics page %d: %w", page, err)
		}
		collector.Wait()
		if state.fetchErr != nil {
			return nil, state.fetchErr
		}
	}

	return state.rows, nil
}

// scrapeState acumula el resultado de la descarga a medida que el collector
// visita las páginas del feed.
type scrapeState struct {
	fetchErr   error
	rows       []model.PlayerStatisticsRow
	totalPages int
	mu         sync.Mutex
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
	page, err := parseFeedPage(res.Body)
	if err != nil {
		s.fetchErr = err
		return
	}
	s.mu.Lock()
	s.rows = append(s.rows, page.rows...)
	if page.totalPages > s.totalPages {
		s.totalPages = page.totalPages
	}
	s.mu.Unlock()
}

func (c *WhoscoredClient) newCollector(ctx context.Context) (*colly.Collector, error) {
	collector := colly.NewCollector(
		colly.AllowedDomains("www.whoscored.com", "127.0.0.1", "localhost"),
		colly.ParseHTTPErrorResponse(),
		colly.StdlibContext(ctx),
		colly.UserAgent(userAgent),
	)
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

func (c *WhoscoredClient) feedURL(league leagueConfig, page int) string {
	params := url.Values{}
	params.Set("category", "summary")
	params.Set("subcategory", "all")
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
	params.Set("tournamentId", strconv.FormatInt(league.tournamentID, 10))
	params.Set("useStatisticsFilter", "true")
	params.Set("formationId", "")
	params.Set("includeZeroMinSubs", "false")
	params.Set("page", strconv.Itoa(page))
	return fmt.Sprintf("%s/statisticsfeed/1/GetPlayerStatistics?%s", c.baseURL, params.Encode())
}

func looksLikeChallenge(body []byte) bool {
	lower := strings.ToLower(string(body))
	return strings.Contains(lower, "incap_ses") || strings.Contains(lower, "captcha")
}
