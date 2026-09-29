package model

// PrivilegeLevel es la distinción de dos valores entre un usuario común y el
// superusuario. Es una propiedad de la cuenta y nunca se deriva de la entrada
// del cliente.
//
// El valor cero es PrivilegeUnknown y jamás alcanza: un claim ausente, un
// typo o un campo olvidado terminan en rechazo y no en superusuario.
type PrivilegeLevel uint8

const (
	PrivilegeUnknown PrivilegeLevel = iota
	PrivilegeUser
	PrivilegeSuperuser
)

const (
	privilegeUserName = "user"
	privilegeSuperuserName = "superuser"
	privilegeUnknownName = "unknown"
)

func (p PrivilegeLevel) String() string {
	switch p {
	case PrivilegeUser:
		return privilegeUserName
	case PrivilegeSuperuser:
		return privilegeSuperuserName
	default:
		return privilegeUnknownName
	}
}

// ParsePrivilege traduce la forma textual que viaja en el claim priv. Nunca
// devuelve error: lo que no reconoce es PrivilegeUnknown, que Satisfies
// rechaza.
func ParsePrivilege(s string) PrivilegeLevel {
	switch s {
	case privilegeUserName:
		return PrivilegeUser
	case privilegeSuperuserName:
		return PrivilegeSuperuser
	default:
		return PrivilegeUnknown
	}
}

// Satisfies responde si este nivel alcanza para lo que required exige. Es
// falso en cuanto alguno de los dos lados es PrivilegeUnknown, aunque ambos
// lo sean: un requisito que no se sabe cuál es no lo cumple nadie.
func (p PrivilegeLevel) Satisfies(required PrivilegeLevel) bool {
	if p == PrivilegeUnknown || required == PrivilegeUnknown {
		return false
	}
	return p >= required
}

