package httphandler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

type APIError interface {
	error
	StatusCode() int
	ErrorType() string
	Message() string
}

type HTTPError struct {
	Type    string `json:"error"`
	Msg     string `json:"message"`
	Code    int    `json:"-"`
}

func (e *HTTPError) Error() string {
	return e.Msg
}

func (e *HTTPError) StatusCode() int {
	return e.Code
}

func (e *HTTPError) ErrorType() string {
	return e.Type
}

func (e *HTTPError) Message() string {
	return e.Msg
}

func NewHTTPError(statusCode int, errorType, message string) *HTTPError {
	return &HTTPError{
		Code: statusCode,
		Type: errorType,
		Msg:  message,
	}
}

type Endpoint func(w http.ResponseWriter, req *http.Request) error

type ErrorResponseBody struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// Wrap wraps an endpoint returning an error and handles errors uniformly.
func Wrap(endpoint Endpoint, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		err := endpoint(w, req)
		if err == nil {
			return
		}

		ctx := req.Context()

		var apiErr APIError
		if errors.As(err, &apiErr) {
			logger.WarnContext(ctx, "Handled HTTP error",
				"status", apiErr.StatusCode(),
				"error", apiErr.ErrorType(),
				"message", apiErr.Message(),
			)
			_ = Encode(w, apiErr.StatusCode(), ErrorResponseBody{
				Error:   apiErr.ErrorType(),
				Message: apiErr.Message(),
			})
			return
		}

		if errors.Is(err, model.ErrNotFound) {
			logger.WarnContext(ctx, "Resource not found", "error", err.Error())
			_ = Encode(w, http.StatusNotFound, ErrorResponseBody{
				Error:   "Not Found",
				Message: "The requested resource was not found",
			})
			return
		}

		if errors.Is(err, model.ErrInvalidInput) {
			logger.WarnContext(ctx, "Invalid input", "error", err.Error())
			_ = Encode(w, http.StatusBadRequest, ErrorResponseBody{
				Error:   "Bad Request",
				Message: err.Error(),
			})
			return
		}

		if errors.Is(err, model.ErrRateLimitExceeded) {
			logger.WarnContext(ctx, "Rate limit exceeded", "error", err.Error())
			_ = Encode(w, http.StatusTooManyRequests, ErrorResponseBody{
				Error:   "Rate Limit Exceeded",
				Message: "External provider rate limit hit; existing data is intact",
			})
			return
		}

		logger.ErrorContext(ctx, "Internal server error", "error", err.Error())
		_ = Encode(w, http.StatusInternalServerError, ErrorResponseBody{
			Error:   "Internal Server Error",
			Message: "An internal server error occurred",
		})
	}
}
