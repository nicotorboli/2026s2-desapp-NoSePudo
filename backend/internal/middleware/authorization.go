package middleware

import (
	"errors"
	"net/http"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/httphandler"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/logger"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

// insufficientPrivilegeMessage es lo que se responde cuando el actor está
// identificado pero su nivel no alcanza. Es deliberadamente distinto del
// mensaje de autenticación: quien llama tiene que poder distinguir "no estás
// autenticado" de "estás autenticado pero no te alcanza" (FR-011).
const insufficientPrivilegeMessage = "insufficient privilege"

var (
	errNoActor               = errors.New("la operación exige privilegio pero la petición no fue autenticada")
	errInsufficientPrivilege = errors.New("el actor no tiene el privilegio que la operación exige")
)

type Authorization struct{}

func NewAuthorization() *Authorization {
	return &Authorization{}
}

// Require exige un nivel de privilegio mínimo.
//
// Es un punto distinto del de autenticación, y por eso el 403 y el 401 salen de
// dos lugares y no de dos ramas de un mismo chequeo: son dos preguntas
// diferentes y se responden por separado.
func (a *Authorization) Require(required model.PrivilegeLevel) Decorator {
	return func(next httphandler.Endpoint) httphandler.Endpoint {
		return func(w http.ResponseWriter, req *http.Request) error {
			ctx := req.Context()

			actor, authenticated := ActorFromContext(ctx)
			if !authenticated {
				// Llegar acá significa que la cadena se armó sin autenticación
				// delante. Se responde 401 y no 403 porque es literalmente
				// cierto —no hay actor— y porque es el que menos cuenta.
				logger.FromContext(ctx).Warn(
					"authorization refused",
					"operation", "authorize",
					"reason", errNoActor.Error(),
				)

				return httphandler.NewError(http.StatusUnauthorized, unauthenticatedMessage, errNoActor)
			}

			// Satisfies es falso en cuanto alguno de los dos lados es
			// PrivilegeUnknown, así que un claim ausente o irreconocible cae
			// acá y nunca pasa por superusuario (FR-018).
			if !actor.Privilege.Satisfies(required) {
				// Acá sí hay sujeto que nombrar, y el actor es su identidad de
				// cuenta: nunca su email, que es dato personal (FR-028).
				logger.FromContext(ctx).Warn(
					"authorization refused",
					"operation", "authorize",
					"actor", actor.ID,
					"reason", errInsufficientPrivilege.Error(),
					"required", required.String(),
					"held", actor.Privilege.String(),
				)

				return httphandler.NewError(http.StatusForbidden, insufficientPrivilegeMessage, errInsufficientPrivilege)
			}

			return next(w, req)
		}
	}
}
