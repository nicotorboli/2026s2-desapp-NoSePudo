package adapters

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// ErrPasswordMismatch es lo que Compare devuelve cuando la contraseña no
// corresponde al digest. Es un error del adaptador y no de dominio: quien
// decide que eso significa "credenciales inválidas" es el service.
var ErrPasswordMismatch = errors.New("la contraseña no corresponde al hash")

// MaxPasswordBytes es el límite de bcrypt. Más allá del byte 72 la librería
// ignora el resto en silencio, lo que haría equivalentes a dos contraseñas
// distintas. El DTO lo rechaza en el borde; acá se nombra porque es la razón.
const MaxPasswordBytes = 72

// Password es la implementación concreta del hashing sobre bcrypt. El factor
// de costo llega por constructor desde la configuración inyectada, así que
// este es el único lugar donde se aplica.
type Password struct {
	dummyHash string
	cost      int
}

func NewPassword(cost int) *Password {
	password := &Password{cost: cost}

	// Se precomputa un digest sobre un valor aleatorio, una sola vez al
	// arrancar, para que CompareWithDummy tenga contra qué comparar.
	if hash, err := password.Hash(uuid.NewString()); err == nil {
		password.dummyHash = hash
	}

	return password
}

// CompareWithDummy hace el trabajo de una comparación sin comparar nada útil.
//
// Existe por el canal lateral de tiempo: si el login devolviera enseguida
// cuando la cuenta no existe y tardara los cientos de milisegundos de bcrypt
// cuando sí existe, la duración de la respuesta delataría cuáles direcciones
// están registradas, que es justo lo que FR-003 no quiere revelar. El camino
// de "cuenta inexistente" paga el mismo precio que el de "contraseña
// incorrecta".
func (p *Password) CompareWithDummy(plain string) {
	if p.dummyHash == "" {
		return
	}
	_ = bcrypt.CompareHashAndPassword([]byte(p.dummyHash), []byte(plain))
}

func (p *Password) Hash(plain string) (string, error) {
	digest, err := bcrypt.GenerateFromPassword([]byte(plain), p.cost)
	if err != nil {
		return "", fmt.Errorf("hashear contraseña: %w", err)
	}
	return string(digest), nil
}

// Compare devuelve ErrPasswordMismatch cuando no coinciden, y cualquier otro
// error cuando el hash guardado no se puede interpretar, que es un problema
// distinto y no debería confundirse con una contraseña equivocada.
func (p *Password) Compare(hash, plain string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
	if err == nil {
		return nil
	}

	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return ErrPasswordMismatch
	}

	return fmt.Errorf("comparar contraseña: %w", err)
}
