package model_test

import (
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

func catalogo() map[int64]model.Player {
	return map[int64]model.Player{
		jugador: {Name: "Bukayo Saka", ClubName: "Arsenal FC", Position: "Offence", ID: jugador},
	}
}

func TestBuildPortfolioValuaLaPosicionConElPrecioVigente(t *testing.T) {
	account := model.Account{UserID: comprador, AvailableBalance: model.CreditsFromUnits(970)}
	positions := []model.Position{posicionCon(20, 0, model.CreditsFromUnits(30))}
	quotes := map[int64]model.Credits{jugador: model.CreditsFromUnits(2)}

	portfolio := model.BuildPortfolio(comprador, account, positions, quotes, catalogo())

	if len(portfolio.Positions) != 1 {
		t.Fatalf("se obtuvieron %d posiciones, se esperaba 1", len(portfolio.Positions))
	}

	got := portfolio.Positions[0]
	if got.Quantity != 20 {
		t.Errorf("Quantity = %d, se esperaba 20", got.Quantity)
	}
	if got.AveragePrice != model.CreditsFromUnits(1)+5000 {
		t.Errorf("AveragePrice = %d, se esperaba 1.5 créditos exactos", got.AveragePrice)
	}
	if got.CurrentValuation != model.CreditsFromUnits(40) {
		t.Errorf("CurrentValuation = %d, se esperaban 40 créditos", got.CurrentValuation)
	}
	if got.UnrealizedResult != model.CreditsFromUnits(10) {
		t.Errorf("UnrealizedResult = %d, se esperaban 10 créditos", got.UnrealizedResult)
	}
}

func TestBuildPortfolioLlevaLosDatosDelJugador(t *testing.T) {
	positions := []model.Position{posicionCon(10, 0, model.CreditsFromUnits(10))}
	quotes := map[int64]model.Credits{jugador: model.CreditsFromUnits(1)}

	portfolio := model.BuildPortfolio(comprador, model.Account{UserID: comprador}, positions, quotes, catalogo())

	got := portfolio.Positions[0]
	if got.PlayerName != "Bukayo Saka" || got.ClubName != "Arsenal FC" || got.PlayerPosition != "Offence" {
		t.Errorf("los datos del jugador llegaron como %+v", got)
	}
}

func TestBuildPortfolioCuentaLosTokensReservadosComoPropios(t *testing.T) {
	account := model.Account{UserID: comprador, AvailableBalance: model.CreditsFromUnits(100)}
	positions := []model.Position{posicionCon(6, 4, model.CreditsFromUnits(10))}
	quotes := map[int64]model.Credits{jugador: model.CreditsFromUnits(2)}

	portfolio := model.BuildPortfolio(comprador, account, positions, quotes, catalogo())

	if portfolio.Positions[0].Quantity != 10 {
		t.Errorf("Quantity = %d: los tokens comprometidos por una venta viva siguen siendo propios",
			portfolio.Positions[0].Quantity)
	}
	if portfolio.TokensValuation != model.CreditsFromUnits(20) {
		t.Errorf("TokensValuation = %d, se esperaban 20 créditos", portfolio.TokensValuation)
	}
}

func TestBuildPortfolioSumaElEfectivoReservadoAlValorTotal(t *testing.T) {
	account := model.Account{
		UserID:           comprador,
		AvailableBalance: model.CreditsFromUnits(960),
		ReservedBalance:  model.CreditsFromUnits(10),
	}
	positions := []model.Position{posicionCon(20, 0, model.CreditsFromUnits(30))}
	quotes := map[int64]model.Credits{jugador: model.CreditsFromUnits(2)}

	portfolio := model.BuildPortfolio(comprador, account, positions, quotes, catalogo())

	if portfolio.AvailableCash != model.CreditsFromUnits(960) {
		t.Errorf("AvailableCash = %d, se esperaban 960 créditos", portfolio.AvailableCash)
	}
	if portfolio.ReservedCash != model.CreditsFromUnits(10) {
		t.Errorf("ReservedCash = %d, se esperaban 10 créditos", portfolio.ReservedCash)
	}
	if portfolio.TotalValue != model.CreditsFromUnits(1010) {
		t.Errorf("TotalValue = %d, se esperaban 1010 créditos", portfolio.TotalValue)
	}
	if portfolio.UnrealizedResult != model.CreditsFromUnits(10) {
		t.Errorf("UnrealizedResult = %d, se esperaban 10 créditos", portfolio.UnrealizedResult)
	}
}

func TestBuildPortfolioSinPosicionesEsSoloEfectivo(t *testing.T) {
	account := model.Account{UserID: comprador, AvailableBalance: model.CreditsFromUnits(1000)}

	portfolio := model.BuildPortfolio(comprador, account, nil, nil, nil)

	if len(portfolio.Positions) != 0 {
		t.Errorf("se obtuvieron %d posiciones, se esperaban 0", len(portfolio.Positions))
	}
	if portfolio.TokensValuation != 0 || portfolio.UnrealizedResult != 0 {
		t.Errorf("valuación = %d y resultado = %d, se esperaban 0", portfolio.TokensValuation, portfolio.UnrealizedResult)
	}
	if portfolio.TotalValue != model.CreditsFromUnits(1000) {
		t.Errorf("TotalValue = %d, se esperaban 1000 créditos", portfolio.TotalValue)
	}
}

func TestBuildPortfolioOmiteLasPosicionesQueQuedaronEnCero(t *testing.T) {
	positions := []model.Position{posicionCon(0, 0, 0)}
	quotes := map[int64]model.Credits{jugador: model.CreditsFromUnits(2)}

	portfolio := model.BuildPortfolio(comprador, model.Account{UserID: comprador}, positions, quotes, catalogo())

	if len(portfolio.Positions) != 0 {
		t.Errorf("se reportó una posición sin tokens: %+v", portfolio.Positions)
	}
}

func TestBuildPortfolioEsExactoConComprasAPreciosDistintos(t *testing.T) {
	position := model.Position{UserID: comprador, PlayerID: jugador}
	if err := position.Receive(10, model.CreditsFromUnits(10)); err != nil {
		t.Fatalf("Receive devolvió %v", err)
	}
	if err := position.Receive(10, model.CreditsFromUnits(20)); err != nil {
		t.Fatalf("Receive devolvió %v", err)
	}

	quotes := map[int64]model.Credits{jugador: model.CreditsFromUnits(2)}
	portfolio := model.BuildPortfolio(comprador, model.Account{UserID: comprador}, []model.Position{position}, quotes, catalogo())

	got := portfolio.Positions[0]
	if got.AveragePrice != 15000 {
		t.Errorf("AveragePrice = %d, se esperaba 15000 unidades escaladas, es decir 1.5000", got.AveragePrice)
	}
	if got.CurrentValuation != model.CreditsFromUnits(40) {
		t.Errorf("CurrentValuation = %d, se esperaban 40 créditos", got.CurrentValuation)
	}
	if got.UnrealizedResult != model.CreditsFromUnits(10) {
		t.Errorf("UnrealizedResult = %d, se esperaban 10 créditos", got.UnrealizedResult)
	}
}
