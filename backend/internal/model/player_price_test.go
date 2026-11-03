package model_test

import (
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

func TestPricePolicyScoreCeroDevuelveLaBase(t *testing.T) {
	policy := model.DefaultPricePolicy()

	if got := policy.Price(0); got != policy.Base {
		t.Errorf("Price(0) = %d, se esperaba la base %d", got, policy.Base)
	}
}

func TestPricePolicyAplicaLaEscala(t *testing.T) {
	policy := model.DefaultPricePolicy()

	cases := []struct {
		name  string
		score float64
		want  model.Credits
	}{
		{"medio", 0.5, model.CreditsFromUnits(51)},
		{"uno", 1.0, model.CreditsFromUnits(101)},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := policy.Price(c.score); got != c.want {
				t.Errorf("Price(%v) = %d, se esperaba %d", c.score, got, c.want)
			}
		})
	}
}

func TestPricePolicyRedondeaAlaDiezmilesima(t *testing.T) {
	policy := model.DefaultPricePolicy()

	// 0.125 * 100 = 12.5 créditos; más la base de 1 da 13.5 créditos.
	if got := policy.Price(0.125); got != model.Credits(135000) {
		t.Errorf("Price(0.125) = %d, se esperaban 13.5 créditos", got)
	}
}

func TestPricePolicyClampaAlPiso(t *testing.T) {
	policy := model.DefaultPricePolicy()

	if got := policy.Price(-1); got != policy.Base {
		t.Errorf("Price(-1) = %d, se esperaba la base %d", got, policy.Base)
	}
}

func TestDefaultPricePolicyUsaBaseYUnoYEscalaCien(t *testing.T) {
	policy := model.DefaultPricePolicy()

	if policy.Base != model.CreditsFromUnits(1) {
		t.Errorf("Base = %d, se esperaba 1 crédito", policy.Base)
	}
	if policy.Scale != model.CreditsFromUnits(100) {
		t.Errorf("Scale = %d, se esperaban 100 créditos", policy.Scale)
	}
}
