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

// PlayerFilterDesdeQuery parses and validates query parameters from an HTTP request.
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
			return dto, fmt.Errorf("%w: invalid query parameter 'page'", model.ErrInvalidInput)
		}
		dto.Page = page
	}

	if limitStr := q.Get("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 || limit > 100 {
			return dto, fmt.Errorf("%w: invalid query parameter 'limit'", model.ErrInvalidInput)
		}
		dto.Limit = limit
	}

	if incStr := q.Get("includeInactive"); incStr != "" {
		inc, err := strconv.ParseBool(incStr)
		if err != nil {
			return dto, fmt.Errorf("%w: invalid query parameter 'includeInactive'", model.ErrInvalidInput)
		}
		dto.IncludeInactive = inc
	}

	return dto, nil
}

// PlayerFilterAModelo maps DTO to domain PlayerFilter model.
func PlayerFilterAModelo(d PlayerFilterDTO) model.PlayerFilter {
	return model.PlayerFilter{
		League:          d.League,
		Club:            d.Club,
		Position:        d.Position,
		Search:          d.Search,
		IncludeInactive: d.IncludeInactive,
		Page:            d.Page,
		Limit:           d.Limit,
	}
}
