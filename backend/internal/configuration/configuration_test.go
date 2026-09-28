package configuration_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/configuration"
)

// usableSecret tiene exactamente los 32 bytes mínimos.
const usableSecret = "0123456789abcdef0123456789abcdef"

// withSecret deja el entorno con un secreto usable y nada más, que es el punto
// de partida de casi todos los casos. t.Setenv restaura al terminar el test.
func withSecret(t *testing.T) {
	t.Helper()
	t.Setenv("NSP_JWT_SECRET", usableSecret)
}

// FR-032 y SC-009: sin un secreto usable el servicio no arranca.
func TestLoadCfgRejectsAnUnusableSecret(t *testing.T) {
	cases := []struct {
		name   string
		secret string
	}{
		{"ausente", ""},
		{"de un carácter", "x"},
		{"de 31 bytes, uno menos que el mínimo", strings.Repeat("a", 31)},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv("NSP_JWT_SECRET", c.secret)

			cfg, err := configuration.LoadCfg()

			if err == nil {
				t.Fatal("LoadCfg no devolvió error con un secreto inusable")
			}
			if !errors.Is(err, configuration.ErrMissingJWTSecret) {
				t.Errorf("error = %v, se esperaba ErrMissingJWTSecret", err)
			}
			if cfg != nil {
				t.Error("LoadCfg devolvió una configuración además del error")
			}
			if strings.Contains(err.Error(), c.secret) && c.secret != "" {
				t.Error("el mensaje de error repite el secreto")
			}
		})
	}
}

func TestLoadCfgAcceptsASecretAtTheMinimumLength(t *testing.T) {
	t.Setenv("NSP_JWT_SECRET", strings.Repeat("a", 32))

	cfg, err := configuration.LoadCfg()
	if err != nil {
		t.Fatalf("32 bytes es el mínimo y debería alcanzar: %v", err)
	}
	if len(cfg.JWTSecret) != 32 {
		t.Errorf("JWTSecret tiene %d bytes", len(cfg.JWTSecret))
	}
}

func TestLoadCfgAppliesDefaultsForTheOptionalValues(t *testing.T) {
	withSecret(t)

	cfg, err := configuration.LoadCfg()
	if err != nil {
		t.Fatalf("LoadCfg devolvió error: %v", err)
	}

	if cfg.Host != "127.0.0.1" {
		t.Errorf("Host = %q, se esperaba 127.0.0.1", cfg.Host)
	}
	if cfg.Port != "8080" {
		t.Errorf("Port = %q, se esperaba 8080", cfg.Port)
	}
	if cfg.AccessTTL != 15*time.Minute {
		t.Errorf("AccessTTL = %v, se esperaban 15m", cfg.AccessTTL)
	}
	if cfg.RefreshTTL != 168*time.Hour {
		t.Errorf("RefreshTTL = %v, se esperaban 168h", cfg.RefreshTTL)
	}
	if cfg.BcryptCost != 12 {
		t.Errorf("BcryptCost = %d, se esperaba 12", cfg.BcryptCost)
	}
}

func TestLoadCfgPrefersTheEnvironmentOverTheDefaults(t *testing.T) {
	withSecret(t)
	t.Setenv("NSPHOST", "0.0.0.0")
	t.Setenv("NSPPORT", "9090")
	t.Setenv("NSP_ACCESS_TTL", "1s")
	t.Setenv("NSP_REFRESH_TTL", "48h")
	t.Setenv("NSP_BCRYPT_COST", "4")

	cfg, err := configuration.LoadCfg()
	if err != nil {
		t.Fatalf("LoadCfg devolvió error: %v", err)
	}

	if cfg.GetServerAddress() != "0.0.0.0:9090" {
		t.Errorf("GetServerAddress() = %q", cfg.GetServerAddress())
	}
	if cfg.AccessTTL != time.Second {
		t.Errorf("AccessTTL = %v, se esperaba 1s", cfg.AccessTTL)
	}
	if cfg.RefreshTTL != 48*time.Hour {
		t.Errorf("RefreshTTL = %v, se esperaban 48h", cfg.RefreshTTL)
	}
	if cfg.BcryptCost != 4 {
		t.Errorf("BcryptCost = %d, se esperaba 4", cfg.BcryptCost)
	}
}

func TestLoadCfgRejectsAnUnparseableLifetime(t *testing.T) {
	cases := []struct {
		variable string
		value    string
	}{
		{"NSP_ACCESS_TTL", "quince minutos"},
		{"NSP_ACCESS_TTL", "15"},
		{"NSP_ACCESS_TTL", "0s"},
		{"NSP_ACCESS_TTL", "-5m"},
		{"NSP_REFRESH_TTL", "una semana"},
		{"NSP_REFRESH_TTL", "0"},
	}

	for _, c := range cases {
		t.Run(c.variable+"="+c.value, func(t *testing.T) {
			withSecret(t)
			t.Setenv(c.variable, c.value)

			if _, err := configuration.LoadCfg(); err == nil {
				t.Fatalf("%s=%q debería ser un error", c.variable, c.value)
			} else if !strings.Contains(err.Error(), c.variable) {
				t.Errorf("el error %q no nombra la variable que molestó", err)
			}
		})
	}
}

func TestLoadCfgRejectsAnUnusableBcryptCost(t *testing.T) {
	cases := []string{"doce", "", "3", "32", "-1"}

	for _, value := range cases {
		t.Run("NSP_BCRYPT_COST="+value, func(t *testing.T) {
			withSecret(t)
			t.Setenv("NSP_BCRYPT_COST", value)

			// Una variable presente pero vacía cae en el default, que es
			// válido; el resto tiene que fallar.
			_, err := configuration.LoadCfg()
			if value == "" {
				if err != nil {
					t.Errorf("una variable vacía debería caer en el default: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("NSP_BCRYPT_COST=%q debería ser un error", value)
			}
		})
	}
}

func TestHasSuperuserCredentialsNeedsBoth(t *testing.T) {
	cases := []struct {
		name     string
		email    string
		password string
		want     bool
	}{
		{"ninguna de las dos", "", "", false},
		{"sólo el email", "admin@nosepudo.ar", "", false},
		{"sólo la contraseña", "", "secreto", false},
		{"las dos", "admin@nosepudo.ar", "secreto", true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			withSecret(t)
			t.Setenv("NSP_SUPERUSER_EMAIL", c.email)
			t.Setenv("NSP_SUPERUSER_PASSWORD", c.password)

			cfg, err := configuration.LoadCfg()
			if err != nil {
				t.Fatalf("LoadCfg devolvió error: %v", err)
			}
			if got := cfg.HasSuperuserCredentials(); got != c.want {
				t.Errorf("HasSuperuserCredentials() = %v, se esperaba %v", got, c.want)
			}
		})
	}
}
