package configuration_test

import (
	"strings"
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/configuration"
)

const balanceTestSecret = "0123456789012345678901234567890123456789"

func TestInitialUserBalanceUsaMilPorDefecto(t *testing.T) {
	t.Setenv("NSP_JWT_SECRET", balanceTestSecret)

	cfg, err := configuration.LoadCfg()
	if err != nil {
		t.Fatalf("LoadCfg devolvió %v", err)
	}

	if cfg.InitialUserBalance != 1000 {
		t.Errorf("el saldo inicial por defecto es %d, se esperaban 1000 créditos enteros", cfg.InitialUserBalance)
	}
}

func TestInitialUserBalanceLeeLaVariableDeEntorno(t *testing.T) {
	t.Setenv("NSP_JWT_SECRET", balanceTestSecret)
	t.Setenv("NSP_INITIAL_USER_BALANCE", "2500")

	cfg, err := configuration.LoadCfg()
	if err != nil {
		t.Fatalf("LoadCfg devolvió %v", err)
	}

	if cfg.InitialUserBalance != 2500 {
		t.Errorf("el saldo inicial es %d, se esperaban 2500", cfg.InitialUserBalance)
	}
}

func TestInitialUserBalanceAceptaCero(t *testing.T) {
	t.Setenv("NSP_JWT_SECRET", balanceTestSecret)
	t.Setenv("NSP_INITIAL_USER_BALANCE", "0")

	cfg, err := configuration.LoadCfg()
	if err != nil {
		t.Fatalf("LoadCfg devolvió %v", err)
	}

	if cfg.InitialUserBalance != 0 {
		t.Errorf("el saldo inicial es %d, se esperaba cero: un despliegue puede no regalar créditos", cfg.InitialUserBalance)
	}
}

func TestInitialUserBalanceRechazaLoQueNoEsUnEnteroNoNegativo(t *testing.T) {
	casos := map[string]string{
		"texto":              "mil",
		"decimal":            "10.5",
		"negativo":           "-1",
		"vacío con espacios": " ",
	}

	for nombre, valor := range casos {
		t.Run(nombre, func(t *testing.T) {
			t.Setenv("NSP_JWT_SECRET", balanceTestSecret)
			t.Setenv("NSP_INITIAL_USER_BALANCE", valor)

			_, err := configuration.LoadCfg()
			if err == nil {
				t.Fatalf("LoadCfg aceptó %q como saldo inicial", valor)
			}
			if !strings.Contains(err.Error(), "NSP_INITIAL_USER_BALANCE") {
				t.Errorf("el error no nombra la variable: %v", err)
			}
		})
	}
}
