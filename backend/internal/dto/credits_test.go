package dto

import (
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

func TestCreditsADecimal(t *testing.T) {
	cases := []struct {
		name    string
		want    string
		credits model.Credits
	}{
		{name: "un credito entero", credits: model.CreditsFromUnits(1), want: "1.0000"},
		{name: "creditos con fraccion", credits: model.CreditsFromUnits(51) + model.Credits(5000), want: "51.5000"},
		{name: "negativo", credits: model.CreditsFromUnits(-2), want: "-2.0000"},
		{name: "cero", credits: 0, want: "0.0000"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := CreditsADecimal(c.credits); got != c.want {
				t.Errorf("CreditsADecimal(%d) = %q, se esperaba %q", c.credits, got, c.want)
			}
		})
	}
}
