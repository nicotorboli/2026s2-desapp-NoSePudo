package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/adapters"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/httphandler"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/logger"
)

// bearerScheme es el único esquema que se acepta. La comparación es
// insensible a mayúsculas porque el RFC 7235 define el esquema así.
const bearerScheme = "bearer"

const unauthenticatedMessage = "authentication required"

type TokenVerifier interface {
	Verify(raw string) (adapters.Claims, error)
}

// Decorator es la forma que toman la autenticación y la autorización: envuelven
// un Endpoint y devuelven otro.

type Decorator func(httphandler.Endpoint) httphandler.Endpoint

type Authentication struct {
	tokenVerifier TokenVerifier
}

func NewAuthentication(tokenVerifier TokenVerifier) *Authentication {
	return &Authentication{tokenVerifier: tokenVerifier}
}

// RequireAccessToken protege una operación sobre un recurso.
func (a *Authentication) RequireAccessToken() Decorator {
	return a.require(adapters.KindAccess)
}

// RequireRefreshToken protege la renovación
func (a *Authentication) RequireRefreshToken() Decorator {
	return a.require(adapters.KindRefresh)
}

// require verifica la credencial antes de que corra cualquier lógica de
// negocio y antes de tocar la persistencia, y publica el actor para
// las capas de abajo.
func (a *Authentication) require(kind adapters.TokenKind) Decorator {
	return func(next httphandler.Endpoint) httphandler.Endpoint {
		return func(w http.ResponseWriter, req *http.Request) error {
			ctx := req.Context()

			raw, err := bearerToken(req.Header.Get("Authorization"))
			if err != nil {
				return a.refuse(ctx, err)
			}

			claims, err := a.tokenVerifier.Verify(raw)
			if err != nil {
				return a.refuse(ctx, err)
			}

			// El tipo se comprueba después de verificar la firma:
			if claims.Kind != kind {
				return a.refuse(ctx, errWrongTokenKind)
			}

			actor := Actor{
				SessionID:    claims.SessionID,
				CredentialID: claims.ID,
				ID:           claims.Subject,
				Privilege:    claims.Privilege,
			}

			ctx = WithActor(ctx, actor)
			ctx = logger.Into(ctx, logger.FromContext(ctx).With("actor", actor.ID))

			return next(w, req.WithContext(ctx))
		}
	}
}

var (
	errMissingCredential = errors.New("falta la cabecera Authorization")
	errMalformedHeader   = errors.New("la cabecera Authorization no tiene la forma Bearer <token>")
	errWrongTokenKind    = errors.New("la credencial no es del tipo que este endpoint requiere")
)

func bearerToken(header string) (string, error) {
	if strings.TrimSpace(header) == "" {
		return "", errMissingCredential
	}

	scheme, token, found := strings.Cut(header, " ")
	if !found {
		return "", errMalformedHeader
	}
	if !strings.EqualFold(scheme, bearerScheme) {
		return "", errMalformedHeader
	}

	token = strings.TrimSpace(token)
	if token == "" || strings.ContainsAny(token, " \t") {
		return "", errMalformedHeader
	}

	return token, nil
}

// refuse registra el rechazo y lo devuelve. Toda autenticación refutada queda logueada con su razón, y
// que se loguee incluso cuando no se pudo identificar ninguna cuenta

func (a *Authentication) refuse(ctx context.Context, cause error) error {
	logger.FromContext(ctx).Warn(
		"authentication refused",
		"operation", "authenticate",
		"reason", cause.Error(),
	)

	return unauthenticated(cause)
}

// unauthenticated envuelve la causa para que quede en el log, y le responde al
// cliente siempre lo mismo.
func unauthenticated(cause error) error {
	return httphandler.NewError(http.StatusUnauthorized, unauthenticatedMessage, cause)
}
