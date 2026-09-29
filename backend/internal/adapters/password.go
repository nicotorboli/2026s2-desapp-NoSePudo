package adapters

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)


var ErrPasswordMismatch = errors.New("la contraseña no corresponde al hash")

// MaxPasswordBytes es el límite de bcrypt.
const MaxPasswordBytes = 72

type Password struct {
	dummyHash string
	cost int
}

func NewPassword(cost int) *Password {
	password := &Password{cost: cost}

	if hash, err := password.Hash(uuid.NewString()); err == nil {
		password.dummyHash = hash
	}

	return password
}


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

