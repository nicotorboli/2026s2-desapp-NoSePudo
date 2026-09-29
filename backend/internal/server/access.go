package server

// AccessLevel define el requisito de autorización de una ruta.
// El valor cero es AccessUndeclared para forzar un esquema "fail-closed":
// si una ruta omite este campo, la registración falla en lugar de quedar pública por defecto.

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
