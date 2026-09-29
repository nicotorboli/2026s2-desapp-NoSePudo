package httphandler_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/httphandler"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// serve corre el endpoint a través de Wrap y devuelve el status y el cuerpo
// que efectivamente le llegan al cliente.
func serve(t *testing.T, endpoint httphandler.Endpoint) (int, map[string]string) {
	t.Helper()

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)

	httphandler.Wrap(endpoint, discardLogger()).ServeHTTP(rec, req)

	var body map[string]string
	if rec.Body.Len() > 0 {
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("el cuerpo de la respuesta no es JSON: %v", err)
		}
	}

	return rec.Code, body
}

func TestWrapEncodesAnUndecoratedAPIError(t *testing.T) {
	endpoint := func(http.ResponseWriter, *http.Request) error {
		return httphandler.NewError(http.StatusUnauthorized, "credencial requerida", nil)
	}

	status, body := serve(t, endpoint)

	if status != http.StatusUnauthorized {
		t.Errorf("status = %d, se esperaba %d", status, http.StatusUnauthorized)
	}
	if body["error"] != "credencial requerida" {
		t.Errorf("error = %q, se esperaba %q", body["error"], "credencial requerida")
	}
}

// Es el criterio y el requerimiento: la refutación nace en una capa interna y sube decorada.
// Con una type assertion pelada esto daba 500.
func TestWrapUnwrapsAnAPIErrorDecoratedByInnerLayers(t *testing.T) {
	endpoint := func(http.ResponseWriter, *http.Request) error {
		inner := httphandler.NewError(http.StatusUnauthorized, "credencial requerida", nil)
		middle := fmt.Errorf("service: verificar sesión: %w", inner)
		return fmt.Errorf("controller: listar jugadores: %w", middle)
	}

	status, body := serve(t, endpoint)

	if status != http.StatusUnauthorized {
		t.Fatalf("status = %d, se esperaba %d: un error decorado se degradó a error interno", status, http.StatusUnauthorized)
	}
	if body["error"] != "credencial requerida" {
		t.Errorf("error = %q, se esperaba el mensaje del error original", body["error"])
	}
}

// El mensaje que se le responde al cliente es sólo Message(): el contexto que
// agregaron las capas intermedias queda en el log y no se filtra.
func TestWrapDoesNotLeakTheDecorationIntoTheResponse(t *testing.T) {
	endpoint := func(http.ResponseWriter, *http.Request) error {
		inner := httphandler.NewError(http.StatusConflict, "el email ya está en uso", errors.New("pq: duplicate key value violates unique constraint"))
		return fmt.Errorf("service: registrar cuenta: %w", inner)
	}

	status, body := serve(t, endpoint)

	if status != http.StatusConflict {
		t.Errorf("status = %d, se esperaba %d", status, http.StatusConflict)
	}
	if body["error"] != "el email ya está en uso" {
		t.Errorf("error = %q: se filtró la causa interna al cliente", body["error"])
	}
}

func TestWrapFallsBackToInternalServerErrorForAPlainError(t *testing.T) {
	endpoint := func(http.ResponseWriter, *http.Request) error {
		return fmt.Errorf("dao: %w", errors.New("connection refused"))
	}

	status, body := serve(t, endpoint)

	if status != http.StatusInternalServerError {
		t.Errorf("status = %d, se esperaba %d", status, http.StatusInternalServerError)
	}
	if body["error"] != "Internal Server Error" {
		t.Errorf("error = %q: un error sin status intencional no debe revelar su causa", body["error"])
	}
}

func TestWrapWritesNothingWhenTheEndpointSucceeds(t *testing.T) {
	endpoint := func(w http.ResponseWriter, _ *http.Request) error {
		return httphandler.Encode(w, http.StatusOK, map[string]string{"ok": "sí"})
	}

	status, body := serve(t, endpoint)

	if status != http.StatusOK {
		t.Errorf("status = %d, se esperaba %d", status, http.StatusOK)
	}
	if body["ok"] != "sí" {
		t.Errorf("body = %v, se esperaba el cuerpo que escribió el endpoint", body)
	}
}

// errors.Is tiene que seguir funcionando a través del error de borde, porque
// es como el controller mapea los centinelas de dominio a un status.
func TestErrorUnwrapsToItsCause(t *testing.T) {
	cause := errors.New("centinela de dominio")
	wrapped := fmt.Errorf("capa intermedia: %w", httphandler.NewError(http.StatusConflict, "conflicto", cause))

	if !errors.Is(wrapped, cause) {
		t.Error("errors.Is no encuentra la causa a través del error de borde")
	}
}

