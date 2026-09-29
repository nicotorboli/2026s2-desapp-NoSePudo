package dto

import (
	"fmt"
	"strings"
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)


const tokenTypeBearer = "Bearer"

const (
	maxEmailLength = 254
	minPasswordBytes = 8
	maxPasswordBytes = 72
)


type RegisterRequest struct {
	Email string `json:"email"`
	Password string `json:"password"`
}

func (r *RegisterRequest) Validate() error {
	if err := validateEmail(r.Email); err != nil {
		return err
	}
	return validatePassword(r.Password)
}

type LoginRequest struct {
	Email string `json:"email"`
	Password string `json:"password"`
}

func (l *LoginRequest) Validate() error {
	if err := validateEmail(l.Email); err != nil {
		return err
	}
	if l.Password == "" {
		return requiredFieldError("password")
	}
	return nil
}

type AccountResponse struct {
	Email string `json:"email"`
	ID int64 `json:"id"`
}


func AccountResponseDesdeModelo(user model.User) AccountResponse {
	return AccountResponse{
		Email: user.Email,
		ID: user.ID,
	}
}

type SessionResponse struct {
	AccessExpiresAt time.Time `json:"access_expires_at"`
	RefreshExpiresAt time.Time `json:"refresh_expires_at"`
	AccessToken string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType string `json:"token_type"`
}

func SessionResponseDesdeModelo(session model.Session) SessionResponse {
	return SessionResponse{
		AccessExpiresAt: session.AccessExpiresAt,
		RefreshExpiresAt: session.RefreshExpiresAt,
		AccessToken: session.AccessToken,
		RefreshToken: session.RefreshToken,
		TokenType: tokenTypeBearer,
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

