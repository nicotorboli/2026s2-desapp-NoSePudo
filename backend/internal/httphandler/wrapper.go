package httphandler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/logger"
)

// APIError is an abstraction that controllers use to pass errors up to the wrapper
type APIError interface {
	error
	StatusCode() int
	Message() string
}

type Endpoint func(w http.ResponseWriter, req *http.Request) error

func Wrap(endpoint Endpoint, log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		// El logger se publica en el contexto acá, que es el decorador más
		// externo de la cadena. Sin esto, las capas de abajo que logueen con
		// logger.FromContext caerían en el logger por defecto en vez del que se
		// le inyectó al servidor, y lo que escriban no tendría con qué
		// decorarse ni adónde ir.
		req = req.WithContext(logger.Into(req.Context(), log))
		ctx := req.Context()

		err := endpoint(w, req)
		if err == nil {
			return
		}

		// Se recorre la cadena de %w en vez de mirar sólo el error de
		// arriba: cualquier capa intermedia que decore con
		// fmt.Errorf("...: %w", err) convertiría un 401 intencional en un
		// 500 si se usara una type assertion pelada.
		if apiErr, ok := errors.AsType[APIError](err); ok {
			log.WarnContext(ctx, "Handled HTTP error", "status", apiErr.StatusCode(), "error", err.Error())
			if encodeErr := Encode(w, apiErr.StatusCode(), map[string]string{"error": apiErr.Message()}); encodeErr != nil {
				log.ErrorContext(ctx, "Error encoding API error", "error", encodeErr.Error())
			}
			return
		}

		log.ErrorContext(ctx, "Internal server error", "error", err.Error())
		if encodeErr := Encode(w, http.StatusInternalServerError, map[string]string{"error": "Internal Server Error"}); encodeErr != nil {
			log.ErrorContext(ctx, "Error encoding internal server error", "error", encodeErr.Error())
		}
	}
}
