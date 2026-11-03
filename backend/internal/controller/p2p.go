package controller

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/controller/dto"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/httphandler"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/service"
)

type P2PService interface {
	Create(ctx context.Context, cmd service.CreateP2PCommand) (model.P2PSession, model.Player, error)
	Get(ctx context.Context, token string) (model.P2PSession, model.Player, error)
	Accept(ctx context.Context, token string, buyerID int64, actor string) (model.Trade, error)
	Cancel(ctx context.Context, token string, actorID int64, actor string) error
}

type RestP2PController struct {
	p2pService P2PService
}

func NewP2PController(p2pService P2PService) *RestP2PController {
	return &RestP2PController{p2pService: p2pService}
}

func (c *RestP2PController) Create() httphandler.Endpoint {
	return func(w http.ResponseWriter, req *http.Request) error {
		actor, err := actorOf(req)
		if err != nil {
			return err
		}

		request, err := httphandler.Decode[dto.CreateP2PSessionRequest](req)
		if err != nil {
			return err
		}

		if err = request.Validate(); err != nil {
			return httphandler.NewError(http.StatusBadRequest, err.Error(), err)
		}

		created, player, err := c.p2pService.Create(req.Context(), service.CreateP2PCommand{
			Actor:    auditActorOf(actor),
			SellerID: actor.ID,
			PlayerID: request.PlayerID,
			Quantity: request.Quantity,
			Price:    model.Credits(request.PricePerToken),
		})
		if err != nil {
			return p2pStatusFor(err)
		}

		sessionURL := fmt.Sprintf("/?token=%s", created.Token)

		return httphandler.Encode(w, http.StatusCreated, dto.P2PSessionResponseDesdeModelo(created, player, sessionURL))
	}
}

func (c *RestP2PController) Get() httphandler.Endpoint {
	return func(w http.ResponseWriter, req *http.Request) error {
		token := req.PathValue("token")
		if token == "" {
			return httphandler.NewError(http.StatusBadRequest, "token es obligatorio", errors.New("token faltante"))
		}

		session, player, err := c.p2pService.Get(req.Context(), token)
		if err != nil {
			return p2pStatusFor(err)
		}

		sessionURL := fmt.Sprintf("/?token=%s", session.Token)
		return httphandler.Encode(w, http.StatusOK, dto.P2PSessionResponseDesdeModelo(session, player, sessionURL))
	}
}

func (c *RestP2PController) Accept() httphandler.Endpoint {
	return func(w http.ResponseWriter, req *http.Request) error {
		actor, err := actorOf(req)
		if err != nil {
			return err
		}

		token := req.PathValue("token")
		if token == "" {
			return httphandler.NewError(http.StatusBadRequest, "token es obligatorio", errors.New("token faltante"))
		}

		trade, err := c.p2pService.Accept(req.Context(), token, actor.ID, auditActorOf(actor))
		if err != nil {
			return p2pStatusFor(err)
		}

		return httphandler.Encode(w, http.StatusOK, dto.P2PTradeResponseDesdeModelo(trade, token))
	}
}

func (c *RestP2PController) Cancel() httphandler.Endpoint {
	return func(w http.ResponseWriter, req *http.Request) error {
		actor, err := actorOf(req)
		if err != nil {
			return err
		}

		token := req.PathValue("token")
		if token == "" {
			return httphandler.NewError(http.StatusBadRequest, "token es obligatorio", errors.New("token faltante"))
		}

		if err := c.p2pService.Cancel(req.Context(), token, actor.ID, auditActorOf(actor)); err != nil {
			return p2pStatusFor(err)
		}

		w.WriteHeader(http.StatusNoContent)
		return nil
	}
}

func p2pStatusFor(err error) error {
	switch {
	case errors.Is(err, service.ErrNotFound), errors.Is(err, service.ErrPlayerInactive),
		errors.Is(err, model.ErrP2PSessionNotFound):
		return httphandler.NewError(http.StatusNotFound, "sesión o jugador no encontrado", err)
	case errors.Is(err, service.ErrP2PForbidden):
		return httphandler.NewError(http.StatusForbidden, "sólo el vendedor puede cancelar su propia oferta", err)
	case errors.Is(err, model.ErrP2PSessionSelfTrade), errors.Is(err, service.ErrP2PSelfTrade):
		return httphandler.NewError(http.StatusConflict, "el vendedor no puede aceptar su propia oferta", err)
	case errors.Is(err, model.ErrP2PSessionAlreadyTaken), errors.Is(err, service.ErrP2PAlreadyTaken):
		return httphandler.NewError(http.StatusConflict, "la oferta ya fue tomada por otro usuario", err)
	case errors.Is(err, model.ErrP2PSessionNotLive), errors.Is(err, service.ErrP2PNotLive):
		return httphandler.NewError(http.StatusConflict, "la sesión P2P ya no está activa", err)
	case errors.Is(err, model.ErrP2PSessionExpired), errors.Is(err, service.ErrP2PExpired):
		return httphandler.NewError(http.StatusGone, "la oferta venció y ya no puede aceptarse", err)
	case errors.Is(err, model.ErrInsufficientFunds):
		return httphandler.NewError(http.StatusUnprocessableEntity, "el saldo disponible no alcanza", err)
	case errors.Is(err, model.ErrInsufficientTokens), errors.Is(err, model.ErrPositionNotFound):
		return httphandler.NewError(http.StatusUnprocessableEntity, "la posición disponible no alcanza", err)
	case errors.Is(err, service.ErrInvalidQuantity), errors.Is(err, service.ErrInvalidPrice):
		return httphandler.NewError(http.StatusBadRequest, err.Error(), err)
	default:
		return err
	}
}
