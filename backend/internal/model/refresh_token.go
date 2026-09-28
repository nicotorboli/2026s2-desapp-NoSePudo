package model

import (
	"errors"
	"time"
)

// Errores de la credencial de renovación, al lado del tipo que los levanta.
//
// Se llaman ErrRefreshToken* y no ErrToken* como decía el data-model, porque
// adapters ya tiene un ErrTokenExpired para la expiración de la firma. Son dos
// cosas distintas —que el JWT haya vencido y que la fila persistida ya no
// sirva— y confundirlas en la lectura es pedir un error.
var (
	// ErrRefreshTokenReused indica que se presentó una credencial que ya había
	// sido usada o revocada. Se trata como evidencia de robo: además de
	// rechazarla, se cortan todas las credenciales vivas de la cuenta.
	ErrRefreshTokenReused = errors.New("la credencial de renovación ya fue usada o revocada")

	// ErrRefreshTokenExpired indica que la fila existe y está viva pero su
	// instante de expiración ya pasó. El titular vuelve a iniciar sesión.
	ErrRefreshTokenExpired = errors.New("la credencial de renovación expiró")

	// ErrRefreshTokenNotFound indica que no hay fila con ese identificador. Es
	// un hecho interno: el service lo trata como reuso, porque una credencial
	// correctamente firmada cuya fila no existe es una que ya fue consumida.
	ErrRefreshTokenNotFound = errors.New("no existe una credencial de renovación con ese identificador")

	// ErrRefreshTokenRevoked indica que la credencial fue cortada, típicamente
	// por un cierre de sesión. Se rechaza y nada más.
	//
	// Es distinto de ErrRefreshTokenReused a propósito, y la diferencia vale la
	// pena: una credencial USADA que reaparece es evidencia de que hay un
	// segundo poseedor, porque el legítimo ya la consumió. Una REVOCADA que
	// reaparece es el mismo cliente reintentando después de cerrar sesión, y
	// tratarlo como robo le tiraría abajo las sesiones de sus otros
	// dispositivos, que es justo lo que el caso borde de la especificación
	// prohíbe.
	ErrRefreshTokenRevoked = errors.New("la credencial de renovación fue revocada")
)

// RefreshToken es la credencial de renovación tal como el sistema la persiste.
//
// La cadena del token no se guarda: esta fila es lo que la hace revocable, y su
// ID es el claim jti de la credencial firmada. Es un modelo de dominio porque
// cruza un límite de repositorio y porque tiene invariantes propias que vale
// testear sin base.
//
// Los campos van de mayor a menor tamaño porque govet corre con fieldalignment.
type RefreshToken struct {
	IssuedAt  time.Time
	ExpiresAt time.Time
	UsedAt    *time.Time
	RevokedAt *time.Time
	ID        string
	FamilyID  string
	UserID    int64
}

// IsLive es un método y no una columna, así que los tres indicadores no pueden
// contradecirse entre sí.
func (t RefreshToken) IsLive(now time.Time) bool {
	return t.UsedAt == nil && t.RevokedAt == nil && now.Before(t.ExpiresAt)
}

// IsExpired distingue el motivo por el que una credencial no está viva: haber
// expirado manda a iniciar sesión de nuevo, mientras que haber sido usada o
// revocada es lo que dispara la respuesta al robo.
func (t RefreshToken) IsExpired(now time.Time) bool {
	return !now.Before(t.ExpiresAt)
}

// IsConsumed es verdadero cuando la credencial ya no sirve porque se usó o se
// revocó, sin distinguir cuál de las dos.
func (t RefreshToken) IsConsumed() bool {
	return t.UsedAt != nil || t.RevokedAt != nil
}

// IsUsed es la condición que convierte un segundo uso en evidencia de robo: el
// poseedor legítimo ya la canjeó, así que quien la presenta ahora es otro.
func (t RefreshToken) IsUsed() bool {
	return t.UsedAt != nil
}

// IsRevoked es la credencial que se cortó a mano, normalmente por un cierre de
// sesión. Presentarla de nuevo se rechaza sin más consecuencias.
func (t RefreshToken) IsRevoked() bool {
	return t.RevokedAt != nil
}
