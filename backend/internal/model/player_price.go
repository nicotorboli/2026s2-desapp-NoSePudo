package model

import "math"

const (
	DefaultPriceBaseUnits  = 1
	DefaultPriceScaleUnits = 100
)

type PricePolicy struct {
	Base  Credits
	Scale Credits
}

func DefaultPricePolicy() PricePolicy {
	return PricePolicy{
		Base:  CreditsFromUnits(DefaultPriceBaseUnits),
		Scale: CreditsFromUnits(DefaultPriceScaleUnits),
	}
}

func (p PricePolicy) Price(score float64) Credits {
	price := p.Base + Credits(int64(math.Round(score*float64(p.Scale))))
	if price < p.Base {
		return p.Base
	}

	return price
}
