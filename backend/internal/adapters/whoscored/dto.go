package whoscored

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

type statisticsFeed struct {
	PlayerTableStats []playerTableStat `json:"playerTableStats"`
	Paging           paging            `json:"paging"`
}

type paging struct {
	Page        int `json:"page"`
	NumberPages int `json:"numberPages"`
}

type playerTableStat struct {
	Name     string          `json:"name"`
	Rating   any             `json:"rating"`
	Stats    map[string]stat `json:"stats"`
	TeamName string          `json:"teamName"`
}

type stat struct {
	Value any `json:"value"`
}

type feedPage struct {
	rows       []model.PlayerStatisticsRow
	totalPages int
}

func parseFeedPage(body []byte) (feedPage, error) {
	var feed statisticsFeed
	if err := json.Unmarshal(body, &feed); err != nil {
		return feedPage{}, fmt.Errorf("unmarshal statistics feed: %w", err)
	}

	rows := make([]model.PlayerStatisticsRow, 0, len(feed.PlayerTableStats))
	for _, p := range feed.PlayerTableStats {
		rating, _ := statNumber(p.Rating)
		if r, ok := firstStat(p.Stats, "rating"); ok {
			rating = r
		}
		rows = append(rows, model.PlayerStatisticsRow{
			Name:      strings.TrimSpace(p.Name),
			ClubName:  strings.TrimSpace(p.TeamName),
			Goals:     statInt(p.Stats, "goals"),
			Assists:   statInt(p.Stats, "assists"),
			Shots:     statInt(p.Stats, "totalShots", "shots"),
			KeyPasses: statInt(p.Stats, "keyPasses", "totalKeyPasses"),
			Dribbles:  statInt(p.Stats, "dribbles", "totalSuccessfulDribbles", "successfulDribbles"),
			Tackles:   statInt(p.Stats, "tackles", "totalTackles"),
			Rating:    rating,
		})
	}

	return feedPage{rows: rows, totalPages: feed.Paging.NumberPages}, nil
}

func firstStat(stats map[string]stat, keys ...string) (float64, bool) {
	for _, key := range keys {
		if s, ok := stats[key]; ok {
			if v, ok := statNumber(s.Value); ok {
				return v, true
			}
		}
	}
	return 0, false
}

func statInt(stats map[string]stat, keys ...string) int {
	v, ok := firstStat(stats, keys...)
	if !ok {
		return 0
	}
	return int(v + 0.5)
}

func statNumber(raw any) (float64, bool) {
	switch v := raw.(type) {
	case float64:
		return v, true
	case json.Number:
		f, err := v.Float64()
		if err != nil {
			return 0, false
		}
		return f, true
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err != nil {
			return 0, false
		}
		return f, true
	default:
		return 0, false
	}
}
