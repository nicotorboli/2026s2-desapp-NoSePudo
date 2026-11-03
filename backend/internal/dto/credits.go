package dto

import (
	"fmt"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

func CreditsADecimal(credits model.Credits) string {
	negative := credits < 0
	units := int64(credits)
	if negative {
		units = -units
	}

	whole := units / model.CreditScale
	fraction := units % model.CreditScale

	rendered := fmt.Sprintf("%d.%04d", whole, fraction)
	if negative {
		rendered = "-" + rendered
	}

	return rendered
}
