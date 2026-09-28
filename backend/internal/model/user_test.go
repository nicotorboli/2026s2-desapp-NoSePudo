package model_test

import (
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

func TestNormalizeEmail(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"ya normalizado queda igual", "nico@nosepudo.ar", "nico@nosepudo.ar"},
		{"mayúsculas se bajan", "Nico@NoSePudo.AR", "nico@nosepudo.ar"},
		{"espacios alrededor se recortan", "  nico@nosepudo.ar  ", "nico@nosepudo.ar"},
		{"tabs y saltos de línea se recortan", "\t\nnico@nosepudo.ar\n", "nico@nosepudo.ar"},
		{"mayúsculas y espacios juntos", "  NICO@NoSePudo.ar\t", "nico@nosepudo.ar"},
		{"cadena vacía queda vacía", "", ""},
		{"sólo espacios queda vacía", "   \t ", ""},
		{"los espacios internos no se tocan", "ni co@nosepudo.ar", "ni co@nosepudo.ar"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := model.NormalizeEmail(c.input); got != c.want {
				t.Errorf("NormalizeEmail(%q) = %q, se esperaba %q", c.input, got, c.want)
			}
		})
	}
}

// Es el caso borde que la spec nombra: dos formas de la misma dirección tienen
// que resolver a la misma cuenta, o el índice único no protege nada.
func TestNormalizeEmailCollapsesTheSpecEdgeCase(t *testing.T) {
	registered := model.NormalizeEmail("  Nico@NoSePudo.AR ")
	signingIn := model.NormalizeEmail("nico@nosepudo.ar")

	if registered != signingIn {
		t.Errorf("el alta normalizó a %q y el login a %q: una dirección se volvería dos cuentas", registered, signingIn)
	}
}

func TestNormalizeEmailIsIdempotent(t *testing.T) {
	inputs := []string{"  Nico@NoSePudo.AR ", "nico@nosepudo.ar", "", "   "}

	for _, input := range inputs {
		once := model.NormalizeEmail(input)
		twice := model.NormalizeEmail(once)

		if once != twice {
			t.Errorf("NormalizeEmail(%q): una pasada dio %q y dos dieron %q", input, once, twice)
		}
	}
}

func TestAccountErrorsAreDistinct(t *testing.T) {
	if model.ErrEmailTaken == nil || model.ErrInvalidCredentials == nil {
		t.Fatal("los centinelas de dominio no pueden ser nil")
	}
	if model.ErrEmailTaken == model.ErrInvalidCredentials { //nolint:errorlint // se compara la identidad de dos centinelas, no se inspecciona una cadena
		t.Error("ErrEmailTaken y ErrInvalidCredentials son el mismo error: el controller no podría darles status distintos")
	}
}
