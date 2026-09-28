package service

import (
	"context"
	"errors"
	"fmt"
	"time"

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
	IssueRefresh(subject int64, sessionID string) (string, adapters.Claims, error)
}

// RefreshTokenRepository es lo que el service necesita para que una credencial
// de renovación sea revocable. Es la diferencia de fondo con la de acceso: esta
// se persiste justamente para poder cortarla antes de que expire.
type RefreshTokenRepository interface {
	Insert(ctx context.Context, token model.RefreshToken) error
	GetByID(ctx context.Context, id string) (model.RefreshToken, error)
	Rotate(ctx context.Context, presentedID string, replacement model.RefreshToken) error
	RevokeFamily(ctx context.Context, familyID string, userID int64) error
	RevokeAllLiveForUser(ctx context.Context, userID int64) error
}

// Clock es el reloj del service, inyectado para que los casos que miran la
// expiración de una fila puedan escribirse.
type Clock func() time.Time

type Auth struct {
	userRepository         UserRepository
	passwordHasher         PasswordHasher
	tokenIssuer            TokenIssuer
	refreshTokenRepository RefreshTokenRepository
	now                    Clock
}

func NewAuthService(
	userRepository UserRepository,
	passwordHasher PasswordHasher,
	tokenIssuer TokenIssuer,
	refreshTokenRepository RefreshTokenRepository,
	now Clock,
) *Auth {
	return &Auth{
		userRepository:         userRepository,
		passwordHasher:         passwordHasher,
		tokenIssuer:            tokenIssuer,
		refreshTokenRepository: refreshTokenRepository,
		now:                    now,
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

	// Cada inicio de sesión abre una familia propia, y es lo que permite cerrar
	// una sesión sin tocar las de los otros dispositivos.
	return a.openSession(ctx, user, uuid.NewString())
}

// openSession emite las dos credenciales de una sesión y persiste la de
// renovación. La comparten el inicio de sesión, que abre una familia nueva, y
// la renovación, que hereda la que ya venía.
func (a *Auth) openSession(ctx context.Context, user model.User, sessionID string) (model.Session, error) {
	accessToken, accessClaims, err := a.tokenIssuer.IssueAccess(user.ID, user.Privilege, sessionID)
	if err != nil {
		return model.Session{}, fmt.Errorf("emitir la credencial de acceso: %w", err)
	}

	refreshToken, refreshClaims, err := a.tokenIssuer.IssueRefresh(user.ID, sessionID)
	if err != nil {
		return model.Session{}, fmt.Errorf("emitir la credencial de renovación: %w", err)
	}

	// La fila se guarda antes de devolver la credencial: una que el sistema no
	// pueda revocar no es aceptable (FR-035).
	err = a.refreshTokenRepository.Insert(ctx, model.RefreshToken{
		ExpiresAt: refreshClaims.ExpiresAt,
		ID:        refreshClaims.ID,
		FamilyID:  sessionID,
		UserID:    user.ID,
	})
	if err != nil {
		return model.Session{}, fmt.Errorf("persistir la credencial de renovación: %w", err)
	}

	return model.Session{
		AccessExpiresAt:  accessClaims.ExpiresAt,
		RefreshExpiresAt: refreshClaims.ExpiresAt,
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
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

// EnsureSuperuser aprovisiona la única cuenta de superusuario, y es el segundo
// camino de creación que existe: Register no puede producir una, porque su DTO
// no tiene campo de privilegio y el nivel está escrito fijo (FR-019, FR-022).
//
// Es idempotente y deliberadamente conservadora: si ya hay una cuenta con ese
// identificador no hace nada, ni siquiera le cambia la contraseña. Eso es lo
// que permite que el contenedor reinicie sin crear una segunda cuenta ni
// fallar, y lo que evita que un reinicio le deshaga en silencio al dueño una
// contraseña que había cambiado.
//
// Vive en el service y no en cmd aunque cmd sea quien la dispara: crear una
// cuenta es normalizar un email, hashear al costo configurado y hacer cumplir
// la unicidad, y esas tres son reglas de negocio. El trabajo de cmd es armar el
// grafo de dependencias y arrancar el servidor, no conocerlas.
func (a *Auth) EnsureSuperuser(ctx context.Context, email, password string) error {
	normalizedEmail := model.NormalizeEmail(email)

	_, err := a.userRepository.GetByEmail(ctx, normalizedEmail)
	switch {
	case err == nil:
		return nil
	case !errors.Is(err, model.ErrUserNotFound):
		return fmt.Errorf("buscar la cuenta de superusuario: %w", err)
	}

	passwordHash, err := a.passwordHasher.Hash(password)
	if err != nil {
		return fmt.Errorf("hashear la contraseña del superusuario: %w", err)
	}

	_, err = a.userRepository.Insert(ctx, model.User{
		Email:        normalizedEmail,
		PasswordHash: passwordHash,
		Privilege:    model.PrivilegeSuperuser,
		Active:       true,
	})
	if err != nil {
		// Si otro proceso la creó entre la consulta y esta escritura, el
		// objetivo igual se cumplió: hay exactamente un superusuario.
		if errors.Is(err, model.ErrEmailTaken) {
			return nil
		}
		return fmt.Errorf("crear la cuenta de superusuario: %w", err)
	}

	return nil
}

// Refresh cambia una credencial de renovación por una sesión nueva.
//
// Los tres datos que recibe salen del actor que el middleware publicó, es decir
// de una credencial ya verificada, y no de nada que el cliente haya mandado. Se
// pasan como argumentos explícitos para que el service no tenga que importar el
// paquete del middleware.
func (a *Auth) Refresh(
	ctx context.Context,
	presentedID string,
	userID int64,
	sessionID string,
) (model.Session, error) {
	presented, err := a.refreshTokenRepository.GetByID(ctx, presentedID)
	if err != nil {
		// Una credencial correctamente firmada cuya fila no existe es una que
		// ya fue consumida, o una emitida con una clave que se perdió. En
		// cualquiera de los dos casos se responde como reuso.
		if errors.Is(err, model.ErrRefreshTokenNotFound) {
			return model.Session{}, a.respondToTheft(ctx, userID)
		}
		return model.Session{}, fmt.Errorf("buscar la credencial de renovación: %w", err)
	}

	// Expirada no es lo mismo que robada: el titular simplemente vuelve a
	// entrar, y cortarle el resto de las sesiones sería castigarlo por dejar
	// pasar una semana.
	if presented.IsExpired(a.now()) {
		return model.Session{}, model.ErrRefreshTokenExpired
	}

	// Usada es evidencia de robo: el poseedor legítimo ya la canjeó, así que
	// quien la presenta ahora es un segundo poseedor (FR-037).
	if presented.IsUsed() {
		return model.Session{}, a.respondToTheft(ctx, userID)
	}

	// Revocada, en cambio, se rechaza y nada más. FR-037 dice "usada o
	// invalidada", pero el cuarto escenario de la historia dice sólo "usada", y
	// el caso borde de la especificación exige que dos sesiones sean
	// independientes: si el teléfono reintenta renovar después de haber cerrado
	// sesión, eso no puede tirarle abajo la sesión del escritorio. Se resuelve
	// a favor de los escenarios porque son más específicos, y porque una
	// credencial revocada que reaparece no es evidencia de un segundo poseedor.
	if presented.IsRevoked() {
		return model.Session{}, model.ErrRefreshTokenRevoked
	}

	// El privilegio se relee de la cuenta en este momento, no se hereda de la
	// credencial anterior: una sesión larga no puede congelar un nivel viejo.
	user, err := a.userRepository.GetByID(ctx, userID)
	if err != nil {
		return model.Session{}, fmt.Errorf("buscar la cuenta a renovar: %w", err)
	}

	accessToken, accessClaims, err := a.tokenIssuer.IssueAccess(user.ID, user.Privilege, sessionID)
	if err != nil {
		return model.Session{}, fmt.Errorf("emitir la credencial de acceso: %w", err)
	}

	refreshToken, refreshClaims, err := a.tokenIssuer.IssueRefresh(user.ID, sessionID)
	if err != nil {
		return model.Session{}, fmt.Errorf("emitir la credencial de renovación: %w", err)
	}

	// La rotación es la que decide de verdad: su UPDATE lleva las condiciones
	// de "está viva" en el WHERE, así que si dos peticiones llegan con la misma
	// credencial sólo una gana. Los chequeos de arriba existen para poder
	// distinguir expirada de consumida, no para tomar la decisión.
	replacement := model.RefreshToken{
		ExpiresAt: refreshClaims.ExpiresAt,
		ID:        refreshClaims.ID,
		FamilyID:  sessionID,
		UserID:    user.ID,
	}

	if err := a.refreshTokenRepository.Rotate(ctx, presentedID, replacement); err != nil {
		if errors.Is(err, model.ErrRefreshTokenReused) {
			return model.Session{}, a.respondToTheft(ctx, userID)
		}
		return model.Session{}, fmt.Errorf("rotar la credencial de renovación: %w", err)
	}

	return model.Session{
		AccessExpiresAt:  accessClaims.ExpiresAt,
		RefreshExpiresAt: refreshClaims.ExpiresAt,
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
	}, nil
}

// respondToTheft corta todas las credenciales de renovación de la cuenta y
// devuelve el reuso.
//
// El alcance es la cuenta entera y no la familia afectada, porque quien tiene
// una credencial robada de una familia puede tener otra, y el costo de
// equivocarse es un inicio de sesión extra.
//
// Si la revocación falla, se devuelve ese error y no el de reuso: dejar
// credenciales vivas después de detectar un robo es peor que responder 500.
func (a *Auth) respondToTheft(ctx context.Context, userID int64) error {
	if err := a.refreshTokenRepository.RevokeAllLiveForUser(ctx, userID); err != nil {
		return fmt.Errorf("revocar las credenciales tras detectar un reuso: %w", err)
	}

	return model.ErrRefreshTokenReused
}

// Logout cierra la sesión que la credencial de acceso presentada nombra.
//
// No recibe parámetros del cliente: la familia sale del claim sid de una
// credencial que el middleware ya verificó. Un valor que el cliente manda es un
// valor que hay que validar, testear y desconfiar, y la forma más barata de
// tratarlo es no aceptarlo.
//
// La credencial de acceso no se invalida: expira sola, y es lo bastante corta
// para que eso alcance.
func (a *Auth) Logout(ctx context.Context, userID int64, sessionID string) error {
	if err := a.refreshTokenRepository.RevokeFamily(ctx, sessionID, userID); err != nil {
		return fmt.Errorf("cerrar la sesión: %w", err)
	}

	return nil
}
