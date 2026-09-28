package controller

import (
	"context"
	"errors"
	"net/http"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/controller/dto"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/httphandler"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

// AuthService es lo que el controller necesita del service.
type AuthService interface {
	Register(ctx context.Context, email, password string) (model.User, error)
	Login(ctx context.Context, email, password string) (model.Session, error)
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
