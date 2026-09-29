package model

import (
	"errors"
	"strings"
	"time"
)

// Errores de dominio de la cuenta. Viven al lado del tipo que los levanta y no
// saben nada de HTTP: es el controller el que los traduce a un status code.
var (
	// ErrEmailTaken indica que el identificador elegido ya está en uso. La
	// cuenta existente queda intacta.
	ErrEmailTaken = errors.New("el email ya está en uso")

	// ErrInvalidCredentials es el único desenlace de una autenticación
	// fallida: no distingue una contraseña incorrecta de una cuenta
	// inexistente.
	ErrInvalidCredentials = errors.New("credenciales inválidas")

	// ErrUserNotFound lo devuelve la persistencia cuando no hay cuenta con
	// ese identificador. Es un hecho interno y nunca se le responde al
	// cliente tal cual: el login lo convierte en ErrInvalidCredentials
	// justamente para no revelar si la cuenta existe.
	ErrUserNotFound = errors.New("no existe una cuenta con ese identificador")
)

// User es el participante del mercado. Los campos van de mayor a menor tamaño
// porque govet corre con fieldalignment.
type User struct {
	CreatedAt time.Time
	Email string
	PasswordHash string
	ID int64
	Privilege PrivilegeLevel
	Active bool
}

// NormalizeEmail deja la dirección en la forma en que se guarda y se busca:
// sin espacios alrededor y en minúsculas. La misma función corre en el alta y
// en el login, que es lo que hace que el índice único proteja algo — si sólo
// normalizara una de las dos puntas, una dirección podría volverse dos cuentas.
//
// Es idempotente: normalizar algo ya normalizado lo devuelve igual.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

