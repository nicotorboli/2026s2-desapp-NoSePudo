package adapters

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

type TokenKind uint8

const (
	KindUnknown TokenKind = iota
	KindAccess
	KindRefresh
)

const (
	kindAccessName  = "access"
	kindRefreshName = "refresh"
)

func (k TokenKind) String() string {
	switch k {
	case KindAccess:
		return kindAccessName
	case KindRefresh:
		return kindRefreshName
	default:
		return "unknown"
	}
}

func parseTokenKind(s string) TokenKind {
	switch s {
	case kindAccessName:
		return KindAccess
	case kindRefreshName:
		return KindRefresh
	default:
		return KindUnknown
	}
}

var (
	ErrTokenInvalid = errors.New("la credencial no es válida")
	ErrTokenExpired = errors.New("la credencial expiró")
)

type Claims struct {
	IssuedAt  time.Time
	ExpiresAt time.Time
	ID        string
	SessionID string
	Subject   int64
	Privilege model.PrivilegeLevel
	Kind      TokenKind
}

// tokenClaims es la forma serializada, con los nombres de claim que fija el
// data-model.
type tokenClaims struct {
	Kind      string `json:"typ"`
	Privilege string `json:"priv,omitempty"`
	SessionID string `json:"sid"`
	jwt.RegisteredClaims
}

// JWT emite y verifica las credenciales con HS256.

type JWT struct {
	now        func() time.Time
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewJWT(secret string, accessTTL, refreshTTL time.Duration, now func() time.Time) *JWT {
	return &JWT{
		now:        now,
		secret:     []byte(secret),
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

// IssueAccess emite la credencial de acceso. Lleva el nivel de privilegio
// porque es la que gobierna el acceso a los recursos.
func (j *JWT) IssueAccess(subject int64, privilege model.PrivilegeLevel, sessionID string) (string, Claims, error) {
	return j.issue(subject, privilege, sessionID, KindAccess, j.accessTTL)
}

// IssueRefresh emite la credencial de renovación.
func (j *JWT) IssueRefresh(subject int64, sessionID string) (string, Claims, error) {
	return j.issue(subject, model.PrivilegeUnknown, sessionID, KindRefresh, j.refreshTTL)
}

func (j *JWT) issue(
	subject int64,
	privilege model.PrivilegeLevel,
	sessionID string,
	kind TokenKind,
	ttl time.Duration,
) (string, Claims, error) {
	issuedAt := j.now()
	claims := Claims{
		IssuedAt:  issuedAt,
		ExpiresAt: issuedAt.Add(ttl),
		ID:        uuid.NewString(),
		SessionID: sessionID,
		Subject:   subject,
		Privilege: privilege,
		Kind:      kind,
	}

	serialized := tokenClaims{
		Kind:      kind.String(),
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(subject, 10),
			ID:        claims.ID,
			IssuedAt:  jwt.NewNumericDate(claims.IssuedAt),
			ExpiresAt: jwt.NewNumericDate(claims.ExpiresAt),
		},
	}
	if kind == KindAccess {
		serialized.Privilege = privilege.String()
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, serialized).SignedString(j.secret)
	if err != nil {
		return "", Claims{}, fmt.Errorf("firmar credencial: %w", err)
	}

	return signed, claims, nil
}

// Verify comprueba firma y expiración y recién entonces devuelve lo que la
// credencial afirma. Todo lo que falle es ErrTokenInvalid o ErrTokenExpired
func (j *JWT) Verify(raw string) (Claims, error) {
	var parsed tokenClaims

	_, err := jwt.ParseWithClaims(
		raw,
		&parsed,
		func(*jwt.Token) (any, error) { return j.secret, nil },
		// La allowlist explícita es lo que impide que una credencial se
		// verifique según el algoritmo que ella misma declara.
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithTimeFunc(j.now),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return Claims{}, ErrTokenExpired
		}
		return Claims{}, fmt.Errorf("%w: %w", ErrTokenInvalid, err)
	}

	return claimsFrom(parsed)
}

func claimsFrom(parsed tokenClaims) (Claims, error) {
	subject, err := strconv.ParseInt(parsed.Subject, 10, 64)
	if err != nil {
		return Claims{}, fmt.Errorf("%w: el claim sub no es un identificador de cuenta", ErrTokenInvalid)
	}

	kind := parseTokenKind(parsed.Kind)
	if kind == KindUnknown {
		return Claims{}, fmt.Errorf("%w: el claim typ no se reconoce", ErrTokenInvalid)
	}

	if parsed.IssuedAt == nil || parsed.ExpiresAt == nil {
		return Claims{}, fmt.Errorf("%w: faltan los instantes de emisión o de expiración", ErrTokenInvalid)
	}

	return Claims{
		IssuedAt:  parsed.IssuedAt.Time,
		ExpiresAt: parsed.ExpiresAt.Time,
		ID:        parsed.ID,
		SessionID: parsed.SessionID,
		Subject:   subject,
		Privilege: model.ParsePrivilege(parsed.Privilege),
		Kind:      kind,
	}, nil
}
