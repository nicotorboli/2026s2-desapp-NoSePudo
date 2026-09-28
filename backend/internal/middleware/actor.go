package middleware

import "github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"

// Actor es la cuenta a la que se le atribuye una operación, leída de una
// credencial ya verificada y de ningún otro lado (FR-010).
//
// Vive acá y no en model aunque la spec lo nombre entre sus entidades: fuera
// de una petición autenticada un Actor no significa nada. Es lo que el
// middleware extrae de una credencial que acaba de verificar, y es un User
// recortado —identidad y privilegio— que sólo al borde le sirve.
//
// No lleva el email a propósito: los logs nombran al actor, y ahí no puede
// haber datos personales (FR-028).
type Actor struct {
	SessionID string
	ID        int64
	Privilege model.PrivilegeLevel
}
