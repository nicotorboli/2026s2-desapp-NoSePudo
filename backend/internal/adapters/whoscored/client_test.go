package whoscored_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/adapters/whoscored"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

const offensiveFeed = `{
  "playerTableStats": [
    {"playerId": 1, "name": "Bukayo Saka", "teamName": "Arsenal", "goal": 10, "assistTotal": 5, "shotsPerGame": 2.5, "keyPassPerGame": 1.8, "dribbleWonPerGame": 2.1, "rating": 7.8},
    {"playerId": 2, "name": "Gabriel Magalhaes", "teamName": "Arsenal", "goal": 2, "assistTotal": 1, "shotsPerGame": 0.8, "keyPassPerGame": 0.3, "dribbleWonPerGame": 0.1, "rating": 7.1}
  ],
  "paging": {"page": 1, "numberPages": 1}
}`

const defensiveFeed = `{
  "playerTableStats": [
    {"playerId": 1, "name": "Bukayo Saka", "teamName": "Arsenal", "tacklePerGame": 1.1, "interceptionPerGame": 0.6, "foulsPerGame": 0.7, "clearancePerGame": 0.4, "outfielderBlockPerGame": 0.2, "wasDribbledPerGame": 0.9, "rating": 7.8},
    {"playerId": 2, "name": "Gabriel Magalhaes", "teamName": "Arsenal", "tacklePerGame": 2.0, "interceptionPerGame": 1.5, "foulsPerGame": 1.2, "clearancePerGame": 4.5, "outfielderBlockPerGame": 1.1, "wasDribbledPerGame": 0.3, "rating": 7.1}
  ],
  "paging": {"page": 1, "numberPages": 1}
}`

// feedServer devuelve un servidor de fixtures que distingue los dos feeds por
// el query param subcategory y cuenta cuántas veces se pidió cada uno.
func feedServer(t *testing.T, status int) (*httptest.Server, *int, *int) {
	t.Helper()
	offensiveHits := 0
	defensiveHits := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if status != 0 && status != http.StatusOK {
			w.WriteHeader(status)
			return
		}
		if r.URL.Path != "/statisticsfeed/1/GetPlayerStatistics" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Query().Get("subcategory") {
		case "offensive":
			offensiveHits++
			_, _ = w.Write([]byte(offensiveFeed))
		case "defensive":
			defensiveHits++
			_, _ = w.Write([]byte(defensiveFeed))
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	t.Cleanup(server.Close)

	return server, &offensiveHits, &defensiveHits
}

func assertRow(t *testing.T, got, want model.PlayerStatisticsRow) {
	t.Helper()
	if got != want {
		t.Errorf("fila mal fusionada:\n got  %+v\n want %+v", got, want)
	}
}

func TestFetchLeagueStatistics_MergesOffensiveAndDefensive(t *testing.T) {
	server, offensiveHits, defensiveHits := feedServer(t, 0)

	client := whoscored.NewClient(
		whoscored.WithBaseURL(server.URL),
		whoscored.WithRequestDelay(0),
	)

	rows, err := client.FetchLeagueStatistics(t.Context(), "PL")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if *offensiveHits != 1 || *defensiveHits != 1 {
		t.Fatalf("expected 1 offensive and 1 defensive request, got %d and %d", *offensiveHits, *defensiveHits)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 merged rows, got %d", len(rows))
	}

	assertRow(t, rows[0], model.PlayerStatisticsRow{
		Name: "Bukayo Saka", ClubName: "Arsenal",
		Goals: 10, Assists: 5,
		Shots: 2.5, KeyPasses: 1.8, Dribbles: 2.1, Tackles: 1.1,
		Interceptions: 0.6, Fouls: 0.7, Clears: 0.4, Blocks: 0.2, WasDribbled: 0.9, Rating: 7.8,
	})
	assertRow(t, rows[1], model.PlayerStatisticsRow{
		Name: "Gabriel Magalhaes", ClubName: "Arsenal",
		Goals: 2, Assists: 1,
		Shots: 0.8, KeyPasses: 0.3, Dribbles: 0.1, Tackles: 2.0,
		Interceptions: 1.5, Fouls: 1.2, Clears: 4.5, Blocks: 1.1, WasDribbled: 0.3, Rating: 7.1,
	})
}

func TestFetchLeagueStatistics_UnsupportedLeague(t *testing.T) {
	server, _, _ := feedServer(t, 0)
	client := whoscored.NewClient(
		whoscored.WithBaseURL(server.URL),
		whoscored.WithRequestDelay(0),
	)

	if _, err := client.FetchLeagueStatistics(t.Context(), "XX"); err == nil {
		t.Fatal("expected error for unsupported league, got nil")
	}
}

func TestFetchLeagueStatistics_BotChallenge(t *testing.T) {
	server, _, _ := feedServer(t, http.StatusForbidden)
	client := whoscored.NewClient(
		whoscored.WithBaseURL(server.URL),
		whoscored.WithRequestDelay(0),
	)

	if _, err := client.FetchLeagueStatistics(t.Context(), "PL"); !errors.Is(err, whoscored.ErrBotChallenge) {
		t.Fatalf("expected ErrBotChallenge, got %v", err)
	}
}

func TestFetchLeagueStatistics_RateLimit(t *testing.T) {
	server, _, _ := feedServer(t, http.StatusTooManyRequests)
	client := whoscored.NewClient(
		whoscored.WithBaseURL(server.URL),
		whoscored.WithRequestDelay(0),
	)

	if _, err := client.FetchLeagueStatistics(t.Context(), "PL"); !errors.Is(err, whoscored.ErrRateLimitExceeded) {
		t.Fatalf("expected ErrRateLimitExceeded, got %v", err)
	}
}
