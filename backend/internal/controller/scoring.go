package controller

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/dto"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/httphandler"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/service"
)

type ScoringService interface {
	ListStrategies() []dto.ScoringStrategyResponse
	ActiveStrategy(ctx context.Context) (dto.ActiveStrategyResponse, bool, error)
	ActivateStrategy(ctx context.Context, strategyID int, actorID int64) error
	SetLeagueCoefficient(ctx context.Context, leagueCode string, coefficient float64, actorID int64) error
	GetLeagueCoefficient(ctx context.Context, leagueCode string) (dto.LeagueCoefficientResponse, bool, error)
	GetScore(ctx context.Context, playerID int64) (dto.PlayerScoreResponse, error)
	GetScoreHistory(ctx context.Context, playerID int64) ([]dto.PlayerScoreResponse, error)
	ListRanking(ctx context.Context, filter dto.PlayerFilterDTO, sortOrder string, offset int) (dto.PlayerRankingPageResponse, error)
}

type RestScoringController struct {
	svc ScoringService
}

func NewScoringController(svc ScoringService) *RestScoringController {
	return &RestScoringController{svc: svc}
}

func (c *RestScoringController) ListStrategies() httphandler.Endpoint {
	return func(w http.ResponseWriter, req *http.Request) error {
		return httphandler.Encode(w, http.StatusOK, c.svc.ListStrategies())
	}
}

func (c *RestScoringController) GetActiveStrategy() httphandler.Endpoint {
	return func(w http.ResponseWriter, req *http.Request) error {
		response, found, err := c.svc.ActiveStrategy(req.Context())
		if err != nil {
			return err
		}
		if !found {
			return httphandler.NewError(http.StatusNotFound, "no active strategy", nil)
		}
		return httphandler.Encode(w, http.StatusOK, response)
	}
}

func (c *RestScoringController) ActivateStrategy() httphandler.Endpoint {
	return func(w http.ResponseWriter, req *http.Request) error {
		actor, err := actorOf(req)
		if err != nil {
			return err
		}

		request, err := httphandler.Decode[dto.ActivateStrategyRequest](req)
		if err != nil {
			return err
		}

		if err = c.svc.ActivateStrategy(req.Context(), request.ID, actor.ID); err != nil {
			return err
		}

		response, found, err := c.svc.ActiveStrategy(req.Context())
		if err != nil {
			return err
		}
		if !found {
			return httphandler.NewError(http.StatusInternalServerError, "strategy not active after activation", nil)
		}

		return httphandler.Encode(w, http.StatusOK, response)
	}
}

func (c *RestScoringController) GetLeagueCoefficient() httphandler.Endpoint {
	return func(w http.ResponseWriter, req *http.Request) error {
		leagueCode := req.PathValue("league")
		if leagueCode == "" {
			return httphandler.NewError(http.StatusBadRequest, "missing league code", nil)
		}

		response, found, err := c.svc.GetLeagueCoefficient(req.Context(), leagueCode)
		if err != nil {
			return err
		}
		if !found {
			return httphandler.NewError(http.StatusNotFound, "league coefficient not set", nil)
		}

		return httphandler.Encode(w, http.StatusOK, response)
	}
}

func (c *RestScoringController) SetLeagueCoefficient() httphandler.Endpoint {
	return func(w http.ResponseWriter, req *http.Request) error {
		actor, err := actorOf(req)
		if err != nil {
			return err
		}

		leagueCode := req.PathValue("league")
		if leagueCode == "" {
			return httphandler.NewError(http.StatusBadRequest, "missing league code", nil)
		}

		request, err := httphandler.Decode[dto.SetLeagueCoefficientRequest](req)
		if err != nil {
			return err
		}

		if err = c.svc.SetLeagueCoefficient(req.Context(), leagueCode, request.Coefficient, actor.ID); err != nil {
			return err
		}

		response, found, err := c.svc.GetLeagueCoefficient(req.Context(), leagueCode)
		if err != nil {
			return err
		}
		if !found {
			return httphandler.NewError(http.StatusInternalServerError, "league coefficient not found after set", nil)
		}

		return httphandler.Encode(w, http.StatusOK, response)
	}
}

func (c *RestScoringController) GetScore() httphandler.Endpoint {
	return func(w http.ResponseWriter, req *http.Request) error {
		playerID, err := scoringPlayerID(req)
		if err != nil {
			return err
		}

		score, err := c.svc.GetScore(req.Context(), playerID)
		if err != nil {
			if errors.Is(err, service.ErrNotFound) || errors.Is(err, service.ErrScoreNotFound) {
				return httphandler.NewError(http.StatusNotFound, "player score not found", err)
			}
			return err
		}

		return httphandler.Encode(w, http.StatusOK, score)
	}
}

func (c *RestScoringController) GetScoreHistory() httphandler.Endpoint {
	return func(w http.ResponseWriter, req *http.Request) error {
		playerID, err := scoringPlayerID(req)
		if err != nil {
			return err
		}

		scores, err := c.svc.GetScoreHistory(req.Context(), playerID)
		if err != nil {
			if errors.Is(err, service.ErrNotFound) {
				return httphandler.NewError(http.StatusNotFound, "player not found", err)
			}
			return err
		}

		return httphandler.Encode(w, http.StatusOK, scores)
	}
}

func (c *RestScoringController) GetPlayersRanking() httphandler.Endpoint {
	return func(w http.ResponseWriter, req *http.Request) error {
		filter, err := dto.PlayerFilterDesdeQuery(req)
		if err != nil {
			return httphandler.NewError(http.StatusBadRequest, err.Error(), nil)
		}

		sortOrder := req.URL.Query().Get("sort")
		if sortOrder == "" {
			sortOrder = "score_desc"
		}
		switch sortOrder {
		case "name", "score_asc", "score_desc":
		default:
			return httphandler.NewError(http.StatusBadRequest, "invalid query parameter 'sort'", nil)
		}

		offset := 0
		if offsetStr := req.URL.Query().Get("offset"); offsetStr != "" {
			parsed, parseErr := strconv.Atoi(offsetStr)
			if parseErr != nil || parsed < 0 {
				return httphandler.NewError(http.StatusBadRequest, "invalid query parameter 'offset'", nil)
			}
			offset = parsed
		}

		ranking, err := c.svc.ListRanking(req.Context(), filter, sortOrder, offset)
		if err != nil {
			return err
		}

		return httphandler.Encode(w, http.StatusOK, ranking)
	}
}

func scoringPlayerID(req *http.Request) (int64, error) {
	idParam := req.PathValue("id")
	if idParam == "" {
		return 0, httphandler.NewError(http.StatusBadRequest, "missing player id", nil)
	}

	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil || id <= 0 {
		return 0, httphandler.NewError(http.StatusBadRequest, "invalid player ID format", err)
	}

	return id, nil
}
