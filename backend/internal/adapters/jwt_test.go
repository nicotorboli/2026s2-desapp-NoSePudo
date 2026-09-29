package adapters_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/adapters"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

const (
	testSecret     = "0123456789abcdef0123456789abcdef"
	testAccessTTL  = 15 * time.Minute
	testRefreshTTL = 168 * time.Hour
)

// baseInstant está truncado al segundo porque el claim exp se serializa con
// precisión de segundo: un "tick" acá tiene que ser un segundo, o el borde no
// se estaría probando donde realmente cae.
var baseInstant = time.Date(2026, time.September, 28, 12, 0, 0, 0, time.UTC)

const tick = time.Second

// clock es un reloj movible: se emite en un instante y se verifica en otro,
// que es lo que hace escribible el caso "justo en exp".
type clock struct {
	instant time.Time
}

func (c *clock) now() time.Time { return c.instant }

func newJWTAt(instant time.Time) (*adapters.JWT, *clock) {
	c := &clock{instant: instant}
	return adapters.NewJWT(testSecret, testAccessTTL, testRefreshTTL, c.now), c
}

func TestJWTAccessTokenRoundTripsItsClaims(t *testing.T) {
	issuer, _ := newJWTAt(baseInstant)

	raw, issued, err := issuer.IssueAccess(42, model.PrivilegeSuperuser, "sesion-abc")
	if err != nil {
		t.Fatalf("IssueAccess devolvió error: %v", err)
	}

	verified, err := issuer.Verify(raw)
	if err != nil {
		t.Fatalf("Verify devolvió error: %v", err)
	}

	if verified.Subject != 42 {
		t.Errorf("Subject = %d, se esperaba 42", verified.Subject)
	}
	if verified.Privilege != model.PrivilegeSuperuser {
		t.Errorf("Privilege = %v, se esperaba superuser", verified.Privilege)
	}
	if verified.SessionID != "sesion-abc" {
		t.Errorf("SessionID = %q", verified.SessionID)
	}
	if verified.Kind != adapters.KindAccess {
		t.Errorf("Kind = %v, se esperaba access", verified.Kind)
	}
	if verified.ID != issued.ID {
		t.Errorf("el jti verificado (%q) difiere del emitido (%q)", verified.ID, issued.ID)
	}
	if !verified.ExpiresAt.Equal(baseInstant.Add(testAccessTTL)) {
		t.Errorf("ExpiresAt = %v, se esperaba %v", verified.ExpiresAt, baseInstant.Add(testAccessTTL))
	}
}

// El refresh token no lleva priv: no da acceso a ningún recurso.
func TestJWTRefreshTokenCarriesNoPrivilege(t *testing.T) {
	issuer, _ := newJWTAt(baseInstant)

	raw, _, err := issuer.IssueRefresh(42, "sesion-abc")
	if err != nil {
		t.Fatalf("IssueRefresh devolvió error: %v", err)
	}

	verified, err := issuer.Verify(raw)
	if err != nil {
		t.Fatalf("Verify devolvió error: %v", err)
	}

	if verified.Kind != adapters.KindRefresh {
		t.Errorf("Kind = %v, se esperaba refresh", verified.Kind)
	}
	if verified.Privilege != model.PrivilegeUnknown {
		t.Errorf("Privilege = %v: el refresh token no debería llevar privilegio", verified.Privilege)
	}
	if !verified.ExpiresAt.Equal(baseInstant.Add(testRefreshTTL)) {
		t.Errorf("ExpiresAt = %v, se esperaba %v", verified.ExpiresAt, baseInstant.Add(testRefreshTTL))
	}
}

func TestJWTEachTokenGetsItsOwnIdentifier(t *testing.T) {
	issuer, _ := newJWTAt(baseInstant)

	_, first, err := issuer.IssueAccess(42, model.PrivilegeUser, "sesion-abc")
	if err != nil {
		t.Fatalf("IssueAccess devolvió error: %v", err)
	}
	_, second, err := issuer.IssueAccess(42, model.PrivilegeUser, "sesion-abc")
	if err != nil {
		t.Fatalf("IssueAccess devolvió error: %v", err)
	}

	if first.ID == second.ID {
		t.Error("dos credenciales comparten el jti: el refresh token no se podría revocar individualmente")
	}
}

// El primer caso borde: un instante antes, justo en, y un instante
// después de exp. El límite queda definido como "válida estrictamente antes de
// exp", y se comporta igual en los tres puntos.
func TestJWTExpiryBoundary(t *testing.T) {
	issuer, clk := newJWTAt(baseInstant)

	raw, issued, err := issuer.IssueAccess(42, model.PrivilegeUser, "sesion-abc")
	if err != nil {
		t.Fatalf("IssueAccess devolvió error: %v", err)
	}
	expiry := issued.ExpiresAt

	cases := []struct {
		instant     time.Time
		name        string
		wantExpired bool
	}{
		{expiry.Add(-tick), "un tick antes de exp", false},
		{expiry, "justo en exp", true},
		{expiry.Add(tick), "un tick después de exp", true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			clk.instant = c.instant

			_, err := issuer.Verify(raw)

			if !c.wantExpired {
				if err != nil {
					t.Errorf("Verify devolvió %v, se esperaba que aceptara la credencial", err)
				}
				return
			}
			if !errors.Is(err, adapters.ErrTokenExpired) {
				t.Errorf("Verify devolvió %v, se esperaba ErrTokenExpired", err)
			}
		})
	}
}

// el requerimiento: la allowlist es explícita, así que una credencial no se verifica
// según el algoritmo que ella misma declara.
func TestJWTRejectsAnAlgorithmOutsideTheAllowlist(t *testing.T) {
	issuer, _ := newJWTAt(baseInstant)

	claims := jwt.RegisteredClaims{
		Subject:   "42",
		ID:        "un-jti",
		IssuedAt:  jwt.NewNumericDate(baseInstant),
		ExpiresAt: jwt.NewNumericDate(baseInstant.Add(testAccessTTL)),
	}

	t.Run("none", func(t *testing.T) {
		raw, err := jwt.NewWithClaims(jwt.SigningMethodNone, claims).SignedString(jwt.UnsafeAllowNoneSignatureType)
		if err != nil {
			t.Fatalf("no se pudo construir el token de prueba: %v", err)
		}

		if _, err := issuer.Verify(raw); !errors.Is(err, adapters.ErrTokenInvalid) {
			t.Errorf("Verify devolvió %v, se esperaba ErrTokenInvalid para alg=none", err)
		}
	})

	t.Run("HS512 con el mismo secreto", func(t *testing.T) {
		raw, err := jwt.NewWithClaims(jwt.SigningMethodHS512, claims).SignedString([]byte(testSecret))
		if err != nil {
			t.Fatalf("no se pudo construir el token de prueba: %v", err)
		}

		if _, err := issuer.Verify(raw); !errors.Is(err, adapters.ErrTokenInvalid) {
			t.Errorf("Verify devolvió %v, se esperaba ErrTokenInvalid para HS512", err)
		}
	})
}

func TestJWTRejectsAnAlteredToken(t *testing.T) {
	issuer, _ := newJWTAt(baseInstant)

	raw, _, err := issuer.IssueAccess(42, model.PrivilegeUser, "sesion-abc")
	if err != nil {
		t.Fatalf("IssueAccess devolvió error: %v", err)
	}

	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		t.Fatalf("un JWS compacto tiene tres partes, se obtuvieron %d", len(parts))
	}

	cases := map[string]string{
		"payload alterado":       parts[0] + "." + parts[1][:len(parts[1])-2] + "XY." + parts[2],
		"firma alterada":         parts[0] + "." + parts[1] + "." + parts[2][:len(parts[2])-2] + "XY",
		"firma recortada":        parts[0] + "." + parts[1] + ".",
		"partes de más":          raw + ".extra",
		"no es un JWT":           "esto-no-es-una-credencial",
		"cadena vacía":           "",
		"sólo el header":         parts[0],
		"firmado con otra clave": signedWithAnotherKey(t),
	}

	for name, altered := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := issuer.Verify(altered); err == nil {
				t.Error("Verify aceptó una credencial que no debería haber aceptado")
			}
		})
	}
}

func signedWithAnotherKey(t *testing.T) string {
	t.Helper()

	other := adapters.NewJWT("ffffffffffffffffffffffffffffffff", testAccessTTL, testRefreshTTL, func() time.Time { return baseInstant })
	raw, _, err := other.IssueAccess(42, model.PrivilegeUser, "sesion-abc")
	if err != nil {
		t.Fatalf("no se pudo construir el token de prueba: %v", err)
	}

	return raw
}

// el requerimiento: un priv que no se reconoce no es superusuario, es insuficiente.
func TestJWTUnrecognizedPrivilegeBecomesUnknown(t *testing.T) {
	raw := signedWithClaims(t, tokenClaimsFixture{
		typ:  "access",
		priv: "root",
		sub:  "42",
	})

	issuer, _ := newJWTAt(baseInstant)

	verified, err := issuer.Verify(raw)
	if err != nil {
		t.Fatalf("Verify devolvió error: %v", err)
	}
	if verified.Privilege != model.PrivilegeUnknown {
		t.Errorf("Privilege = %v, se esperaba PrivilegeUnknown para un priv irreconocible", verified.Privilege)
	}
	if verified.Privilege.Satisfies(model.PrivilegeUser) {
		t.Error("un priv irreconocible satisface PrivilegeUser")
	}
}

func TestJWTRejectsAnUnrecognizedKind(t *testing.T) {
	raw := signedWithClaims(t, tokenClaimsFixture{typ: "session", sub: "42"})

	issuer, _ := newJWTAt(baseInstant)

	if _, err := issuer.Verify(raw); !errors.Is(err, adapters.ErrTokenInvalid) {
		t.Errorf("Verify devolvió %v, se esperaba ErrTokenInvalid para un typ desconocido", err)
	}
}

func TestJWTRejectsASubjectThatIsNotAnAccountID(t *testing.T) {
	raw := signedWithClaims(t, tokenClaimsFixture{typ: "access", sub: "nico@nosepudo.ar"})

	issuer, _ := newJWTAt(baseInstant)

	if _, err := issuer.Verify(raw); !errors.Is(err, adapters.ErrTokenInvalid) {
		t.Errorf("Verify devolvió %v, se esperaba ErrTokenInvalid para un sub que no es un id", err)
	}
}

func TestJWTRejectsATokenWithoutExpiry(t *testing.T) {
	raw := signedWithClaims(t, tokenClaimsFixture{typ: "access", sub: "42", noExpiry: true})

	issuer, _ := newJWTAt(baseInstant)

	if _, err := issuer.Verify(raw); err == nil {
		t.Error("Verify aceptó una credencial sin instante de expiración")
	}
}

// tokenClaimsFixture arma credenciales a mano para los casos que el emisor
// nunca produciría pero que un atacante sí podría presentar.
type tokenClaimsFixture struct {
	typ      string
	priv     string
	sub      string
	noExpiry bool
}

func signedWithClaims(t *testing.T, fixture tokenClaimsFixture) string {
	t.Helper()

	claims := jwt.MapClaims{
		"typ": fixture.typ,
		"sub": fixture.sub,
		"sid": "sesion-abc",
		"jti": "un-jti",
		"iat": jwt.NewNumericDate(baseInstant),
	}
	if fixture.priv != "" {
		claims["priv"] = fixture.priv
	}
	if !fixture.noExpiry {
		claims["exp"] = jwt.NewNumericDate(baseInstant.Add(testAccessTTL))
	}

	raw, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("no se pudo construir el token de prueba: %v", err)
	}

	return raw
}
