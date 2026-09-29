package middleware

import (
	"errors"
	"net/http"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/httphandler"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/logger"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

const insufficientPrivilegeMessage = "insufficient privilege"

var (
	errNoActor               = errors.New("la operación exige privilegio pero la petición no fue autenticada")
	errInsufficientPrivilege = errors.New("el actor no tiene el privilegio que la operación exige")
)

type Authorization struct{}

func NewAuthorization() *Authorization {
	return &Authorization{}
}

func (a *Authorization) Require(required model.PrivilegeLevel) Decorator {
	return func(next httphandler.Endpoint) httphandler.Endpoint {
		return func(w http.ResponseWriter, req *http.Request) error {
			ctx := req.Context()

			actor, authenticated := ActorFromContext(ctx)
			if !authenticated {
				logger.FromContext(ctx).Warn(
					"authorization refused",
					"operation", "authorize",
					"reason", errNoActor.Error(),
				)

				return httphandler.NewError(http.StatusUnauthorized, unauthenticatedMessage, errNoActor)
			}

			if !actor.Privilege.Satisfies(required) {
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
