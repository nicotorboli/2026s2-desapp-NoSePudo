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

// AuthService es lo que el controller necesita del service.
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

// Register da de alta una cuenta.
//
// El controller hace tres cosas y ninguna más: decodifica, delega y traduce el
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
			// errors.Is y no una comparación pelada: el service decora el
			// error al subirlo, y el centinela sigue estando en la cadena.
			if errors.Is(err, model.ErrEmailTaken) {
				return httphandler.NewError(http.StatusConflict, "email already registered", err)
			}
			return err
		}

		return httphandler.Encode(w, http.StatusCreated, dto.AccountResponseDesdeModelo(user))
	}
}

// Login intercambia credenciales por una sesión.
//
// El 401 es uno solo y dice lo mismo para una contraseña incorrecta que para
// una cuenta inexistente: quién falló y por qué no es algo que se le cuente a
// quien no pudo entrar (FR-003).
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

// Refresh cambia la credencial de renovación por una sesión nueva.
//
// No toma cuerpo: los tres datos que el service necesita salen del actor que el
// middleware publicó, es decir de una credencial que ya verificó. Un valor que
// el cliente manda es un valor que hay que validar, testear y desconfiar, y la
// forma más barata de tratarlo es no aceptarlo.
func (c *RestAuthController) Refresh() httphandler.Endpoint {
	return func(w http.ResponseWriter, req *http.Request) error {
		actor, err := actorOf(req)
		if err != nil {
			return err
		}

		session, err := c.authService.Refresh(req.Context(), actor.CredentialID, actor.ID, actor.SessionID)
		if err != nil {
			// El reuso y la expiración se responden igual, con 401: al que
			// presentó una credencial que no sirve no se le explica por qué.
			// Que el reuso además haya cortado toda la cuenta es una decisión
			// interna y no algo que se le cuente.
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

// Logout cierra la sesión que nombra la credencial de acceso presentada.
//
// Tampoco toma cuerpo, y por eso no hay comparación de propiedad que hacer ni
// un 403 que testear: la familia sale de un claim verificado y no puede
// pertenecer a otro.
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
