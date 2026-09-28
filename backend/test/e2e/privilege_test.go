package e2e_test

import (
	"net/http"
	"testing"
)

const (
	superuserEmail    = "admin@nosepudo.ar"
	superuserPassword = "la-del-superusuario" //nolint:gosec // valor de prueba, no una credencial real
)

// US5, escenarios 2 y 3: cada credencial declara el nivel de la cuenta que la
// pidió, y ese nivel sale de la cuenta y nunca del cliente.
func TestCredentialStatesTheAccountPrivilege(t *testing.T) {
	stack := newStack(t)
	stack.ensureSuperuser(t, superuserEmail, superuserPassword)

	commonUser := stack.registerAndLogin(t, "nico@nosepudo.ar", "una-contraseña")

	status, superuser := stack.postJSON(t, "/auth/login",
		`{"email":"`+superuserEmail+`","password":"`+superuserPassword+`"}`)
	if status != http.StatusOK {
		t.Fatalf("el superusuario no pudo iniciar sesión: %d %v", status, superuser)
	}

	commonClaims := claimsOf(t, commonUser["access_token"].(string))
	superuserClaims := claimsOf(t, superuser["access_token"].(string))

	if commonClaims["priv"] != "user" {
		t.Errorf("la credencial del usuario común declara priv = %v", commonClaims["priv"])
	}
	if superuserClaims["priv"] != "superuser" {
		t.Errorf("la credencial del superusuario declara priv = %v", superuserClaims["priv"])
	}
}

// US5, escenario 4: después del aprovisionamiento hay exactamente un
// superusuario, y no se creó a través del alta.
func TestExactlyOneSuperuserExists(t *testing.T) {
	stack := newStack(t)
	stack.ensureSuperuser(t, superuserEmail, superuserPassword)

	// Varias cuentas comunes por el medio, incluida una que se llama como si
	// fuera administradora.
	for _, email := range []string{"nico@nosepudo.ar", "otro@nosepudo.ar", "admin2@nosepudo.ar"} {
		status, body := stack.postJSON(t, "/auth/register",
			`{"email":"`+email+`","password":"una-contraseña"}`)
		if status != http.StatusCreated {
			t.Fatalf("el alta de %s devolvió %d: %v", email, status, body)
		}
	}

	if count := stack.countSuperusers(t); count != 1 {
		t.Errorf("hay %d superusuarios, se esperaba exactamente 1", count)
	}
}

// El aprovisionamiento es idempotente: reiniciar no crea una segunda cuenta ni
// le pisa la contraseña a la que ya estaba.
func TestProvisioningTheSuperuserTwiceChangesNothing(t *testing.T) {
	stack := newStack(t)
	stack.ensureSuperuser(t, superuserEmail, superuserPassword)

	hashBefore := stack.superuserHash(t)

	// Un segundo arranque, con otra contraseña en la configuración.
	stack.ensureSuperuser(t, superuserEmail, "una-contraseña-distinta")

	if hashAfter := stack.superuserHash(t); hashAfter != hashBefore {
		t.Error("el segundo aprovisionamiento le cambió la contraseña al superusuario")
	}
	if count := stack.countSuperusers(t); count != 1 {
		t.Errorf("hay %d superusuarios después de dos aprovisionamientos", count)
	}

	// Y la contraseña original sigue sirviendo.
	status, _ := stack.postJSON(t, "/auth/login",
		`{"email":"`+superuserEmail+`","password":"`+superuserPassword+`"}`)
	if status != http.StatusOK {
		t.Errorf("el superusuario ya no puede entrar con su contraseña: %d", status)
	}
}

func (s *stack) superuserHash(t *testing.T) string {
	t.Helper()

	var hash string
	if err := s.db.QueryRowContext(t.Context(),
		"SELECT password_hash FROM users WHERE email = $1", superuserEmail,
	).Scan(&hash); err != nil {
		t.Fatalf("no se pudo leer el hash del superusuario: %v", err)
	}

	return hash
}

// FR-019: no hay forma de conseguir privilegio de superusuario a través del
// alta, ni pidiéndolo en el cuerpo con el nombre que sea.
func TestRegistrationCannotReachSuperuserPrivilege(t *testing.T) {
	stack := newStack(t)

	bodies := []string{
		`{"email":"a@nosepudo.ar","password":"una-contraseña","privilege":"superuser"}`,
		`{"email":"b@nosepudo.ar","password":"una-contraseña","priv":"superuser"}`,
		`{"email":"c@nosepudo.ar","password":"una-contraseña","is_admin":true}`,
		`{"email":"d@nosepudo.ar","password":"una-contraseña","role":"superuser"}`,
	}

	for _, body := range bodies {
		status, response := stack.postJSON(t, "/auth/register", body)
		if status != http.StatusBadRequest {
			t.Errorf("status = %d para %s, se esperaba 400: %v", status, body, response)
		}
	}

	// El alta legítima crea un usuario común.
	if status, _ := stack.postJSON(t, "/auth/register",
		`{"email":"legitimo@nosepudo.ar","password":"una-contraseña"}`); status != http.StatusCreated {
		t.Fatalf("el alta legítima devolvió %d", status)
	}

	if count := stack.countSuperusers(t); count != 0 {
		t.Errorf("hay %d superusuarios y ninguno debería haberse creado por el alta", count)
	}
}

// US5, escenario 1: toda cuenta tiene exactamente uno de los dos niveles, y
// ninguna queda con el valor cero, que no alcanza para nada.
func TestEveryAccountHoldsExactlyOneOfTheTwoLevels(t *testing.T) {
	stack := newStack(t)
	stack.ensureSuperuser(t, superuserEmail, superuserPassword)
	stack.registerAndLogin(t, "nico@nosepudo.ar", "una-contraseña")

	rows, err := stack.db.QueryContext(t.Context(), "SELECT email, privilege FROM users")
	if err != nil {
		t.Fatalf("no se pudieron leer las cuentas: %v", err)
	}
	defer func() { _ = rows.Close() }()

	accounts := 0
	for rows.Next() {
		var (
			email     string
			privilege int16
		)
		if err := rows.Scan(&email, &privilege); err != nil {
			t.Fatalf("no se pudo leer una cuenta: %v", err)
		}
		accounts++

		// 1 es usuario común y 2 superusuario; 0 es el valor cero del dominio,
		// que no alcanza para nada y no debería estar guardado nunca.
		if privilege != 1 && privilege != 2 {
			t.Errorf("la cuenta %q tiene privilegio %d, que no es ninguno de los dos niveles", email, privilege)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("error recorriendo las cuentas: %v", err)
	}

	if accounts != 2 {
		t.Errorf("se examinaron %d cuentas, se esperaban 2", accounts)
	}
}
