package middleware_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/adapters"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/httphandler"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/middleware"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

// serveAuthenticatedAs corre la cadena completa —autenticación y después
// autorización— con una credencial que declara el privilegio indicado. Es el
// orden en que buildMux las aplica, y es el único en que la autorización tiene
// un actor para mirar.
func serveAuthenticatedAs(
	t *testing.T,
	held model.PrivilegeLevel,
	required model.PrivilegeLevel,
	spy *spyEndpoint,
) (int, map[string]string) {
	t.Helper()

	claims := validAccessClaims()
	claims.Privilege = held

	authentication := middleware.NewAuthentication(&mockTokenVerifier{claims: claims})
	authorization := middleware.NewAuthorization()

	// La autenticación va por fuera: corre primero y publica el actor.
	decorated := authentication.RequireAccessToken()(authorization.Require(required)(spy.endpoint()))

	return serveDecorated(t, decorated, "Bearer una-credencial")
}

// serveDecorated corre un endpoint ya decorado a través de Wrap.
func serveDecorated(t *testing.T, endpoint httphandler.Endpoint, authorization string) (int, map[string]string) {
	t.Helper()

	handler := httphandler.Wrap(endpoint, slog.New(slog.NewTextHandler(io.Discard, nil)))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/una-operacion", nil)
	if authorization != "" {
		request.Header.Set("Authorization", authorization)
	}
	handler.ServeHTTP(recorder, request)

	var body map[string]string
	if recorder.Body.Len() > 0 {
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
			t.Fatalf("la respuesta no es JSON: %v", err)
		}
	}

	return recorder.Code, body
}

// La tabla de decisión del privilegio. Incluye las dos filas con
// PrivilegeUnknown porque el requerimiento exige que un nivel ausente o irreconocible
// sea insuficiente y nunca superusuario.
func TestRequirePrivilegeDecisionTable(t *testing.T) {
	cases := []struct {
		name string
		held model.PrivilegeLevel
		required model.PrivilegeLevel
		want int
	}{
		{"usuario donde se pide usuario", model.PrivilegeUser, model.PrivilegeUser, http.StatusOK},
		{"superusuario donde se pide usuario", model.PrivilegeSuperuser, model.PrivilegeUser, http.StatusOK},
		{"superusuario donde se pide superusuario", model.PrivilegeSuperuser, model.PrivilegeSuperuser, http.StatusOK},
		{"usuario donde se pide superusuario", model.PrivilegeUser, model.PrivilegeSuperuser, http.StatusForbidden},
		{"privilegio desconocido donde se pide usuario", model.PrivilegeUnknown, model.PrivilegeUser, http.StatusForbidden},
		{"privilegio desconocido donde se pide superusuario", model.PrivilegeUnknown, model.PrivilegeSuperuser, http.StatusForbidden},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			spy := &spyEndpoint{}

			status, body := serveAuthenticatedAs(t, c.held, c.required, spy)

			if status != c.want {
				t.Errorf("status = %d, se esperaba %d", status, c.want)
			}

			if c.want == http.StatusOK {
				if spy.calls != 1 {
					t.Errorf("la operación corrió %d veces, se esperaba 1", spy.calls)
				}
				return
			}

			if spy.calls != 0 {
				t.Error("la operación corrió pese al privilegio insuficiente")
			}
			if body["error"] != "insufficient privilege" {
				t.Errorf("error = %q, se esperaba el mensaje de privilegio insuficiente", body["error"])
			}
		})
	}
}

// el requerimiento: quien llama tiene que poder distinguir "no estás autenticado" de
// "estás autenticado pero no te alcanza". Son dos status y dos mensajes.
func TestUnauthenticatedAndUnprivilegedAreDistinguishable(t *testing.T) {
	authentication := middleware.NewAuthentication(&mockTokenVerifier{claims: validAccessClaims()})
	authorization := middleware.NewAuthorization()

	chain := func(spy *spyEndpoint) httphandler.Endpoint {
		return authentication.RequireAccessToken()(authorization.Require(model.PrivilegeSuperuser)(spy.endpoint()))
	}

	// Sin credencial.
	unauthenticatedStatus, unauthenticatedBody := serveDecorated(t, chain(&spyEndpoint{}), "")

	// Con credencial válida de un usuario común, que no alcanza.
	unprivilegedStatus, unprivilegedBody := serveAuthenticatedAs(
		t, model.PrivilegeUser, model.PrivilegeSuperuser, &spyEndpoint{},
	)

	if unauthenticatedStatus != http.StatusUnauthorized {
		t.Errorf("sin credencial dio %d, se esperaba 401", unauthenticatedStatus)
	}
	if unprivilegedStatus != http.StatusForbidden {
		t.Errorf("con privilegio insuficiente dio %d, se esperaba 403", unprivilegedStatus)
	}
	if unauthenticatedBody["error"] == unprivilegedBody["error"] {
		t.Errorf("los dos rechazos dicen lo mismo (%q) y no se pueden distinguir", unauthenticatedBody["error"])
	}
}

// Si la cadena se armara sin autenticación delante, la autorización no tiene
// actor que mirar. Se responde 401 y no se deja pasar: el fallo de un armado
// mal hecho es rechazar, no permitir.
func TestRequirePrivilegeRefusesWhenThereIsNoActor(t *testing.T) {
	spy := &spyEndpoint{}

	decorated := middleware.NewAuthorization().Require(model.PrivilegeUser)(spy.endpoint())

	status, _ := serveDecorated(t, decorated, "Bearer una-credencial")

	if status != http.StatusUnauthorized {
		t.Errorf("status = %d, se esperaba 401", status)
	}
	if spy.calls != 0 {
		t.Error("la operación corrió sin que nadie hubiera sido autenticado")
	}
}

// El privilegio se lee del actor que puso la autenticación, es decir de la
// credencial verificada, y no de nada que el cliente haya mandado.
func TestRequirePrivilegeReadsTheLevelFromTheVerifiedCredential(t *testing.T) {
	claims := validAccessClaims()
	claims.Privilege = model.PrivilegeSuperuser

	authentication := middleware.NewAuthentication(&mockTokenVerifier{claims: claims})
	spy := &spyEndpoint{}

	decorated := authentication.RequireAccessToken()(
		middleware.NewAuthorization().Require(model.PrivilegeSuperuser)(spy.endpoint()),
	)

	// La petición trae un privilegio en una cabecera y en la query. Ninguno
	// tiene por qué importar.
	handler := httphandler.Wrap(decorated, slog.New(slog.NewTextHandler(io.Discard, nil)))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/una-operacion?privilege=user", nil)
	request.Header.Set("Authorization", "Bearer una-credencial")
	request.Header.Set("X-Privilege", "user")
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Errorf("status = %d: el privilegio del cliente pesó sobre el de la credencial", recorder.Code)
	}
	if spy.actor.Privilege != model.PrivilegeSuperuser {
		t.Errorf("actor.Privilege = %v, se esperaba superuser", spy.actor.Privilege)
	}
}

// Una credencial de refresco no abre una operación de superusuario: la cadena
// la rechaza en el primer eslabón, antes de mirar privilegio alguno.
func TestSuperuserChainRefusesARefreshToken(t *testing.T) {
	claims := validAccessClaims()
	claims.Kind = adapters.KindRefresh
	claims.Privilege = model.PrivilegeSuperuser

	authentication := middleware.NewAuthentication(&mockTokenVerifier{claims: claims})
	spy := &spyEndpoint{}

	decorated := authentication.RequireAccessToken()(
		middleware.NewAuthorization().Require(model.PrivilegeSuperuser)(spy.endpoint()),
	)

	status, body := serveDecorated(t, decorated, "Bearer una-credencial")

	if status != http.StatusUnauthorized {
		t.Errorf("status = %d, se esperaba 401", status)
	}
	if body["error"] != "authentication required" {
		t.Errorf("error = %q: se esperaba el rechazo de autenticación y no el de privilegio", body["error"])
	}
	if spy.calls != 0 {
		t.Error("la operación corrió con una credencial de refresco")
	}
}

