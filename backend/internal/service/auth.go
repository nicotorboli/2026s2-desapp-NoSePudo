package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/adapters"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/logger"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)


type UserRepository interface {
	Insert(ctx context.Context, user model.User) (model.User, error)
	GetByEmail(ctx context.Context, email string) (model.User, error)
	GetByID(ctx context.Context, id int64) (model.User, error)
}


type PasswordHasher interface {
	Hash(plain string) (string, error)
	Compare(hash, plain string) error
	CompareWithDummy(plain string)
}

type TokenIssuer interface {
	IssueAccess(subject int64, privilege model.PrivilegeLevel, sessionID string) (string, adapters.Claims, error)
	IssueRefresh(subject int64, sessionID string) (string, adapters.Claims, error)
}

type RefreshTokenRepository interface {
	Insert(ctx context.Context, token model.RefreshToken) error
	GetByID(ctx context.Context, id string) (model.RefreshToken, error)
	Rotate(ctx context.Context, presentedID string, replacement model.RefreshToken) error
	RevokeFamily(ctx context.Context, familyID string, userID int64) error
	RevokeAllLiveForUser(ctx context.Context, userID int64) error
}


type Clock func() time.Time

type Auth struct {
	userRepository UserRepository
	passwordHasher PasswordHasher
	tokenIssuer TokenIssuer
	refreshTokenRepository RefreshTokenRepository
	now Clock
}

func NewAuthService(
	userRepository UserRepository,
	passwordHasher PasswordHasher,
	tokenIssuer TokenIssuer,
	refreshTokenRepository RefreshTokenRepository,
	now Clock,
) *Auth {
	return &Auth{
		userRepository: userRepository,
		passwordHasher: passwordHasher,
		tokenIssuer: tokenIssuer,
		refreshTokenRepository: refreshTokenRepository,
		now: now,
	}
}


func (a *Auth) Login(ctx context.Context, email, password string) (model.Session, error) {
	user, err := a.userRepository.GetByEmail(ctx, model.NormalizeEmail(email))
	if err != nil {
		if errors.Is(err, model.ErrUserNotFound) {
			a.passwordHasher.CompareWithDummy(password)

			logRefusedSignIn(ctx, "no existe una cuenta con ese identificador")

			return model.Session{}, model.ErrInvalidCredentials
		}
		return model.Session{}, fmt.Errorf("buscar la cuenta: %w", err)
	}

	if err = a.passwordHasher.Compare(user.PasswordHash, password); err != nil {
		logRefusedSignIn(ctx, "la contraseña no coincide", "actor", user.ID)

		return model.Session{}, model.ErrInvalidCredentials
	}

	// Cada inicio de sesión abre una familia propia, y es lo que permite cerrar
	// una sesión sin tocar las de los otros dispositivos.
	return a.openSession(ctx, user, uuid.NewString())
}

func logRefusedSignIn(ctx context.Context, reason string, attributes ...any) {
	logger.FromContext(ctx).Warn(
		"sign-in refused",
		append([]any{"operation", "login", "reason", reason}, attributes...)...,
	)
}

func (a *Auth) openSession(ctx context.Context, user model.User, sessionID string) (model.Session, error) {
	accessToken, accessClaims, err := a.tokenIssuer.IssueAccess(user.ID, user.Privilege, sessionID)
	if err != nil {
		return model.Session{}, fmt.Errorf("emitir la credencial de acceso: %w", err)
	}

	refreshToken, refreshClaims, err := a.tokenIssuer.IssueRefresh(user.ID, sessionID)
	if err != nil {
		return model.Session{}, fmt.Errorf("emitir la credencial de renovación: %w", err)
	}

	err = a.refreshTokenRepository.Insert(ctx, model.RefreshToken{
		ExpiresAt: refreshClaims.ExpiresAt,
		ID: refreshClaims.ID,
		FamilyID: sessionID,
		UserID: user.ID,
	})
	if err != nil {
		return model.Session{}, fmt.Errorf("persistir la credencial de renovación: %w", err)
	}

	return model.Session{
		AccessExpiresAt: accessClaims.ExpiresAt,
		RefreshExpiresAt: refreshClaims.ExpiresAt,
		AccessToken: accessToken,
		RefreshToken: refreshToken,
	}, nil
}

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
		Email: normalizedEmail,
		PasswordHash: passwordHash,
		Privilege: model.PrivilegeUser,
		Active: true,
	})
	if err != nil {
		return model.User{}, fmt.Errorf("crear la cuenta: %w", err)
	}

	return user, nil
}


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
		Email: normalizedEmail,
		PasswordHash: passwordHash,
		Privilege: model.PrivilegeSuperuser,
		Active: true,
	})
	if err != nil {

		if errors.Is(err, model.ErrEmailTaken) {
			return nil
		}
		return fmt.Errorf("crear la cuenta de superusuario: %w", err)
	}

	return nil
}


func (a *Auth) Refresh(
	ctx context.Context,
	presentedID string,
	userID int64,
	sessionID string,
) (model.Session, error) {
	presented, err := a.refreshTokenRepository.GetByID(ctx, presentedID)
	if err != nil {

		if errors.Is(err, model.ErrRefreshTokenNotFound) {
			return model.Session{}, a.respondToTheft(ctx, userID)
		}
		return model.Session{}, fmt.Errorf("buscar la credencial de renovación: %w", err)
	}


	if presented.IsExpired(a.now()) {
		return model.Session{}, model.ErrRefreshTokenExpired
	}

	if presented.IsUsed() {
		return model.Session{}, a.respondToTheft(ctx, userID)
	}


	if presented.IsRevoked() {
		return model.Session{}, model.ErrRefreshTokenRevoked
	}

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
	// credencial sólo una gana.
	replacement := model.RefreshToken{
		ExpiresAt: refreshClaims.ExpiresAt,
		ID: refreshClaims.ID,
		FamilyID: sessionID,
		UserID: user.ID,
	}

	if err := a.refreshTokenRepository.Rotate(ctx, presentedID, replacement); err != nil {
		if errors.Is(err, model.ErrRefreshTokenReused) {
			return model.Session{}, a.respondToTheft(ctx, userID)
		}
		return model.Session{}, fmt.Errorf("rotar la credencial de renovación: %w", err)
	}

	return model.Session{
		AccessExpiresAt: accessClaims.ExpiresAt,
		RefreshExpiresAt: refreshClaims.ExpiresAt,
		AccessToken: accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (a *Auth) respondToTheft(ctx context.Context, userID int64) error {
	logger.FromContext(ctx).Warn(
		"refresh credential reused",
		"operation", "refresh",
		"actor", userID,
		"reason", "se presentó una credencial de renovación ya consumida",
	)

	if err := a.refreshTokenRepository.RevokeAllLiveForUser(ctx, userID); err != nil {
		return fmt.Errorf("revocar las credenciales tras detectar un reuso: %w", err)
	}

	return model.ErrRefreshTokenReused
}

func (a *Auth) Logout(ctx context.Context, userID int64, sessionID string) error {
	if err := a.refreshTokenRepository.RevokeFamily(ctx, sessionID, userID); err != nil {
		return fmt.Errorf("cerrar la sesión: %w", err)
	}

	return nil
}

