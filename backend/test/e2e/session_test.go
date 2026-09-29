package e2e_test

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"
)

// postWithBearer hace una petición sin cuerpo presentando la credencial
// indicada, que es como se llaman renovación y cierre de sesión.
func (s *stack) postWithBearer(t *testing.T, path, token string) (int, map[string]any) {
	t.Helper()

	request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, s.server.URL+path, nil)
	if err != nil {
		t.Fatalf("no se pudo armar la petición: %v", err)
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}

	response, err := s.server.Client().Do(request)
	if err != nil {
		t.Fatalf("la petición falló: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	raw, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("no se pudo leer la respuesta: %v", err)
	}

	var body map[string]any
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Fatalf("la respuesta no es JSON: %v (%s)", err, raw)
		}
	}

	return response.StatusCode, body
}

func tokensOf(t *testing.T, session map[string]any) (accessToken, refreshToken string) {
	t.Helper()

	accessToken, _ = session["access_token"].(string)
	refreshToken, _ = session["refresh_token"].(string)

	if accessToken == "" || refreshToken == "" {
		t.Fatalf("la sesión no trae las dos credenciales: %v", session)
	}

	return accessToken, refreshToken
}

// Escenario: el inicio de sesión entrega las dos credenciales, y la de
// renovación vive más que la de acceso.
func TestSignInDeliversBothCredentials(t *testing.T) {
	stack := newStack(t)

	session := stack.registerAndLogin(t, "nico@nosepudo.ar", "una-contraseña")
	accessToken, refreshToken := tokensOf(t, session)

	accessExpiry, err := time.Parse(time.RFC3339, session["access_expires_at"].(string))
	if err != nil {
		t.Fatalf("access_expires_at no es RFC 3339: %v", err)
	}
	refreshExpiry, err := time.Parse(time.RFC3339, session["refresh_expires_at"].(string))
	if err != nil {
		t.Fatalf("refresh_expires_at no es RFC 3339: %v", err)
	}
	if !refreshExpiry.After(accessExpiry) {
		t.Errorf("la renovación (%v) no vive más que el acceso (%v)", refreshExpiry, accessExpiry)
	}

	// Las dos credenciales comparten familia y difieren en tipo.
	accessClaims := claimsOf(t, accessToken)
	refreshClaims := claimsOf(t, refreshToken)

	if accessClaims["typ"] != "access" || refreshClaims["typ"] != "refresh" {
		t.Errorf("los tipos son %v y %v", accessClaims["typ"], refreshClaims["typ"])
	}
	if accessClaims["sid"] != refreshClaims["sid"] {
		t.Error("las dos credenciales de una sesión no comparten familia")
	}
	if refreshClaims["priv"] != nil {
		t.Errorf("la credencial de renovación lleva priv = %v y no debería llevarlo", refreshClaims["priv"])
	}
}

// Escenario:escenarios 2 y 3: la renovación entrega una credencial nueva sin
// reenviar la contraseña, e invalida la que se usó.
func TestRefreshIssuesANewSessionAndConsumesTheOldCredential(t *testing.T) {
	stack := newStack(t)

	session := stack.registerAndLogin(t, "nico@nosepudo.ar", "una-contraseña")
	_, refreshToken := tokensOf(t, session)

	status, renewed := stack.postWithBearer(t, "/auth/refresh", refreshToken)
	if status != http.StatusOK {
		t.Fatalf("status = %d, se esperaba 200: %v", status, renewed)
	}

	newAccessToken, newRefreshToken := tokensOf(t, renewed)
	if newRefreshToken == refreshToken {
		t.Error("la renovación devolvió la misma credencial de renovación")
	}

	// La credencial nueva sirve para el catálogo.
	if catalogStatus, _ := stack.get(t, "/players", "Bearer "+newAccessToken); catalogStatus != http.StatusOK {
		t.Errorf("la credencial renovada no abre el catálogo: %d", catalogStatus)
	}

	// Y sigue en la misma sesión.
	if claimsOf(t, newRefreshToken)["sid"] != claimsOf(t, refreshToken)["sid"] {
		t.Error("la renovación cambió de familia de sesión")
	}
}

// Escenario:, del todo: con la credencial de acceso ya vencida, la
// aplicación sigue funcionando sin volver a pedir la contraseña.
func TestRefreshKeepsTheSessionAliveAfterTheAccessCredentialExpires(t *testing.T) {
	stack := newStackWithAccessTTL(t, time.Second)

	session := stack.registerAndLogin(t, "nico@nosepudo.ar", "una-contraseña")
	accessToken, refreshToken := tokensOf(t, session)

	time.Sleep(2 * time.Second)

	// La de acceso ya no sirve.
	if status, _ := stack.get(t, "/players", "Bearer "+accessToken); status != http.StatusUnauthorized {
		t.Fatalf("la credencial de acceso seguía sirviendo: %d", status)
	}

	// La de renovación sí, y devuelve una de acceso nueva.
	status, renewed := stack.postWithBearer(t, "/auth/refresh", refreshToken)
	if status != http.StatusOK {
		t.Fatalf("la renovación devolvió %d: %v", status, renewed)
	}

	newAccessToken, _ := tokensOf(t, renewed)
	if catalogStatus, _ := stack.get(t, "/players", "Bearer "+newAccessToken); catalogStatus != http.StatusOK {
		t.Errorf("la credencial renovada no abre el catálogo: %d", catalogStatus)
	}
}

// Escenario: y el criterio: presentar dos veces la misma credencial se rechaza
// y deja la cuenta sin ninguna credencial de renovación usable.
func TestReusingARefreshCredentialRevokesTheWholeAccount(t *testing.T) {
	stack := newStack(t)

	session := stack.registerAndLogin(t, "nico@nosepudo.ar", "una-contraseña")
	_, stolenToken := tokensOf(t, session)

	// Una segunda sesión, como si fuera otro dispositivo.
	status, second := stack.postJSON(t, "/auth/login",
		`{"email":"nico@nosepudo.ar","password":"una-contraseña"}`)
	if status != http.StatusOK {
		t.Fatalf("el segundo inicio de sesión devolvió %d", status)
	}
	_, secondRefreshToken := tokensOf(t, second)

	// Uso legítimo.
	status, renewed := stack.postWithBearer(t, "/auth/refresh", stolenToken)
	if status != http.StatusOK {
		t.Fatalf("la primera renovación devolvió %d: %v", status, renewed)
	}
	_, renewedToken := tokensOf(t, renewed)

	// El robo: la misma credencial otra vez.
	status, body := stack.postWithBearer(t, "/auth/refresh", stolenToken)
	if status != http.StatusUnauthorized {
		t.Fatalf("el segundo uso devolvió %d, se esperaba 401: %v", status, body)
	}

	if live := stack.countLiveRefreshTokens(t); live != 0 {
		t.Errorf("quedaron %d credenciales vivas, se esperaban 0", live)
	}

	// Ni la que la renovación había emitido ni la de la otra sesión sirven.
	for name, token := range map[string]string{
		"la que emitió la renovación": renewedToken,
		"la de la otra sesión":        secondRefreshToken,
	} {
		if status, _ := stack.postWithBearer(t, "/auth/refresh", token); status != http.StatusUnauthorized {
			t.Errorf("%s sigue sirviendo: %d", name, status)
		}
	}
}

// Escenario: una credencial de renovación expirada se rechaza y el
// titular vuelve a iniciar sesión.
func TestAnExpiredRefreshCredentialIsRefused(t *testing.T) {
	stack := newStackWithRefreshTTL(t, time.Second)

	session := stack.registerAndLogin(t, "nico@nosepudo.ar", "una-contraseña")
	_, refreshToken := tokensOf(t, session)

	time.Sleep(2 * time.Second)

	status, body := stack.postWithBearer(t, "/auth/refresh", refreshToken)
	if status != http.StatusUnauthorized {
		t.Errorf("status = %d, se esperaba 401: %v", status, body)
	}

	// Y volver a entrar funciona.
	loginStatus, _ := stack.postJSON(t, "/auth/login",
		`{"email":"nico@nosepudo.ar","password":"una-contraseña"}`)
	if loginStatus != http.StatusOK {
		t.Errorf("no se pudo volver a iniciar sesión: %d", loginStatus)
	}
}

// Escenario: y el criterio: al cerrar sesión la credencial de renovación queda
// invalidada y no se puede usar de nuevo.
func TestSignOutInvalidatesTheRefreshCredential(t *testing.T) {
	stack := newStack(t)

	session := stack.registerAndLogin(t, "nico@nosepudo.ar", "una-contraseña")
	accessToken, refreshToken := tokensOf(t, session)

	status, body := stack.postWithBearer(t, "/auth/logout", accessToken)
	if status != http.StatusNoContent {
		t.Fatalf("el cierre de sesión devolvió %d, se esperaba 204: %v", status, body)
	}

	if status, _ := stack.postWithBearer(t, "/auth/refresh", refreshToken); status != http.StatusUnauthorized {
		t.Errorf("la credencial de renovación sigue sirviendo después de cerrar sesión: %d", status)
	}

	// La de acceso no se invalida: expira sola, y es corta.
	if status, _ := stack.get(t, "/players", "Bearer "+accessToken); status != http.StatusOK {
		t.Errorf("la credencial de acceso se invalidó al cerrar sesión: %d", status)
	}
}

// El caso borde: cerrar una sesión no cierra las otras.
func TestSignOutClosesOneSessionOnly(t *testing.T) {
	stack := newStack(t)

	phone := stack.registerAndLogin(t, "nico@nosepudo.ar", "una-contraseña")
	phoneAccess, phoneRefresh := tokensOf(t, phone)

	status, desktop := stack.postJSON(t, "/auth/login",
		`{"email":"nico@nosepudo.ar","password":"una-contraseña"}`)
	if status != http.StatusOK {
		t.Fatalf("el segundo inicio de sesión devolvió %d", status)
	}
	_, desktopRefresh := tokensOf(t, desktop)

	// Las dos sesiones tienen familias distintas.
	if claimsOf(t, phoneRefresh)["sid"] == claimsOf(t, desktopRefresh)["sid"] {
		t.Fatal("los dos inicios de sesión comparten familia")
	}

	if status, _ := stack.postWithBearer(t, "/auth/logout", phoneAccess); status != http.StatusNoContent {
		t.Fatalf("el cierre de sesión devolvió %d", status)
	}

	if status, _ := stack.postWithBearer(t, "/auth/refresh", phoneRefresh); status != http.StatusUnauthorized {
		t.Errorf("la sesión cerrada sigue renovando: %d", status)
	}
	if status, _ := stack.postWithBearer(t, "/auth/refresh", desktopRefresh); status != http.StatusOK {
		t.Errorf("cerrar una sesión cerró la otra: %d", status)
	}
}

// Escenario:escenarios 7 y 8, y el requerimiento: las dos credenciales no son intercambiables
// en ninguna dirección.
func TestTheTwoCredentialsAreNotInterchangeable(t *testing.T) {
	stack := newStack(t)

	session := stack.registerAndLogin(t, "nico@nosepudo.ar", "una-contraseña")
	accessToken, refreshToken := tokensOf(t, session)

	// Escenario 8: una credencial de acceso presentada a la renovación.
	if status, _ := stack.postWithBearer(t, "/auth/refresh", accessToken); status != http.StatusUnauthorized {
		t.Errorf("la renovación aceptó una credencial de acceso: %d", status)
	}

	// Escenario 7: una credencial de renovación presentada a cualquier otro
	// endpoint. Nunca da acceso a un recurso.
	if status, _ := stack.get(t, "/players", "Bearer "+refreshToken); status != http.StatusUnauthorized {
		t.Errorf("el catálogo aceptó una credencial de renovación: %d", status)
	}
	if status, _ := stack.postWithBearer(t, "/auth/logout", refreshToken); status != http.StatusUnauthorized {
		t.Errorf("el cierre de sesión aceptó una credencial de renovación: %d", status)
	}

	// Y la credencial de renovación sigue viva: rechazarla en el lugar
	// equivocado no es consumirla.
	if status, _ := stack.postWithBearer(t, "/auth/refresh", refreshToken); status != http.StatusOK {
		t.Errorf("presentarla donde no correspondía la invalidó: %d", status)
	}
}

// Las dos operaciones de sesión exigen credencial, como declara la tabla.
func TestSessionEndpointsRequireACredential(t *testing.T) {
	stack := newStack(t)

	for _, path := range []string{"/auth/refresh", "/auth/logout"} {
		t.Run(path, func(t *testing.T) {
			status, body := stack.postWithBearer(t, path, "")

			if status != http.StatusUnauthorized {
				t.Errorf("status = %d, se esperaba 401: %v", status, body)
			}
			if body["error"] != "authentication required" {
				t.Errorf("error = %v", body["error"])
			}
		})
	}
}

// Una cadena de renovaciones sucesivas funciona, y en cada paso queda
// exactamente una credencial viva.
func TestSuccessiveRenewalsStayInOneChain(t *testing.T) {
	stack := newStack(t)

	session := stack.registerAndLogin(t, "nico@nosepudo.ar", "una-contraseña")
	_, refreshToken := tokensOf(t, session)
	family := claimsOf(t, refreshToken)["sid"]

	for step := range 4 {
		status, renewed := stack.postWithBearer(t, "/auth/refresh", refreshToken)
		if status != http.StatusOK {
			t.Fatalf("la renovación %d devolvió %d: %v", step+1, status, renewed)
		}

		_, refreshToken = tokensOf(t, renewed)

		if claimsOf(t, refreshToken)["sid"] != family {
			t.Fatalf("la renovación %d cambió de familia", step+1)
		}
		if live := stack.countLiveRefreshTokens(t); live != 1 {
			t.Errorf("después de la renovación %d hay %d credenciales vivas, se esperaba 1", step+1, live)
		}
	}
}
