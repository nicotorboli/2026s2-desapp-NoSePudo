package middleware

import "github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"

// Actor es la cuenta a la que se le atribuye una operación, leída de una
// credencial ya verificada y de ningún otro lado.
//
// Vive acá y no en model aunque se mencione entre sus entidades: fuera
// de una petición autenticada un Actor no significa nada. Es lo que el
// middleware extrae de una credencial que acaba de verificar, y es un User
// recortado —identidad y privilegio— que sólo al borde le sirve.
//
// No lleva el email a propósito: los logs nombran al actor, y ahí no puede
// haber datos personales.
type Actor struct {
	SessionID string

	// CredentialID es el jti de la credencial que identificó a este actor.
	// El data-model no lo listaba, y hace falta: la renovación tiene que
	// marcar usada exactamente la fila que le presentaron, y sin el jti no hay
	// forma de nombrarla.
	CredentialID string

	ID        int64
	Privilege model.PrivilegeLevel
}
