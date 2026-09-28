package dto

import (
	"fmt"
	"strings"
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

// tokenTypeBearer es el esquema con el que se presenta la credencial. El
// contrato lo fija como constante, no como algo que el cliente pueda elegir.
const tokenTypeBearer = "Bearer"

const (
	maxEmailLength = 254

	// Los dos extremos de la contraseña. El de arriba es el de bcrypt: ignora
	// todo lo que pase del byte 72, así que aceptar algo más largo haría que
	// dos contraseñas distintas terminaran siendo la misma. Rechazarlo es lo
	// honesto.
	minPasswordBytes = 8
	maxPasswordBytes = 72
)

// RegisterRequest es el cuerpo del alta de cuenta.
//
// No tiene campo de privilegio, y eso no es un olvido: es lo que hace
// imposible que un cliente se registre como superusuario. Como Decode rechaza
// los campos que el tipo no declara, un body que traiga "privilege" se rechaza
// con 400 y no hay ningún camino por el que ese valor llegue a una cuenta
// (FR-022, FR-024).
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Validate comprueba la forma de los campos, que es lo único que se puede
// responder mirando la petición y nada más. Las reglas de dominio —
// normalización, unicidad, el conjunto cerrado de privilegios — viven en model
// y en el service.
//
// La invoca httphandler.Decode, no el controller, así que ningún endpoint
// puede saltearla por olvidarse de llamarla.
func (r *RegisterRequest) Validate() error {
	if err := validateEmail(r.Email); err != nil {
		return err
	}
	return validatePassword(r.Password)
}

// LoginRequest es el cuerpo del inicio de sesión.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Validate rechaza un campo ausente, vacío o con forma imposible antes de que
// se busque cuenta alguna o se verifique credencial alguna.
//
// A diferencia del alta no comprueba los largos de la contraseña: acá no es
// una regla de forma sino el secreto que se va a comparar, y exigirle un
// mínimo le contaría a quien prueba contraseñas que las cortas ni siquiera
// llegan a compararse.
func (l *LoginRequest) Validate() error {
	if err := validateEmail(l.Email); err != nil {
		return err
	}
	if l.Password == "" {
		return requiredFieldError("password")
	}
	return nil
}

// AccountResponse es la cuenta recién creada.
//
// No lleva contraseña en ninguna forma, ni el nivel de privilegio: una cuenta
// autoregistrada es siempre un usuario común, y decirlo en la respuesta
// invitaría a creer que podría ser otra cosa.
type AccountResponse struct {
	Email string `json:"email"`
	ID    int64  `json:"id"`
}

// AccountResponseDesdeModelo es la conversión de dominio a borde. Vive en el
// DTO y no en el controller, que se limita a devolverla.
func AccountResponseDesdeModelo(user model.User) AccountResponse {
	return AccountResponse{
		Email: user.Email,
		ID:    user.ID,
	}
}

// SessionResponse es la sesión entregada tras un inicio de sesión exitoso.
//
// Es un objeto con campos nombrados y no la credencial pelada, para que las
// entregas siguientes puedan agregarle campos sin romper a los clientes que ya
// existan (FR-006).
type SessionResponse struct {
	AccessExpiresAt  time.Time `json:"access_expires_at"`
	RefreshExpiresAt time.Time `json:"refresh_expires_at"`
	AccessToken      string    `json:"access_token"`
	RefreshToken     string    `json:"refresh_token"`
	TokenType        string    `json:"token_type"`
}

func SessionResponseDesdeModelo(session model.Session) SessionResponse {
	return SessionResponse{
		AccessExpiresAt:  session.AccessExpiresAt,
		RefreshExpiresAt: session.RefreshExpiresAt,
		AccessToken:      session.AccessToken,
		RefreshToken:     session.RefreshToken,
		TokenType:        tokenTypeBearer,
	}
}

func requiredFieldError(field string) error {
	return fmt.Errorf("field %q is required", field)
}

func validateEmail(email string) error {
	trimmed := strings.TrimSpace(email)
	if trimmed == "" {
		return requiredFieldError("email")
	}

	if len(trimmed) > maxEmailLength {
		return fmt.Errorf("field %q must be at most %d characters", "email", maxEmailLength)
	}

	local, domain, found := strings.Cut(trimmed, "@")
	if !found || local == "" || domain == "" || strings.Contains(domain, "@") {
		return fmt.Errorf("field %q must be a valid email address", "email")
	}

	return nil
}

func validatePassword(password string) error {
	if password == "" {
		return requiredFieldError("password")
	}

	// Se miden bytes y no runas porque el límite de bcrypt son bytes: una
	// contraseña con acentos o emoji ocupa más de lo que aparenta.
	if len(password) < minPasswordBytes || len(password) > maxPasswordBytes {
		return fmt.Errorf(
			"field %q must be between %d and %d characters",
			"password", minPasswordBytes, maxPasswordBytes,
		)
	}

	return nil
}
