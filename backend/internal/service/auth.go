package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/adapters"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

// UserRepository es lo que el service necesita de la persistencia de cuentas.
// La interfaz vive acá, del lado de quien la consume, así que el service se
// testea con un mock y sin base.
type UserRepository interface {
	Insert(ctx context.Context, user model.User) (model.User, error)
	GetByEmail(ctx context.Context, email string) (model.User, error)
	GetByID(ctx context.Context, id int64) (model.User, error)
}

// PasswordHasher es la capacidad de hashear y comparar que el service pide
// prestada al adaptador, para no pagar un bcrypt real en cada caso de test ni
// depender de la librería.
type PasswordHasher interface {
	Hash(plain string) (string, error)
	Compare(hash, plain string) error

	// CompareWithDummy cuesta lo mismo que Compare pero no compara contra
	// nada. Lo usa el camino de "la cuenta no existe" para que ese caso no se
	// distinga del de "la contraseña es incorrecta" por lo que tarda.
	CompareWithDummy(plain string)
}

// TokenIssuer es la capacidad de emitir credenciales firmadas. El service no
// sabe que son JWT ni cómo se firman: pide una credencial para una cuenta y
// recibe la cadena y cuándo deja de valer.
type TokenIssuer interface {
	IssueAccess(subject int64, privilege model.PrivilegeLevel, sessionID string) (string, adapters.Claims, error)
}

type Auth struct {
	userRepository UserRepository
	passwordHasher PasswordHasher
	tokenIssuer    TokenIssuer
}

func NewAuthService(
	userRepository UserRepository,
	passwordHasher PasswordHasher,
	tokenIssuer TokenIssuer,
) *Auth {
	return &Auth{
		userRepository: userRepository,
		passwordHasher: passwordHasher,
		tokenIssuer:    tokenIssuer,
	}
}

// Login verifica las credenciales y emite la sesión.
//
// Los dos modos de fallar — la cuenta no existe, la contraseña no coincide —
// devuelven el mismo error y pagan el mismo costo de hashing, así que ni la
// respuesta ni el tiempo dicen cuáles direcciones están registradas (FR-003).
func (a *Auth) Login(ctx context.Context, email, password string) (model.Session, error) {
	user, err := a.userRepository.GetByEmail(ctx, model.NormalizeEmail(email))
	if err != nil {
		if errors.Is(err, model.ErrUserNotFound) {
			a.passwordHasher.CompareWithDummy(password)
			return model.Session{}, model.ErrInvalidCredentials
		}
		return model.Session{}, fmt.Errorf("buscar la cuenta: %w", err)
	}

	if err = a.passwordHasher.Compare(user.PasswordHash, password); err != nil {
		return model.Session{}, model.ErrInvalidCredentials
	}

	// Cada inicio de sesión abre una familia propia. Todavía no hay nada que
	// la consuma, pero es lo que va a permitir cerrar una sesión sin tocar las
	// de los otros dispositivos, y ponerla desde ahora evita cambiarle los
	// claims a la credencial más adelante.
	sessionID := uuid.NewString()

	accessToken, claims, err := a.tokenIssuer.IssueAccess(user.ID, user.Privilege, sessionID)
	if err != nil {
		return model.Session{}, fmt.Errorf("emitir la credencial de acceso: %w", err)
	}

	return model.Session{
		AccessExpiresAt: claims.ExpiresAt,
		AccessToken:     accessToken,
	}, nil
}

// Register da de alta una cuenta de usuario común.
//
// El nivel de privilegio está escrito acá y no llega de ningún lado: el DTO no
// tiene campo para él y este método no recibe parámetro. Esa es la garantía de
// FR-022, y es también por qué el superusuario necesita su propio camino de
// creación en vez de un flag en este.
func (a *Auth) Register(ctx context.Context, email, password string) (model.User, error) {
	normalizedEmail := model.NormalizeEmail(email)

	if err := a.ensureEmailIsAvailable(ctx, normalizedEmail); err != nil {
		return model.User{}, err
	}

	passwordHash, err := a.passwordHasher.Hash(password)
	if err != nil {
		return model.User{}, fmt.Errorf("hashear la contraseña: %w", err)
	}

	user, err := a.userRepository.Insert(ctx, model.User{
		Email:        normalizedEmail,
		PasswordHash: passwordHash,
		Privilege:    model.PrivilegeUser,
		Active:       true,
	})
	if err != nil {
		return model.User{}, fmt.Errorf("crear la cuenta: %w", err)
	}

	return user, nil
}

// ensureEmailIsAvailable consulta antes de escribir para poder devolver un
// error útil, pero no es lo que garantiza la unicidad: entre esta consulta y
// el INSERT hay una carrera, y quien la cierra es el índice único de la
// columna. Por eso Insert también traduce la violación a ErrEmailTaken.
func (a *Auth) ensureEmailIsAvailable(ctx context.Context, normalizedEmail string) error {
	_, err := a.userRepository.GetByEmail(ctx, normalizedEmail)

	switch {
	case err == nil:
		return model.ErrEmailTaken
	case errors.Is(err, model.ErrUserNotFound):
		return nil
	default:
		return fmt.Errorf("verificar si el email está disponible: %w", err)
	}
}
