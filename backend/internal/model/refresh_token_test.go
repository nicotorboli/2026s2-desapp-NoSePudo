package model_test

import (
	"testing"
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

var tokenIssuedAt = time.Date(2026, time.September, 28, 12, 0, 0, 0, time.UTC)

func aLiveToken() model.RefreshToken {
	return model.RefreshToken{
		IssuedAt:  tokenIssuedAt,
		ExpiresAt: tokenIssuedAt.Add(168 * time.Hour),
		ID:        "un-jti",
		FamilyID:  "una-familia",
		UserID:    42,
	}
}

func instant(t time.Time) *time.Time { return &t }

// El límite de expiración, un instante antes, justo en, y un instante después.
// Queda definido igual que el de la credencial de acceso: viva estrictamente
// antes de expires_at.
func TestRefreshTokenExpiryBoundary(t *testing.T) {
	token := aLiveToken()
	const tick = time.Nanosecond

	cases := []struct {
		now      time.Time
		name     string
		wantLive bool
	}{
		{token.ExpiresAt.Add(-tick), "un instante antes de expirar", true},
		{token.ExpiresAt, "justo en el instante de expiración", false},
		{token.ExpiresAt.Add(tick), "un instante después de expirar", false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := token.IsLive(c.now); got != c.wantLive {
				t.Errorf("IsLive = %v, se esperaba %v", got, c.wantLive)
			}
			if got := token.IsExpired(c.now); got == c.wantLive {
				t.Errorf("IsExpired = %v, contradice a IsLive", got)
			}
		})
	}
}

// Usada y revocada matan la credencial cada una por su cuenta, sin depender de
// la otra ni del reloj.
func TestRefreshTokenIsLiveRequiresAllThreeConditions(t *testing.T) {
	used := tokenIssuedAt.Add(time.Hour)

	cases := []struct {
		mutate   func(*model.RefreshToken)
		name     string
		wantLive bool
	}{
		{func(*model.RefreshToken) {}, "recién emitida", true},
		{func(token *model.RefreshToken) { token.UsedAt = instant(used) }, "usada", false},
		{func(token *model.RefreshToken) { token.RevokedAt = instant(used) }, "revocada", false},
		{func(token *model.RefreshToken) {
			token.UsedAt = instant(used)
			token.RevokedAt = instant(used)
		}, "usada y revocada", false},
	}

	now := tokenIssuedAt.Add(2 * time.Hour)

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			token := aLiveToken()
			c.mutate(&token)

			if got := token.IsLive(now); got != c.wantLive {
				t.Errorf("IsLive = %v, se esperaba %v", got, c.wantLive)
			}
		})
	}
}

// La diferencia entre "expiró" y "ya se consumió" es la que decide si el
// titular sólo tiene que volver a entrar o si hay que cortar toda la cuenta.
func TestRefreshTokenDistinguishesExpiredFromConsumed(t *testing.T) {
	now := tokenIssuedAt.Add(time.Hour)

	expired := aLiveToken()
	expired.ExpiresAt = tokenIssuedAt.Add(time.Minute)

	consumed := aLiveToken()
	consumed.UsedAt = instant(now)

	if !expired.IsExpired(now) {
		t.Error("una credencial vencida no se reporta como expirada")
	}
	if expired.IsConsumed() {
		t.Error("una credencial vencida se reporta como consumida: dispararía la respuesta al robo sin motivo")
	}

	if !consumed.IsConsumed() {
		t.Error("una credencial usada no se reporta como consumida")
	}
	if consumed.IsExpired(now) {
		t.Error("una credencial usada se reporta como expirada")
	}
}

// Una credencial usada sigue existiendo, y eso es a propósito: borrar la fila
// haría que un reuso fuera indistinguible de un token que nunca existió, y la
// respuesta al robo no podría dispararse nunca.
func TestRefreshTokenUsedIsTerminalAndVisible(t *testing.T) {
	token := aLiveToken()
	token.UsedAt = instant(tokenIssuedAt.Add(time.Hour))

	if token.ID == "" {
		t.Error("la credencial usada perdió su identificador")
	}
	if token.FamilyID == "" {
		t.Error("la credencial usada perdió su familia")
	}
	if !token.IsConsumed() {
		t.Error("la credencial usada no queda marcada como consumida")
	}
}

func TestRefreshTokenErrorsAreDistinct(t *testing.T) {
	errs := []error{
		model.ErrRefreshTokenReused,
		model.ErrRefreshTokenExpired,
		model.ErrRefreshTokenNotFound,
	}

	for i, first := range errs {
		for j, second := range errs {
			if i != j && first == second { //nolint:errorlint // se compara la identidad de dos centinelas
				t.Errorf("los centinelas %d y %d son el mismo error", i, j)
			}
		}
	}
}
