package adapters_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/adapters"
)

// testCost es el mínimo que acepta bcrypt. Los tests corren contra la
// librería real, así que el costo se baja para que no tarden: lo que se está
// probando es el comportamiento del adaptador, no cuánto cuesta hashear.
const testCost = 4

func TestPasswordHashVerifiesAgainstItsOwnPassword(t *testing.T) {
	password := adapters.NewPassword(testCost)

	hash, err := password.Hash("una-contraseña-cualquiera")
	if err != nil {
		t.Fatalf("Hash devolvió error: %v", err)
	}

	if err := password.Compare(hash, "una-contraseña-cualquiera"); err != nil {
		t.Errorf("Compare rechazó la contraseña que generó el hash: %v", err)
	}
}

func TestPasswordCompareRejectsAnotherPassword(t *testing.T) {
	password := adapters.NewPassword(testCost)

	hash, err := password.Hash("la-correcta")
	if err != nil {
		t.Fatalf("Hash devolvió error: %v", err)
	}

	err = password.Compare(hash, "la-incorrecta")
	if !errors.Is(err, adapters.ErrPasswordMismatch) {
		t.Errorf("Compare devolvió %v, se esperaba ErrPasswordMismatch", err)
	}
}

// El salt hace que el mismo texto dé digests distintos. Es lo que evita que
// dos cuentas con la misma contraseña se delaten entre sí.
func TestPasswordHashesOfTheSamePasswordDiffer(t *testing.T) {
	password := adapters.NewPassword(testCost)

	first, err := password.Hash("la-misma")
	if err != nil {
		t.Fatalf("Hash devolvió error: %v", err)
	}
	second, err := password.Hash("la-misma")
	if err != nil {
		t.Fatalf("Hash devolvió error: %v", err)
	}

	if first == second {
		t.Error("dos hashes de la misma contraseña son idénticos: no se está salando")
	}
	if err := password.Compare(second, "la-misma"); err != nil {
		t.Errorf("el segundo hash no verifica: %v", err)
	}
}

// el requerimiento: del registro guardado no se puede recuperar la contraseña original.
func TestPasswordHashDoesNotContainThePlaintext(t *testing.T) {
	password := adapters.NewPassword(testCost)
	const plain = "contraseña-reconocible"

	hash, err := password.Hash(plain)
	if err != nil {
		t.Fatalf("Hash devolvió error: %v", err)
	}

	if strings.Contains(hash, plain) {
		t.Error("el digest contiene la contraseña en claro")
	}
}

// El digest de bcrypt mide exactamente 60 caracteres, que es lo que justifica
// el VARCHAR(60) de la columna password_hash.
func TestPasswordHashIsSixtyCharacters(t *testing.T) {
	password := adapters.NewPassword(testCost)

	hash, err := password.Hash("cualquiera")
	if err != nil {
		t.Fatalf("Hash devolvió error: %v", err)
	}

	if len(hash) != 60 {
		t.Errorf("el digest mide %d caracteres, la columna está dimensionada para 60", len(hash))
	}
}

func TestPasswordHashRejectsAPasswordLongerThanBcryptAccepts(t *testing.T) {
	password := adapters.NewPassword(testCost)

	_, err := password.Hash(strings.Repeat("a", adapters.MaxPasswordBytes+1))
	if err == nil {
		t.Error("bcrypt no acepta más de 72 bytes: Hash debería fallar en vez de truncar")
	}
}

// Un hash ilegible es un problema distinto de una contraseña equivocada, y no
// debe confundirse con ella.
func TestPasswordCompareDistinguishesACorruptHashFromAMismatch(t *testing.T) {
	password := adapters.NewPassword(testCost)

	err := password.Compare("esto-no-es-un-digest-de-bcrypt", "cualquiera")
	if err == nil {
		t.Fatal("Compare aceptó un hash ilegible")
	}
	if errors.Is(err, adapters.ErrPasswordMismatch) {
		t.Error("un hash corrupto se reportó como contraseña incorrecta")
	}
}
