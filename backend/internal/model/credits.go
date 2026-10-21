package model

type Credits int64

const CreditScale = 10000

func CreditsFromUnits(units int64) Credits {
	return Credits(units * CreditScale)
}

func (c Credits) Times(quantity int64) Credits {
	return Credits(int64(c) * quantity)
}

func (c Credits) Plus(other Credits) Credits {
	return c + other
}

func (c Credits) Minus(other Credits) Credits {
	return c - other
}

func (c Credits) DividedBy(quantity int64) Credits {
	if quantity == 0 {
		return 0
	}
	return Credits(int64(c) / quantity)
}
