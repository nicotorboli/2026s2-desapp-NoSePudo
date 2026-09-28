package middleware

import (
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

// unauthenticatedMessage es lo único que se le dice a quien no pudo entrar.
// No distingue una credencial ausente de una expirada, alterada o del tipo
// equivocado: por qué falló no es asunto de quien la presentó.
const unauthenticatedMessage = "authentication required"

// TokenVerifier es lo que el middleware necesita para decidir. La interfaz se
// declara acá, del lado de quien la consume, así que los casos de prueba
// pueden devolver credenciales arbitrarias sin firmar nada.
type TokenVerifier interface {
	Verify(raw string) (adapters.Claims, error)
}

// Decorator es la forma que toman la autenticación y la autorización: envuelven
// un Endpoint y devuelven otro.
//
// Son decoradores de Endpoint y no de http.Handler porque un rechazo de
// autenticación es un error, y el código ya tiene una sola manera de convertir
// un error en respuesta: devolverlo y dejar que httphandler.Wrap lo codifique.
// Así el rechazo sale con la misma forma JSON que cualquier otro fallo, sin un
// segundo camino de error que mantener sincronizado.
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

// RequireRefreshToken protege la renovación, donde la credencial de refresco
// es la credencial y un token de acceso no sirve (FR-038).
func (a *Authentication) RequireRefreshToken() Decorator {
	return a.require(adapters.KindRefresh)
}

// require verifica la credencial antes de que corra cualquier lógica de
// negocio y antes de tocar la persistencia (FR-008), y publica el actor para
// las capas de abajo.
func (a *Authentication) require(kind adapters.TokenKind) Decorator {
	return func(next httphandler.Endpoint) httphandler.Endpoint {
		return func(w http.ResponseWriter, req *http.Request) error {
			raw, err := bearerToken(req.Header.Get("Authorization"))
			if err != nil {
				return unauthenticated(err)
			}

			claims, err := a.tokenVerifier.Verify(raw)
			if err != nil {
				return unauthenticated(err)
			}

			// El tipo se comprueba después de verificar la firma: hasta ese
			// momento nada de lo que dice la credencial es digno de confianza.
			if claims.Kind != kind {
				return unauthenticated(errWrongTokenKind)
			}

			actor := Actor{
				SessionID:    claims.SessionID,
				CredentialID: claims.ID,
				ID:           claims.Subject,
				Privilege:    claims.Privilege,
			}

			ctx := WithActor(req.Context(), actor)
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

// bearerToken exige exactamente "Bearer <token>". Una cabecera ausente, vacía,
// sin esquema, con un esquema desconocido o con algo más detrás del token se
// rechaza en vez de intentar interpretarla.
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

	// Ni un token vacío ni uno seguido de cualquier otra cosa.
	token = strings.TrimSpace(token)
	if token == "" || strings.ContainsAny(token, " \t") {
		return "", errMalformedHeader
	}

	return token, nil
}

// unauthenticated envuelve la causa para que quede en el log, y le responde al
// cliente siempre lo mismo. Es un 401 y no un 403: "no estás autenticado" es
// distinto de "estás autenticado pero no te alcanza" (FR-011).
func unauthenticated(cause error) error {
	return httphandler.NewError(http.StatusUnauthorized, unauthenticatedMessage, cause)
}
