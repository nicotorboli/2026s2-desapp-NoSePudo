package controller

import (
	"context"
	"errors"
	"net/http"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/controller/dto"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/httphandler"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/middleware"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

type AuthService interface {
	Register(ctx context.Context, email, password string) (model.User, error)
	Login(ctx context.Context, email, password string) (model.Session, error)
	Refresh(ctx context.Context, presentedID string, userID int64, sessionID string) (model.Session, error)
	Logout(ctx context.Context, userID int64, sessionID string) error
}

type RestAuthController struct {
	authService AuthService
}

func NewAuthController(authService AuthService) *RestAuthController {
	return &RestAuthController{authService: authService}
}

// El controller hace tres cosas: decodifica, delega y traduce el
// error de dominio a un status. La validación la corre Decode y las reglas las
// tiene el service, que no sabe nada de HTTP.
func (c *RestAuthController) Register() httphandler.Endpoint {
	return func(w http.ResponseWriter, req *http.Request) error {
		// Decode ya devuelve un error de borde con 400 cuando el cuerpo trae
		// un campo inesperado, no es JSON válido o no pasa el Validate del DTO.
		request, err := httphandler.Decode[dto.RegisterRequest](req)
		if err != nil {
			return err
		}

		user, err := c.authService.Register(req.Context(), request.Email, request.Password)
		if err != nil {
			if errors.Is(err, model.ErrEmailTaken) {
				return httphandler.NewError(http.StatusConflict, "email already registered", err)
			}
			return err
		}

		return httphandler.Encode(w, http.StatusCreated, dto.AccountResponseDesdeModelo(user))
	}
}

func (c *RestAuthController) Login() httphandler.Endpoint {
	return func(w http.ResponseWriter, req *http.Request) error {
		request, err := httphandler.Decode[dto.LoginRequest](req)
		if err != nil {
			return err
		}

		session, err := c.authService.Login(req.Context(), request.Email, request.Password)
		if err != nil {
			if errors.Is(err, model.ErrInvalidCredentials) {
				return httphandler.NewError(http.StatusUnauthorized, "invalid credentials", err)
			}
			return err
		}

		return httphandler.Encode(w, http.StatusOK, dto.SessionResponseDesdeModelo(session))
	}
}

func (c *RestAuthController) Refresh() httphandler.Endpoint {
	return func(w http.ResponseWriter, req *http.Request) error {
		actor, err := actorOf(req)
		if err != nil {
			return err
		}

		session, err := c.authService.Refresh(req.Context(), actor.CredentialID, actor.ID, actor.SessionID)
		if err != nil {
			if errors.Is(err, model.ErrRefreshTokenReused) ||
				errors.Is(err, model.ErrRefreshTokenExpired) ||
				errors.Is(err, model.ErrRefreshTokenRevoked) {
				return httphandler.NewError(http.StatusUnauthorized, "authentication required", err)
			}
			return err
		}

		return httphandler.Encode(w, http.StatusOK, dto.SessionResponseDesdeModelo(session))
	}
}

func (c *RestAuthController) Logout() httphandler.Endpoint {
	return func(w http.ResponseWriter, req *http.Request) error {
		actor, err := actorOf(req)
		if err != nil {
			return err
		}

		if err := c.authService.Logout(req.Context(), actor.ID, actor.SessionID); err != nil {
			return err
		}

		w.WriteHeader(http.StatusNoContent)

		return nil
	}
}

// actorOf saca el actor del contexto. Que falte significa que la ruta se
// registró sin autenticación delante, y eso se responde 401 en vez de seguir
// con una cuenta cero.
func actorOf(req *http.Request) (middleware.Actor, error) {
	actor, authenticated := middleware.ActorFromContext(req.Context())
	if !authenticated {
		return middleware.Actor{}, httphandler.NewError(
			http.StatusUnauthorized,
			"authentication required",
			errMissingActor,
		)
	}

	return actor, nil
}

var errMissingActor = errors.New("la operación necesita un actor y la petición no fue autenticada")
