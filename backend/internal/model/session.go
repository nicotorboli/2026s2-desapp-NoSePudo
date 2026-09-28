package model

import "time"

// Session es lo que recibe quien se autentica: la credencial con la que va a
// operar y el instante en que deja de servir.
//
// Vive en el dominio aunque no se persista, por la misma razón por la que
// User.PasswordHash vive acá sin que el dominio sepa de bcrypt: los campos son
// cadenas opacas, y quién las produce y con qué formato es cosa del adaptador.
// Lo que el dominio afirma es que un inicio de sesión exitoso entrega esto.
//
// En esta entrega sólo lleva la credencial de acceso; la de renovación se
// suma cuando exista el almacenamiento que permite revocarla.
type Session struct {
	AccessExpiresAt time.Time
	AccessToken     string
}
