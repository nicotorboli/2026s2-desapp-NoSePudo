package dto

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

type PlayerFilterDTO struct {
	Search          string
	League          string
	Club            string
	Position        string
	Page            int
	Limit           int
	IncludeInactive bool
}

func PlayerFilterDesdeQuery(r *http.Request) (PlayerFilterDTO, error) {
	q := r.URL.Query()

	dto := PlayerFilterDTO{
		Page:            1,
		Limit:           20,
		League:          strings.TrimSpace(q.Get("league")),
		Club:            strings.TrimSpace(q.Get("club")),
		Position:        strings.TrimSpace(q.Get("position")),
		Search:          strings.TrimSpace(q.Get("search")),
		IncludeInactive: false,
	}

	if pageStr := q.Get("page"); pageStr != "" {
		page, err := strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			return dto, fmt.Errorf("invalid query parameter 'page'")
		}
		dto.Page = page
	}

	if limitStr := q.Get("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 || limit > 100 {
			return dto, fmt.Errorf("invalid query parameter 'limit'")
		}
		dto.Limit = limit
	}

	if incStr := q.Get("includeInactive"); incStr != "" {
		inc, err := strconv.ParseBool(incStr)
		if err != nil {
			return dto, fmt.Errorf("invalid query parameter 'includeInactive'")
		}
		dto.IncludeInactive = inc
	}

	return dto, nil
}

func PlayerFilterAModelo(d PlayerFilterDTO) model.PlayerFilter {
	return model.PlayerFilter{
		Search:          d.Search,
		League:          d.League,
		Club:            d.Club,
		Position:        d.Position,
		Page:            d.Page,
		Limit:           d.Limit,
		IncludeInactive: d.IncludeInactive,
	}
}
