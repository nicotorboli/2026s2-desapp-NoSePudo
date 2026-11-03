package service_test

import (
	"errors"
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/service"
)

func regularUser(id int64) model.User {
	return model.User{ID: id, Privilege: model.PrivilegeUser, Active: true}
}

func superUser(id int64) model.User {
	return model.User{ID: id, Privilege: model.PrivilegeSuperuser, Active: true}
}

func TestPortfolioValuaLaPosicionALaCotizacionVigente(t *testing.T) {
	fixture := newMarketFixture(t)
	fixture.seedSuperuser(t)
	playerID := fixture.seedPlayer(t)
	buyer := fixture.seedUser(t, model.CreditsFromUnits(1000))

	if err := fixture.marketMaker.Bootstrap(t.Context()); err != nil {
		t.Fatalf("Bootstrap devolvió %v", err)
	}
	if _, err := fixture.market.PlaceOrder(t.Context(), buyCommand(buyer, playerID, 10)); err != nil {
		t.Fatalf("la compra devolvió %v", err)
	}

	portfolio, err := fixture.portfolio.Portfolio(t.Context(), regularUser(buyer), buyer)
	if err != nil {
		t.Fatalf("Portfolio devolvió %v", err)
	}

	if portfolio.AvailableCash != model.CreditsFromUnits(990) {
		t.Errorf("el efectivo disponible es %d, se esperaban 990", portfolio.AvailableCash)
	}
	if portfolio.ReservedCash != 0 {
		t.Errorf("el efectivo reservado es %d, se esperaba cero", portfolio.ReservedCash)
	}
	if len(portfolio.Positions) != 1 {
		t.Fatalf("hay %d posiciones, se esperaba 1", len(portfolio.Positions))
	}

	position := portfolio.Positions[0]
	if position.Quantity != 10 {
		t.Errorf("la posición es de %d tokens, se esperaban 10", position.Quantity)
	}
	if position.AveragePrice != model.CreditsFromUnits(1) {
		t.Errorf("el precio promedio es %d, se esperaba 1 crédito", position.AveragePrice)
	}
	if position.CurrentValuation != model.CreditsFromUnits(10) {
		t.Errorf("la valuación es %d, se esperaban 10 créditos", position.CurrentValuation)
	}
	if position.UnrealizedResult != 0 {
		t.Errorf("el resultado no realizado es %d, se esperaba cero: compró a la cotización vigente", position.UnrealizedResult)
	}
	if position.PlayerName == "" {
		t.Error("la posición no trae el nombre del jugador")
	}
	if portfolio.TotalValue != model.CreditsFromUnits(1000) {
		t.Errorf("el valor total es %d, se esperaban 1000: nada se perdió al comprar a precio de mercado", portfolio.TotalValue)
	}
}

func TestPortfolioSigueLaCotizacionCuandoSeMueve(t *testing.T) {
	fixture := newMarketFixture(t)
	fixture.seedSuperuser(t)
	playerID := fixture.seedPlayer(t)
	buyer := fixture.seedUser(t, model.CreditsFromUnits(1000))

	if err := fixture.marketMaker.Bootstrap(t.Context()); err != nil {
		t.Fatalf("Bootstrap devolvió %v", err)
	}
	if _, err := fixture.market.PlaceOrder(t.Context(), buyCommand(buyer, playerID, 10)); err != nil {
		t.Fatalf("la compra devolvió %v", err)
	}
	if err := fixture.marketMaker.RecordQuote(t.Context(), playerID, model.CreditsFromUnits(3), "rendimiento", "1"); err != nil {
		t.Fatalf("RecordQuote devolvió %v", err)
	}

	portfolio, err := fixture.portfolio.Portfolio(t.Context(), regularUser(buyer), buyer)
	if err != nil {
		t.Fatalf("Portfolio devolvió %v", err)
	}

	position := portfolio.Positions[0]
	if position.CurrentPrice != model.CreditsFromUnits(3) {
		t.Errorf("el precio vigente es %d, se esperaban 3", position.CurrentPrice)
	}
	if position.CurrentValuation != model.CreditsFromUnits(30) {
		t.Errorf("la valuación es %d, se esperaban 30", position.CurrentValuation)
	}
	if position.UnrealizedResult != model.CreditsFromUnits(20) {
		t.Errorf("el resultado no realizado es %d, se esperaban 20: pagó 10 y vale 30", position.UnrealizedResult)
	}
	if portfolio.TotalValue != model.CreditsFromUnits(1020) {
		t.Errorf("el valor total es %d, se esperaban 1020", portfolio.TotalValue)
	}
}

func TestPortfolioCuentaLosTokensComprometidosEnUnaVenta(t *testing.T) {
	fixture := newMarketFixture(t)
	playerID := fixture.seedPlayer(t)
	seller := fixture.seedUser(t, model.CreditsFromUnits(100))

	fixture.seedQuote(t, playerID, model.CreditsFromUnits(2))
	fixture.seedPosition(t, seller, playerID, 10, model.CreditsFromUnits(10))

	if _, err := fixture.market.PlaceOrder(t.Context(), withPrice(sellCommand(seller, playerID, 6), model.CreditsFromUnits(50))); err != nil {
		t.Fatalf("la venta devolvió %v", err)
	}

	portfolio, err := fixture.portfolio.Portfolio(t.Context(), regularUser(seller), seller)
	if err != nil {
		t.Fatalf("Portfolio devolvió %v", err)
	}

	if portfolio.Positions[0].Quantity != 10 {
		t.Errorf("la posición informa %d tokens, se esperaban los 10: los reservados se siguen teniendo",
			portfolio.Positions[0].Quantity)
	}
}

func TestPortfolioDeUnUsuarioSinNadaNoFalla(t *testing.T) {
	fixture := newMarketFixture(t)
	nuevo := fixture.seedUser(t, model.CreditsFromUnits(1000))

	portfolio, err := fixture.portfolio.Portfolio(t.Context(), regularUser(nuevo), nuevo)
	if err != nil {
		t.Fatalf("Portfolio devolvió %v", err)
	}

	if len(portfolio.Positions) != 0 {
		t.Errorf("hay %d posiciones, se esperaba ninguna", len(portfolio.Positions))
	}
	if portfolio.TotalValue != model.CreditsFromUnits(1000) {
		t.Errorf("el valor total es %d, se esperaban los 1000 de efectivo", portfolio.TotalValue)
	}
}

func TestPortfolioAjenoSoloLoVeElSuperusuario(t *testing.T) {
	fixture := newMarketFixture(t)
	propio := fixture.seedUser(t, model.CreditsFromUnits(1000))
	ajeno := fixture.seedUser(t, model.CreditsFromUnits(500))
	admin := fixture.seedSuperuser(t)

	if _, err := fixture.portfolio.Portfolio(t.Context(), regularUser(propio), ajeno); !errors.Is(err, service.ErrForbidden) {
		t.Errorf("un usuario regular leyó el patrimonio ajeno: %v", err)
	}

	portfolio, err := fixture.portfolio.Portfolio(t.Context(), superUser(admin), ajeno)
	if err != nil {
		t.Fatalf("el superusuario no pudo leer el patrimonio ajeno: %v", err)
	}
	if portfolio.UserID != ajeno {
		t.Errorf("se devolvió el patrimonio de %d, se esperaba el de %d", portfolio.UserID, ajeno)
	}
}

func TestTransactionsInformaElLadoDesdeElPuntoDeVistaDeQuienPregunta(t *testing.T) {
	fixture := newMarketFixture(t)
	fixture.seedSuperuser(t)
	playerID := fixture.seedPlayer(t)
	buyer := fixture.seedUser(t, model.CreditsFromUnits(1000))

	if err := fixture.marketMaker.Bootstrap(t.Context()); err != nil {
		t.Fatalf("Bootstrap devolvió %v", err)
	}
	if _, err := fixture.market.PlaceOrder(t.Context(), buyCommand(buyer, playerID, 10)); err != nil {
		t.Fatalf("la compra devolvió %v", err)
	}

	page, err := fixture.portfolio.Transactions(t.Context(), regularUser(buyer), buyer, 1, 20)
	if err != nil {
		t.Fatalf("Transactions devolvió %v", err)
	}

	if len(page.Items) != 1 {
		t.Fatalf("hay %d operaciones, se esperaba 1", len(page.Items))
	}

	entry := page.Items[0]
	if entry.Side != model.SideBuy {
		t.Errorf("la operación figura como %s para el comprador, se esperaba BUY", entry.Side)
	}
	if entry.Quantity != 10 {
		t.Errorf("la operación es de %d tokens, se esperaban 10", entry.Quantity)
	}
	if entry.TotalAmount != model.CreditsFromUnits(10) {
		t.Errorf("el monto es %d, se esperaban 10 créditos", entry.TotalAmount)
	}
	if entry.PlayerName == "" {
		t.Error("la operación no trae el nombre del jugador")
	}
	if entry.ExecutedAt.IsZero() {
		t.Error("la operación no trae su fecha de ejecución")
	}
}

func TestTransactionsPaginaYRespetaLaAutorizacion(t *testing.T) {
	fixture := newMarketFixture(t)
	fixture.seedSuperuser(t)
	playerID := fixture.seedPlayer(t)
	buyer := fixture.seedUser(t, model.CreditsFromUnits(1000))
	ajeno := fixture.seedUser(t, model.CreditsFromUnits(100))

	if err := fixture.marketMaker.Bootstrap(t.Context()); err != nil {
		t.Fatalf("Bootstrap devolvió %v", err)
	}

	for range 3 {
		if _, err := fixture.market.PlaceOrder(t.Context(), buyCommand(buyer, playerID, 2)); err != nil {
			t.Fatalf("la compra devolvió %v", err)
		}
	}

	page, err := fixture.portfolio.Transactions(t.Context(), regularUser(buyer), buyer, 1, 2)
	if err != nil {
		t.Fatalf("Transactions devolvió %v", err)
	}
	if len(page.Items) != 2 {
		t.Errorf("la página trae %d operaciones, se esperaban 2", len(page.Items))
	}
	if page.Total != 3 {
		t.Errorf("el total informado es %d, se esperaban 3", page.Total)
	}
	if page.TotalPages != 2 {
		t.Errorf("se informan %d páginas, se esperaban 2", page.TotalPages)
	}

	if _, err = fixture.portfolio.Transactions(t.Context(), regularUser(ajeno), buyer, 1, 20); !errors.Is(err, service.ErrForbidden) {
		t.Errorf("un usuario regular leyó el historial ajeno: %v", err)
	}
}
