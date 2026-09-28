package server

// AccessLevel es el requisito de credencial que cada ruta declara.
//
// El valor cero es AccessUndeclared y es un error de registración, no un
// default de "público". Esa es la mitad del trabajo que hace esta feature: en
// Go un endpoint al que se le olvidó el wrapper simplemente queda abierto, y
// ni el compilador ni el linter ni un test común dicen una palabra. Con el
// cero inválido, un literal de route que se olvide del campo no arranca.
type AccessLevel uint8

const (
	AccessUndeclared AccessLevel = iota
	AccessAnonymous
	AccessAuthenticated
	AccessRenewal
	AccessSuperuser
)

func (a AccessLevel) String() string {
	switch a {
	case AccessAnonymous:
		return "anonymous"
	case AccessAuthenticated:
		return "authenticated"
	case AccessRenewal:
		return "renewal"
	case AccessSuperuser:
		return "superuser"
	default:
		return "undeclared"
	}
}
