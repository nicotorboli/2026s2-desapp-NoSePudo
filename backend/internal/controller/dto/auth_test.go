package dto_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/controller/dto"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

// validPassword tiene 8 bytes, el mínimo aceptado.
const validPassword = "12345678"

// fakeStoredHash tiene la forma de un digest de bcrypt. Es un valor de prueba:
// lo que los casos de abajo comprueban es justamente que no salga por la
// respuesta.
const fakeStoredHash = "$2a$12$un-hash-que-no-debe-salir" //nolint:gosec // valor de prueba, no una credencial real

// aStoredUser es la cuenta tal como sale de la persistencia, con todo lo que
// el DTO tiene que dejar afuera.
func aStoredUser() model.User {
	return model.User{
		ID:           42,
		Email:        "nico@nosepudo.ar",
		PasswordHash: fakeStoredHash,
		Privilege:    model.PrivilegeSuperuser,
		Active:       true,
	}
}

func TestRegisterRequestAcceptsValidData(t *testing.T) {
	request := dto.RegisterRequest{Email: "nico@nosepudo.ar", Password: validPassword}

	if err := request.Validate(); err != nil {
		t.Errorf("Validate rechazó datos válidos: %v", err)
	}
}

// Los bordes del largo de la contraseña. 72 es el de bcrypt: más allá de ese
// byte la librería ignora el resto, así que aceptar 73 haría que dos
// contraseñas distintas colisionaran.
func TestRegisterRequestPasswordLengthBoundaries(t *testing.T) {
	cases := []struct {
		name   string
		length int
		valid  bool
	}{
		{"7 bytes, uno menos que el mínimo", 7, false},
		{"8 bytes, el mínimo", 8, true},
		{"9 bytes", 9, true},
		{"71 bytes", 71, true},
		{"72 bytes, el máximo de bcrypt", 72, true},
		{"73 bytes, uno más que el máximo", 73, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			request := dto.RegisterRequest{
				Email:    "nico@nosepudo.ar",
				Password: strings.Repeat("a", c.length),
			}

			err := request.Validate()

			if c.valid && err != nil {
				t.Errorf("una contraseña de %d bytes fue rechazada: %v", c.length, err)
			}
			if !c.valid && err == nil {
				t.Errorf("una contraseña de %d bytes fue aceptada", c.length)
			}
		})
	}
}

// El límite es en bytes, así que una contraseña de 72 caracteres acentuados
// pasa de largo aunque "parezca" que entra.
func TestRegisterRequestPasswordIsMeasuredInBytes(t *testing.T) {
	request := dto.RegisterRequest{
		Email:    "nico@nosepudo.ar",
		Password: strings.Repeat("ñ", 40), // 80 bytes en 40 runas
	}

	if err := request.Validate(); err == nil {
		t.Error("una contraseña de 80 bytes fue aceptada: bcrypt ignoraría lo que pasa de 72")
	}
}

func TestRegisterRequestEmailLengthBoundaries(t *testing.T) {
	// "@nosepudo.ar" son 12 bytes; el resto es la parte local.
	const domain = "@nosepudo.ar"

	cases := []struct {
		name  string
		total int
		valid bool
	}{
		{"254 caracteres, el máximo", 254, true},
		{"255 caracteres, uno más", 255, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			email := strings.Repeat("a", c.total-len(domain)) + domain
			if len(email) != c.total {
				t.Fatalf("el caso está mal armado: el email mide %d y debería medir %d", len(email), c.total)
			}

			request := dto.RegisterRequest{Email: email, Password: validPassword}
			err := request.Validate()

			if c.valid && err != nil {
				t.Errorf("un email de %d caracteres fue rechazado: %v", c.total, err)
			}
			if !c.valid && err == nil {
				t.Errorf("un email de %d caracteres fue aceptado", c.total)
			}
		})
	}
}

func TestRegisterRequestRejectsMalformedEmails(t *testing.T) {
	cases := []struct {
		name  string
		email string
	}{
		{"ausente", ""},
		{"sólo espacios", " \t "},
		{"sin arroba", "nico.nosepudo.ar"},
		{"parte local vacía", "@nosepudo.ar"},
		{"dominio vacío", "nico@"},
		{"dos arrobas", "nico@nose@pudo.ar"},
		{"sólo una arroba", "@"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			request := dto.RegisterRequest{Email: c.email, Password: validPassword}

			if err := request.Validate(); err == nil {
				t.Errorf("se aceptó el email %q", c.email)
			}
		})
	}
}

// el requerimiento: el mensaje dice qué campo estuvo mal y no repite lo que se mandó.
func TestRegisterRequestMessagesNameTheFieldWithoutEchoingTheSecret(t *testing.T) {
	const secret = "esta-contraseña-no-debe-aparecer" //nolint:gosec // valor de prueba, no una credencial real

	request := dto.RegisterRequest{Email: "sin-arroba", Password: secret}

	err := request.Validate()
	if err == nil {
		t.Fatal("se esperaba un error de validación")
	}
	if !strings.Contains(err.Error(), "email") {
		t.Errorf("el mensaje %q no nombra el campo", err)
	}
	if strings.Contains(err.Error(), secret) {
		t.Error("el mensaje repite la contraseña enviada")
	}
	if strings.Contains(err.Error(), "sin-arroba") {
		t.Error("el mensaje repite el valor enviado")
	}
}

func TestRegisterRequestReportsTheMissingFieldFirst(t *testing.T) {
	request := dto.RegisterRequest{Email: "", Password: ""}
	err := request.Validate()
	if err == nil {
		t.Fatal("se esperaba un error")
	}
	if !strings.Contains(err.Error(), "email") {
		t.Errorf("mensaje = %q, se esperaba que nombrara email", err)
	}
}

// El login no le exige largo mínimo a la contraseña: hacerlo le contaría a
// quien prueba contraseñas que las cortas ni siquiera llegan a compararse.
func TestLoginRequestDoesNotImposePasswordLength(t *testing.T) {
	request := dto.LoginRequest{Email: "nico@nosepudo.ar", Password: "x"}

	if err := request.Validate(); err != nil {
		t.Errorf("el login rechazó una contraseña corta: %v", err)
	}
}

func TestLoginRequestRejectsMissingFields(t *testing.T) {
	cases := []struct {
		name     string
		email    string
		password string
	}{
		{"email ausente", "", validPassword},
		{"email en blanco", " ", validPassword},
		{"email con forma imposible", "sin-arroba", validPassword},
		{"contraseña ausente", "nico@nosepudo.ar", ""},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			request := dto.LoginRequest{Email: c.email, Password: c.password}

			if err := request.Validate(); err == nil {
				t.Error("se aceptó una petición inválida")
			}
		})
	}
}

func TestAccountResponseDesdeModelo(t *testing.T) {
	user := aStoredUser()

	response := dto.AccountResponseDesdeModelo(user)

	if response.ID != 42 {
		t.Errorf("ID = %d, se esperaba 42", response.ID)
	}
	if response.Email != "nico@nosepudo.ar" {
		t.Errorf("Email = %q", response.Email)
	}
}

// Lo que importa del DTO es lo que NO lleva: el hash no puede salir por la
// respuesta, y el privilegio tampoco, porque una cuenta
// autoregistrada es siempre común y decirlo invitaría a creer que puede no
// serlo.
func TestAccountResponseSerializesNeitherHashNorPrivilege(t *testing.T) {
	user := aStoredUser()

	encoded, err := json.Marshal(dto.AccountResponseDesdeModelo(user))
	if err != nil {
		t.Fatalf("no se pudo serializar: %v", err)
	}

	body := string(encoded)
	for _, forbidden := range []string{user.PasswordHash, "password", "hash", "privilege", "superuser", "active"} {
		if strings.Contains(strings.ToLower(body), strings.ToLower(forbidden)) {
			t.Errorf("la respuesta contiene %q: %s", forbidden, body)
		}
	}
}
