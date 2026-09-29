package footballdata

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

var (
	ErrMissingAPIKey = errors.New("missing football-data.org API key")
)

type Client interface {
	FetchLeaguePlayers(ctx context.Context, leagueCode string) ([]model.Player, error)
}

type FootballDataClient struct {
	httpClient   *http.Client
	baseURL      string
	apiKey       string
	requestDelay time.Duration
}

type ClientOption func(*FootballDataClient)

func WithBaseURL(url string) ClientOption {
	return func(c *FootballDataClient) {
		c.baseURL = strings.TrimRight(url, "/")
	}
}

func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *FootballDataClient) {
		c.httpClient = httpClient
	}
}

func NewClient(apiKey string, delay time.Duration, opts ...ClientOption) *FootballDataClient {
	c := &FootballDataClient{
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		baseURL:      "https://api.football-data.org",
		apiKey:       apiKey,
		requestDelay: delay,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *FootballDataClient) FetchLeaguePlayers(ctx context.Context, leagueCode string) ([]model.Player, error) {
	if c.apiKey == "" {
		return nil, ErrMissingAPIKey
	}

	compTeams, err := c.fetchCompetitionTeams(ctx, leagueCode)
	if err != nil {
		return nil, fmt.Errorf("fetch competition teams for %s: %w", leagueCode, err)
	}

	leagueName := compTeams.Competition.Name
	if leagueName == "" {
		leagueName = defaultLeagueName(leagueCode)
	}

	players := make([]model.Player, 0)
	for _, team := range compTeams.Teams {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		squad := team.Squad
		if len(squad) == 0 {
			// Fall back to fetching team details if squad is empty
			if c.requestDelay > 0 {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(c.requestDelay):
				}
			}

			teamDetail, err := c.fetchTeamDetail(ctx, team.ID)
			if err != nil {
				return nil, fmt.Errorf("fetch team detail for team %d (%s): %w", team.ID, team.Name, err)
			}
			squad = teamDetail.Squad
		}

		for _, p := range squad {
			mapped := mapPlayer(p, team.Name, leagueName, leagueCode)
			players = append(players, mapped)
		}
	}

	return players, nil
}

func (c *FootballDataClient) fetchCompetitionTeams(ctx context.Context, code string) (CompetitionTeamsResponse, error) {
	url := fmt.Sprintf("%s/v4/competitions/%s/teams", c.baseURL, code)
	var resp CompetitionTeamsResponse
	err := c.doRequest(ctx, url, &resp)
	return resp, err
}

func (c *FootballDataClient) fetchTeamDetail(ctx context.Context, teamID int64) (TeamDetailResponse, error) {
	url := fmt.Sprintf("%s/v4/teams/%d", c.baseURL, teamID)
	var resp TeamDetailResponse
	err := c.doRequest(ctx, url, &resp)
	return resp, err
}

func (c *FootballDataClient) doRequest(ctx context.Context, url string, dest any) error {
	retries := 0
	maxRetries := 3

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}

		req.Header.Set("X-Auth-Token", c.apiKey)
		req.Header.Set("Accept", "application/json")

		res, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("http request: %w", err)
		}

		if res.StatusCode == http.StatusOK {
			body, readErr := io.ReadAll(res.Body)
			_ = res.Body.Close()
			if readErr != nil {
				return fmt.Errorf("read response body: %w", readErr)
			}
			if unmarshalErr := json.Unmarshal(body, dest); unmarshalErr != nil {
				return fmt.Errorf("unmarshal response: %w", unmarshalErr)
			}
			return nil
		}

		if res.StatusCode == http.StatusTooManyRequests {
			_ = res.Body.Close()
			retries++
			if retries > maxRetries {
				return model.ErrRateLimitExceeded
			}

			waitTime := time.Duration(1<<retries) * time.Second
			if retryHeader := res.Header.Get("Retry-After"); retryHeader != "" {
				if seconds, parseErr := strconv.Atoi(retryHeader); parseErr == nil && seconds > 0 {
					waitTime = time.Duration(seconds) * time.Second
				}
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(waitTime):
				continue
			}
		}

		body, _ := io.ReadAll(res.Body)
		_ = res.Body.Close()
		return fmt.Errorf("external API error: status %d body %s", res.StatusCode, string(body))
	}
}

func mapPlayer(p PlayerPayload, clubName, leagueName, leagueCode string) model.Player {
	pos := normalizePosition(p.Position)

	var shirtNumber *int
	if p.ShirtNumber != nil && *p.ShirtNumber > 0 {
		shirtNumber = p.ShirtNumber
	}

	var dob *string
	if p.DateOfBirth != nil && strings.TrimSpace(*p.DateOfBirth) != "" {
		trimmed := strings.TrimSpace(*p.DateOfBirth)
		dob = &trimmed
	}

	var nat *string
	if p.Nationality != nil && strings.TrimSpace(*p.Nationality) != "" {
		trimmed := strings.TrimSpace(*p.Nationality)
		nat = &trimmed
	}

	return model.Player{
		ExternalID:  p.ID,
		Name:        strings.TrimSpace(p.Name),
		ClubName:    clubName,
		LeagueName:  leagueName,
		LeagueCode:  leagueCode,
		Position:    pos,
		DateOfBirth: dob,
		Nationality: nat,
		ShirtNumber: shirtNumber,
		Active:      true,
	}
}

func normalizePosition(raw string) string {
	raw = strings.TrimSpace(raw)
	switch strings.ToLower(raw) {
	case "goalkeeper":
		return "Goalkeeper"
	case "defence", "defender":
		return "Defender"
	case "midfield", "midfielder":
		return "Midfielder"
	case "offence", "forward", "attacker":
		return "Attacker"
	default:
		if raw == "" {
			return "Unknown"
		}
		return raw
	}
}

func defaultLeagueName(code string) string {
	switch code {
	case "PL":
		return "Premier League"
	case "BL1":
		return "Bundesliga"
	case "PD":
		return "La Liga"
	case "SA":
		return "Serie A"
	case "FL1":
		return "Ligue 1"
	default:
		return code
	}
}
