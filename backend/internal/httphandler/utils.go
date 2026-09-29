package httphandler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const maxRequestBodySize = 1048576 // 1MB

// Encode sets the Content-Type header to application/json, writes the HTTP status code, and encodes the data.
func Encode[T any](w http.ResponseWriter, status int, data T) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}
	return nil
}

// Decode reads JSON from the request body with a size limit and decodes it into T.
func Decode[T any](r *http.Request) (T, error) {
	var v T
	r.Body = http.MaxBytesReader(nil, r.Body, maxRequestBodySize)
	defer func() {
		_, _ = io.Copy(io.Discard, r.Body)
		_ = r.Body.Close()
	}()

	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		return v, fmt.Errorf("decode json: %w", err)
	}
	return v, nil
}
