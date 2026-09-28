package httphandler

import (
	"errors"
	"log/slog"
	"net/http"
)

// APIError is an abstraction that controllers use to pass errors up to the wrapper
type APIError interface {
	error
	StatusCode() int
	Message() string
}

type Endpoint func(w http.ResponseWriter, req *http.Request) error

func Wrap(endpoint Endpoint, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		err := endpoint(w, req)
		if err != nil {
			logger.Error("Endpoint error", "error", err.Error())

			// Se recorre la cadena de %w en vez de mirar sólo el error de
			// arriba: cualquier capa intermedia que decore con
			// fmt.Errorf("...: %w", err) convertiría un 401 intencional en un
			// 500 si se usara una type assertion pelada.
			if apiErr, ok := errors.AsType[APIError](err); ok {
				if encodeErr := Encode(w, apiErr.StatusCode(), map[string]string{"error": apiErr.Message()}); encodeErr != nil {
					logger.Error("Error encoding API error", "error", encodeErr.Error())
				}
				return
			}

			if encodeErr := Encode(w, http.StatusInternalServerError, map[string]string{"error": "Internal Server Error"}); encodeErr != nil {
				logger.Error("Error encoding internal server error", "error", encodeErr.Error())
			}
		}
	}
}
