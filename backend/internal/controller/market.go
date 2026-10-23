package controller

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/controller/dto"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/httphandler"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/middleware"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/service"
)

type MarketService interface {
	PlaceOrder(ctx context.Context, command service.PlaceOrderCommand) (service.PlacedOrder, error)
	CancelOrder(ctx context.Context, command service.CancelOrderCommand) (service.CancelledOrder, error)
	ListOpenOrders(ctx context.Context, userID int64, playerID *int64) ([]model.Order, error)
}

type PortfolioService interface {
	Portfolio(ctx context.Context, requester model.User, targetUserID int64) (model.Portfolio, error)
	Transactions(ctx context.Context, requester model.User, targetUserID int64, page, limit int) (model.PageResult[service.TransactionEntry], error)
}

type RestMarketController struct {
	marketService    MarketService
	portfolioService PortfolioService
}

func NewMarketController(marketService MarketService, portfolioService PortfolioService) *RestMarketController {
	return &RestMarketController{marketService: marketService, portfolioService: portfolioService}
}

func (c *RestMarketController) PlaceBuyOrder() httphandler.Endpoint {
	return c.placeOrder(model.SideBuy)
}

func (c *RestMarketController) PlaceSellOrder() httphandler.Endpoint {
	return c.placeOrder(model.SideSell)
}

func (c *RestMarketController) placeOrder(side model.OrderSide) httphandler.Endpoint {
	return func(w http.ResponseWriter, req *http.Request) error {
		actor, err := actorOf(req)
		if err != nil {
			return err
		}

		request, err := httphandler.Decode[dto.PlaceOrderRequest](req)
		if err != nil {
			return err
		}

		price, err := request.ParsedPrice()
		if err != nil {
			return httphandler.NewError(http.StatusBadRequest, err.Error(), err)
		}

		placed, err := c.marketService.PlaceOrder(req.Context(), service.PlaceOrderCommand{
			Price:       price,
			TimeInForce: request.ParsedTimeInForce(),
			Actor:       auditActorOf(actor),
			Side:        side,
			UserID:      actor.ID,
			PlayerID:    request.PlayerID,
			Quantity:    request.Quantity,
		})
		if err != nil {
			return marketStatusFor(err)
		}

		return httphandler.Encode(w, http.StatusCreated, dto.OrderResponseDesdeModelo(placed.Order, placed.Trades))
	}
}

func (c *RestMarketController) ListOpenOrders() httphandler.Endpoint {
	return func(w http.ResponseWriter, req *http.Request) error {
		actor, err := actorOf(req)
		if err != nil {
			return err
		}

		playerID, err := optionalPlayerFilter(req)
		if err != nil {
			return err
		}

		orders, err := c.marketService.ListOpenOrders(req.Context(), actor.ID, playerID)
		if err != nil {
			return marketStatusFor(err)
		}

		return httphandler.Encode(w, http.StatusOK, dto.OrderListResponseDesdeModelo(orders))
	}
}

func optionalPlayerFilter(req *http.Request) (*int64, error) {
	raw := req.URL.Query().Get("playerId")
	if raw == "" {
		return nil, nil
	}

	playerID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || playerID <= 0 {
		return nil, httphandler.NewError(http.StatusBadRequest, "playerId debe ser un entero positivo", errInvalidPlayerFilter)
	}

	return &playerID, nil
}

var errInvalidPlayerFilter = errors.New("el filtro playerId no es un entero positivo")

func (c *RestMarketController) CancelOrder() httphandler.Endpoint {
	return func(w http.ResponseWriter, req *http.Request) error {
		actor, err := actorOf(req)
		if err != nil {
			return err
		}

		orderID, err := identifierFromPath(req, "id")
		if err != nil {
			return err
		}

		cancelled, err := c.marketService.CancelOrder(req.Context(), service.CancelOrderCommand{
			Actor:   auditActorOf(actor),
			UserID:  actor.ID,
			OrderID: orderID,
		})
		if err != nil {
			return marketStatusFor(err)
		}

		return httphandler.Encode(w, http.StatusOK, dto.CancelOrderResponseDesdeModelo(cancelled))
	}
}

func (c *RestMarketController) Portfolio() httphandler.Endpoint {
	return func(w http.ResponseWriter, req *http.Request) error {
		actor, err := actorOf(req)
		if err != nil {
			return err
		}

		targetUserID, err := identifierFromPath(req, "id")
		if err != nil {
			return err
		}

		portfolio, err := c.portfolioService.Portfolio(req.Context(), requesterOf(actor), targetUserID)
		if err != nil {
			return marketStatusFor(err)
		}

		return httphandler.Encode(w, http.StatusOK, dto.PortfolioResponseDesdeModelo(portfolio))
	}
}

func (c *RestMarketController) Transactions() httphandler.Endpoint {
	return func(w http.ResponseWriter, req *http.Request) error {
		actor, err := actorOf(req)
		if err != nil {
			return err
		}

		targetUserID, err := identifierFromPath(req, "id")
		if err != nil {
			return err
		}

		page, limit, err := pagination(req)
		if err != nil {
			return err
		}

		entries, err := c.portfolioService.Transactions(req.Context(), requesterOf(actor), targetUserID, page, limit)
		if err != nil {
			return marketStatusFor(err)
		}

		return httphandler.Encode(w, http.StatusOK, dto.TransactionPageResponseDesdeModelo(entries))
	}
}

func requesterOf(actor middleware.Actor) model.User {
	return model.User{ID: actor.ID, Privilege: actor.Privilege, Active: true}
}

func auditActorOf(actor middleware.Actor) string {
	return "user/" + strconv.FormatInt(actor.ID, 10)
}

func identifierFromPath(req *http.Request, name string) (int64, error) {
	identifier, err := strconv.ParseInt(req.PathValue(name), 10, 64)
	if err != nil || identifier <= 0 {
		return 0, httphandler.NewError(http.StatusBadRequest, name+" debe ser un entero positivo", errInvalidIdentifier)
	}

	return identifier, nil
}

var errInvalidIdentifier = errors.New("el identificador de la ruta no es un entero positivo")

func pagination(req *http.Request) (int, int, error) {
	page, err := positiveQueryParam(req, "page", 1)
	if err != nil {
		return 0, 0, err
	}

	limit, err := positiveQueryParam(req, "limit", 20)
	if err != nil {
		return 0, 0, err
	}

	return page, limit, nil
}

func positiveQueryParam(req *http.Request, name string, fallback int) (int, error) {
	raw := req.URL.Query().Get(name)
	if raw == "" {
		return fallback, nil
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 0, httphandler.NewError(http.StatusBadRequest, name+" debe ser un entero positivo", errInvalidIdentifier)
	}

	return value, nil
}

func marketStatusFor(err error) error {
	switch {
	case errors.Is(err, service.ErrNotFound), errors.Is(err, service.ErrPlayerInactive):
		return httphandler.NewError(http.StatusNotFound, "player or order not found", err)
	case errors.Is(err, service.ErrForbidden), errors.Is(err, service.ErrNotOwner):
		return httphandler.NewError(http.StatusForbidden, "forbidden", err)
	case errors.Is(err, service.ErrNoLiquidity):
		return httphandler.NewError(http.StatusConflict, "insufficient liquidity", err)
	case errors.Is(err, model.ErrOrderNotLive):
		return httphandler.NewError(http.StatusConflict, "order is already final", err)
	case errors.Is(err, model.ErrInsufficientFunds):
		return httphandler.NewError(http.StatusUnprocessableEntity, "insufficient funds", err)
	case errors.Is(err, model.ErrInsufficientTokens), errors.Is(err, model.ErrPositionNotFound):
		return httphandler.NewError(http.StatusUnprocessableEntity, "insufficient token holding", err)
	case errors.Is(err, service.ErrInvalidQuantity), errors.Is(err, service.ErrInvalidPrice),
		errors.Is(err, service.ErrUnknownSide):
		return httphandler.NewError(http.StatusBadRequest, err.Error(), err)
	case errors.Is(err, model.ErrAccountNotFound):
		return httphandler.NewError(http.StatusNotFound, "market account not found", err)
	default:
		return err
	}
}
