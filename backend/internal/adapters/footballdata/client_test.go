package footballdata_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/adapters/footballdata"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

func mockCompetitionServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Auth-Token") != "test-api-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if r.URL.Path == "/v4/competitions/PL/teams" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"competition": {"id": 2021, "name": "Premier League", "code": "PL"},
				"teams": [
					{
						"id": 57,
						"name": "Arsenal FC",
						"squad": [
							{
								"id": 7821,
								"name": "Bukayo Saka",
								"position": "Offence",
								"dateOfBirth": "2001-09-05",
								"nationality": "England",
								"shirtNumber": 7
							},
							{
								"id": 7822,
								"name": "Gabriel Magalhães",
								"position": "Defence",
								"dateOfBirth": "1997-12-19",
								"nationality": "Brazil",
								"shirtNumber": 6
							},
							{
								"id": 7823,
								"name": "Martin Ødegaard",
								"position": "Midfield",
								"dateOfBirth": "1998-12-17",
								"nationality": "Norway",
								"shirtNumber": 8
							},
							{
								"id": 7824,
								"name": "David Raya",
								"position": "Goalkeeper",
								"dateOfBirth": "1995-09-15",
								"nationality": "Spain",
								"shirtNumber": 22
							},
							{
								"id": 7825,
								"name": "Unknown Player",
								"position": "Forward",
								"dateOfBirth": null,
								"nationality": null,
								"shirtNumber": 0
							}
						]
					}
				]
			}`))
			return
		}

		http.NotFound(w, r)
	}))
}

func TestClient_FetchLeaguePlayers_Success(t *testing.T) {
	ts := mockCompetitionServer(t)
	defer ts.Close()

	client := footballdata.NewClient("test-api-key", 0, footballdata.WithBaseURL(ts.URL))

	players, err := client.FetchLeaguePlayers(context.Background(), "PL")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(players) != 5 {
		t.Fatalf("expected 5 players, got %d", len(players))
	}

	assertPlayerMapping(t, players)
}

func assertPlayerMapping(t *testing.T, players []model.Player) {
	t.Helper()
	// 1. Attacker mapping
	saka := players[0]
	if saka.Name != "Bukayo Saka" || saka.Position != "Attacker" || saka.ClubName != "Arsenal FC" || saka.LeagueName != "Premier League" || saka.LeagueCode != "PL" {
		t.Errorf("unexpected saka data: %+v", saka)
	}

	// 2. Positions
	if players[1].Position != "Defender" {
		t.Errorf("expected Defender, got %s", players[1].Position)
	}
	if players[2].Position != "Midfielder" {
		t.Errorf("expected Midfielder, got %s", players[2].Position)
	}
	if players[3].Position != "Goalkeeper" {
		t.Errorf("expected Goalkeeper, got %s", players[3].Position)
	}

	// 3. Fallbacks
	unknown := players[4]
	if unknown.Position != "Attacker" || unknown.ShirtNumber != nil || unknown.DateOfBirth != nil || unknown.Nationality != nil {
		t.Errorf("unexpected unknown player fallbacks: %+v", unknown)
	}
}

func TestClient_FetchLeaguePlayers_EmptySquadFallback(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/v4/competitions/PL/teams" {
			_, _ = w.Write([]byte(`{
				"competition": {"id": 2021, "name": "Premier League", "code": "PL"},
				"teams": [
					{"id": 57, "name": "Arsenal FC", "squad": []}
				]
			}`))
			return
		}
		if r.URL.Path == "/v4/teams/57" {
			_, _ = w.Write([]byte(`{
				"id": 57,
				"name": "Arsenal FC",
				"squad": [
					{"id": 7821, "name": "Bukayo Saka", "position": "Offence"}
				]
			}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	client := footballdata.NewClient("test-api-key", 0, footballdata.WithBaseURL(ts.URL))
	players, err := client.FetchLeaguePlayers(context.Background(), "PL")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(players) != 1 || players[0].Name != "Bukayo Saka" {
		t.Fatalf("unexpected players: %+v", players)
	}
}

func TestClient_FetchLeaguePlayers_RateLimitExceeded(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer ts.Close()

	client := footballdata.NewClient("test-api-key", 0, footballdata.WithBaseURL(ts.URL))
	_, err := client.FetchLeaguePlayers(context.Background(), "PL")
	if !errors.Is(err, footballdata.ErrRateLimitExceeded) {
		t.Fatalf("expected ErrRateLimitExceeded, got %v", err)
	}
}

func TestClient_FetchLeaguePlayers_MissingAPIKey(t *testing.T) {
	client := footballdata.NewClient("", 0)
	_, err := client.FetchLeaguePlayers(context.Background(), "PL")
	if !errors.Is(err, footballdata.ErrMissingAPIKey) {
		t.Fatalf("expected ErrMissingAPIKey, got %v", err)
	}
}
