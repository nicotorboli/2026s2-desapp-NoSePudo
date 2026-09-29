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
// Lleva las dos credenciales que un inicio de sesión entrega: la de acceso, que
// es corta y se verifica sola, y la de renovación, que es larga y está
// persistida justamente para poder cortarla antes de que expire.
type Session struct {
	AccessExpiresAt time.Time
	RefreshExpiresAt time.Time
	AccessToken string
	RefreshToken string
}

