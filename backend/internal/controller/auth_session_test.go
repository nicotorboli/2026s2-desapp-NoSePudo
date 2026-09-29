package controller_test

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/controller"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/httphandler"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/middleware"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

// fakeRefreshToken es el valor que el service simulado devuelve como credencial
// de renovación. Es una cadena de prueba.
const fakeRefreshToken = "una-credencial-de-renovacion" //nolint:gosec // valor de prueba, no una credencial real

// anActor es el actor que el middleware habría publicado tras verificar una
// credencial.
func anActor() middleware.Actor {
	return middleware.Actor{
		SessionID: "una-familia",
		CredentialID: "un-jti",
		ID: 42,
		Privilege: model.PrivilegeUser,
	}
}

// callWithActor corre el endpoint con el actor ya en el contexto, que es el
// estado en que se lo entrega la cadena de middleware. Un actor nulo simula una
// ruta registrada sin autenticación delante.
func callWithActor(
	t *testing.T,
	endpoint httphandler.Endpoint,
	actor *middleware.Actor,
) (int, map[string]any) {
	t.Helper()

	handler := httphandler.Wrap(endpoint, slog.New(slog.NewTextHandler(io.Discard, nil)))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/refresh", nil)
	if actor != nil {
		request = request.WithContext(middleware.WithActor(request.Context(), *actor))
	}
	handler.ServeHTTP(recorder, request)

	var body map[string]any
	if recorder.Body.Len() > 0 {
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
			t.Fatalf("la respuesta no es JSON: %v (%s)", err, recorder.Body.String())
		}
	}

	return recorder.Code, body
}

func TestRefreshEndpointReturns200WithTheNewSession(t *testing.T) {
	accessExpiry := time.Date(2026, time.September, 28, 12, 15, 0, 0, time.UTC)
	refreshExpiry := time.Date(2026, time.October, 5, 12, 0, 0, 0, time.UTC)

	service := &mockAuthService{session: model.Session{
		AccessExpiresAt: accessExpiry,
		RefreshExpiresAt: refreshExpiry,
		AccessToken: fakeAccessToken,
		RefreshToken: fakeRefreshToken,
	}}
	actor := anActor()

	status, body := callWithActor(t, controller.NewAuthController(service).Refresh(), &actor)

	if status != http.StatusOK {
		t.Fatalf("status = %d, se esperaba 200: %v", status, body)
	}

	// El contrato fija los cinco campos.
	for _, field := range []string{"access_token", "refresh_token", "token_type", "access_expires_at", "refresh_expires_at"} {
		if _, present := body[field]; !present {
			t.Errorf("falta el campo %q que fija el contrato: %v", field, body)
		}
	}
	if body["token_type"] != "Bearer" {
		t.Errorf("token_type = %v", body["token_type"])
	}
	if body["refresh_expires_at"] != refreshExpiry.Format(time.RFC3339) {
		t.Errorf("refresh_expires_at = %v, se esperaba RFC 3339", body["refresh_expires_at"])
	}
}

// Los tres datos con los que se llama al service salen del actor verificado y
// de ningún otro lugar.
func TestRefreshEndpointTakesEverythingFromTheVerifiedActor(t *testing.T) {
	service := &mockAuthService{}
	actor := anActor()

	if _, _ = callWithActor(t, controller.NewAuthController(service).Refresh(), &actor); service.refreshCalled != 1 {
		t.Fatalf("el service se llamó %d veces", service.refreshCalled)
	}

	if service.gotCredentialID != "un-jti" {
		t.Errorf("el service recibió la credencial %q", service.gotCredentialID)
	}
	if service.gotUserID != 42 {
		t.Errorf("el service recibió la cuenta %d", service.gotUserID)
	}
	if service.gotSessionID != "una-familia" {
		t.Errorf("el service recibió la familia %q", service.gotSessionID)
	}
}

// El reuso y la expiración se responden igual: 401 y el mismo mensaje. Que el
// reuso además haya cortado toda la cuenta es interno.
func TestRefreshEndpointAnswers401ForBothRefusals(t *testing.T) {
	cases := []struct {
		err error
		name string
	}{
		{fmt.Errorf("rotar: %w", model.ErrRefreshTokenReused), "reuso"},
		{fmt.Errorf("verificar: %w", model.ErrRefreshTokenExpired), "expirada"},
	}

	bodies := make([]map[string]any, 0, len(cases))

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			service := &mockAuthService{err: c.err}
			actor := anActor()

			status, body := callWithActor(t, controller.NewAuthController(service).Refresh(), &actor)

			if status != http.StatusUnauthorized {
				t.Errorf("status = %d, se esperaba 401", status)
			}
			if body["error"] != "authentication required" {
				t.Errorf("error = %v, se esperaba el mensaje del contrato", body["error"])
			}
			bodies = append(bodies, body)
		})
	}

	if len(bodies) == 2 && bodies[0]["error"] != bodies[1]["error"] {
		t.Error("el reuso y la expiración responden distinto, y eso le cuenta al cliente cuál fue")
	}
}

func TestLogoutEndpointReturns204(t *testing.T) {
	service := &mockAuthService{}
	actor := anActor()

	status, body := callWithActor(t, controller.NewAuthController(service).Logout(), &actor)

	if status != http.StatusNoContent {
		t.Fatalf("status = %d, se esperaba 204", status)
	}
	if len(body) != 0 {
		t.Errorf("un 204 no lleva cuerpo: %v", body)
	}
	if service.logoutCalled != 1 {
		t.Errorf("el service se llamó %d veces", service.logoutCalled)
	}
	if service.gotSessionID != "una-familia" {
		t.Errorf("se cerró la familia %q", service.gotSessionID)
	}
	if service.gotUserID != 42 {
		t.Errorf("se cerró la sesión de la cuenta %d", service.gotUserID)
	}
}

// Sin actor las dos operaciones responden 401 en vez de seguir con una cuenta
// cero. Es el fallo de una ruta registrada sin autenticación delante.
func TestSessionEndpointsRefuseWithoutAnActor(t *testing.T) {
	cases := map[string]func(*controller.RestAuthController) httphandler.Endpoint{
		"refresh": (*controller.RestAuthController).Refresh,
		"logout": (*controller.RestAuthController).Logout,
	}

	for name, endpoint := range cases {
		t.Run(name, func(t *testing.T) {
			service := &mockAuthService{}

			status, _ := callWithActor(t, endpoint(controller.NewAuthController(service)), nil)

			if status != http.StatusUnauthorized {
				t.Errorf("status = %d, se esperaba 401", status)
			}
			if service.refreshCalled != 0 || service.logoutCalled != 0 {
				t.Error("el service corrió sin que nadie hubiera sido autenticado")
			}
		})
	}
}

