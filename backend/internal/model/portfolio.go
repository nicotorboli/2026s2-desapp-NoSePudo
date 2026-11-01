package model

type PortfolioPosition struct {
	PlayerName        string
	ClubName          string
	PlayerPosition    string
	PlayerID          int64
	Quantity          int64
	AvailableQuantity int64
	ReservedQuantity  int64
	AveragePrice      Credits
	CurrentPrice      Credits
	CurrentValuation  Credits
	UnrealizedResult  Credits
}

type Portfolio struct {
	Positions        []PortfolioPosition
	UserID           int64
	AvailableCash    Credits
	ReservedCash     Credits
	TokensValuation  Credits
	TotalValue       Credits
	UnrealizedResult Credits
}

func BuildPortfolio(
	userID int64,
	account Account,
	positions []Position,
	quotes map[int64]Credits,
	players map[int64]Player,
) Portfolio {
	portfolio := Portfolio{
		Positions:     make([]PortfolioPosition, 0, len(positions)),
		UserID:        userID,
		AvailableCash: account.AvailableBalance,
		ReservedCash:  account.ReservedBalance,
	}

	for i := range positions {
		quantity := positions[i].TotalQuantity()
		if quantity == 0 {
			continue
		}

		entry := portfolioPositionOf(positions[i], quotes[positions[i].PlayerID], players[positions[i].PlayerID])
		portfolio.Positions = append(portfolio.Positions, entry)
		portfolio.TokensValuation += entry.CurrentValuation
		portfolio.UnrealizedResult += entry.UnrealizedResult
	}

	portfolio.TotalValue = portfolio.AvailableCash.Plus(portfolio.ReservedCash).Plus(portfolio.TokensValuation)

	return portfolio
}

func portfolioPositionOf(position Position, currentPrice Credits, player Player) PortfolioPosition {
	quantity := position.TotalQuantity()
	valuation := currentPrice.Times(quantity)

	return PortfolioPosition{
		PlayerName:        player.Name,
		ClubName:          player.ClubName,
		PlayerPosition:    player.Position,
		PlayerID:          position.PlayerID,
		Quantity:          quantity,
		AvailableQuantity: position.AvailableQuantity,
		ReservedQuantity:  position.ReservedQuantity,
		AveragePrice:      position.AveragePrice(),
		CurrentPrice:      currentPrice,
		CurrentValuation:  valuation,
		UnrealizedResult:  valuation.Minus(position.CostBasis),
	}
}
