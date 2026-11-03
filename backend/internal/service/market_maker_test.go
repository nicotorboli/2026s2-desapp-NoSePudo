package service_test

import (
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/service"
)

func TestBootstrapEmiteCienTokensYLosOfreceALaCotizacionInicial(t *testing.T) {
	fixture := newMarketFixture(t)
	superuser := fixture.seedSuperuser(t)
	playerID := fixture.seedPlayer(t)

	if err := fixture.marketMaker.Bootstrap(t.Context()); err != nil {
		t.Fatalf("Bootstrap devolvió %v", err)
	}

	if total := fixture.tokensOfPlayer(t, playerID); total != service.TokensPerPlayer {
		t.Errorf("se emitieron %d tokens, se esperaban %d", total, service.TokensPerPlayer)
	}

	position := fixture.position(t, superuser, playerID)
	if position.ReservedQuantity != service.TokensPerPlayer {
		t.Errorf("el creador de mercado tiene %d tokens reservados, se esperaban %d: están comprometidos en su oferta",
			position.ReservedQuantity, service.TokensPerPlayer)
	}
	if position.AvailableQuantity != 0 {
		t.Errorf("el creador de mercado tiene %d tokens libres, se esperaba cero", position.AvailableQuantity)
	}
	if position.CostBasis != 0 {
		t.Errorf("el costo de la posición emitida es %d, se esperaba cero: los tokens se emitieron, no se compraron", position.CostBasis)
	}

	order, found := fixture.liveMarketMakerOrder(t, playerID)
	if !found {
		t.Fatal("el creador de mercado no dejó una oferta viva")
	}
	if order.Side != model.SideSell {
		t.Errorf("la oferta es %s, se esperaba SELL", order.Side)
	}
	if order.Quantity != service.TokensPerPlayer {
		t.Errorf("la oferta es por %d tokens, se esperaban %d", order.Quantity, service.TokensPerPlayer)
	}
	if order.Price != model.CreditsFromUnits(1) {
		t.Errorf("la oferta está a %d, se esperaba 1 crédito", order.Price)
	}

	if quotes := fixture.countQuotes(t, playerID); quotes != 1 {
		t.Errorf("hay %d cotizaciones, se esperaba solo la inicial", quotes)
	}
}

func TestBootstrapRepetidoNoDuplicaNada(t *testing.T) {
	fixture := newMarketFixture(t)
	superuser := fixture.seedSuperuser(t)
	playerID := fixture.seedPlayer(t)

	for corrida := range 3 {
		if err := fixture.marketMaker.Bootstrap(t.Context()); err != nil {
			t.Fatalf("la corrida %d de Bootstrap devolvió %v", corrida+1, err)
		}
	}

	if total := fixture.tokensOfPlayer(t, playerID); total != service.TokensPerPlayer {
		t.Errorf("hay %d tokens tras tres corridas, se esperaban %d: la emisión es idempotente",
			total, service.TokensPerPlayer)
	}
	if quotes := fixture.countQuotes(t, playerID); quotes != 1 {
		t.Errorf("hay %d cotizaciones tras tres corridas, se esperaba una sola inicial", quotes)
	}
	if orders := fixture.countOrders(t, superuser); orders != 1 {
		t.Errorf("hay %d órdenes del creador de mercado, se esperaba una sola viva", orders)
	}
}

func TestBootstrapNoReponeLoQueYaSeVendio(t *testing.T) {
	fixture := newMarketFixture(t)
	fixture.seedSuperuser(t)
	playerID := fixture.seedPlayer(t)
	buyer := fixture.seedUser(t, model.CreditsFromUnits(1000))

	if err := fixture.marketMaker.Bootstrap(t.Context()); err != nil {
		t.Fatalf("el primer Bootstrap devolvió %v", err)
	}

	if _, err := fixture.market.PlaceOrder(t.Context(), buyCommand(buyer, playerID, 30)); err != nil {
		t.Fatalf("la compra devolvió %v", err)
	}

	if err := fixture.marketMaker.Bootstrap(t.Context()); err != nil {
		t.Fatalf("el segundo Bootstrap devolvió %v", err)
	}

	if total := fixture.tokensOfPlayer(t, playerID); total != service.TokensPerPlayer {
		t.Errorf("hay %d tokens tras vender 30, se esperaban %d: una venta los mueve, no los destruye",
			total, service.TokensPerPlayer)
	}
	if quantity := fixture.position(t, buyer, playerID).AvailableQuantity; quantity != 30 {
		t.Errorf("el comprador tiene %d tokens, se esperaban 30", quantity)
	}

	order, found := fixture.liveMarketMakerOrder(t, playerID)
	if !found {
		t.Fatal("el creador de mercado se quedó sin oferta viva tras una venta parcial")
	}
	if order.RemainingQuantity() != 70 {
		t.Errorf("la oferta viva tiene %d tokens pendientes, se esperaban 70", order.RemainingQuantity())
	}

	fixture.assertNoReservationOutlivesItsOrder(t, "tras un rebootstrap con inventario parcialmente vendido")
}

func TestRecordQuoteRepreciaLaOfertaDelCreadorDeMercado(t *testing.T) {
	fixture := newMarketFixture(t)
	superuser := fixture.seedSuperuser(t)
	playerID := fixture.seedPlayer(t)

	if err := fixture.marketMaker.Bootstrap(t.Context()); err != nil {
		t.Fatalf("Bootstrap devolvió %v", err)
	}

	original, found := fixture.liveMarketMakerOrder(t, playerID)
	if !found {
		t.Fatal("el creador de mercado no dejó una oferta viva")
	}

	nuevoPrecio := model.CreditsFromUnits(7)
	if err := fixture.marketMaker.RecordQuote(t.Context(), playerID, nuevoPrecio, "rendimiento", "1"); err != nil {
		t.Fatalf("RecordQuote devolvió %v", err)
	}

	repreciada, found := fixture.liveMarketMakerOrder(t, playerID)
	if !found {
		t.Fatal("tras recotizar no quedó una oferta viva: el inventario quedaría sin ofrecer")
	}
	if repreciada.ID == original.ID {
		t.Error("la oferta vieja sigue viva: tenía que cancelarse y reponerse")
	}
	if repreciada.Price != nuevoPrecio {
		t.Errorf("la oferta quedó a %d, se esperaba el precio nuevo de 7", repreciada.Price)
	}
	if repreciada.Quantity != service.TokensPerPlayer {
		t.Errorf("la oferta repuesta es por %d tokens, se esperaban los %d disponibles",
			repreciada.Quantity, service.TokensPerPlayer)
	}

	if fixture.orderStatus(t, original.ID) != model.StatusCancelled {
		t.Error("la oferta original no quedó CANCELLED")
	}

	position := fixture.position(t, superuser, playerID)
	if position.ReservedQuantity != service.TokensPerPlayer || position.AvailableQuantity != 0 {
		t.Errorf("la posición quedó en %d disponibles y %d reservados tras recotizar",
			position.AvailableQuantity, position.ReservedQuantity)
	}

	fixture.assertNoReservationOutlivesItsOrder(t, "tras recotizar")
}

func TestRecordQuoteNoTocaLasOrdenesDeLosUsuarios(t *testing.T) {
	fixture := newMarketFixture(t)
	fixture.seedSuperuser(t)
	playerID := fixture.seedPlayer(t)
	usuario := fixture.seedUser(t, model.CreditsFromUnits(1000))

	if err := fixture.marketMaker.Bootstrap(t.Context()); err != nil {
		t.Fatalf("Bootstrap devolvió %v", err)
	}

	mitadDeCredito := model.Credits(model.CreditScale / 2)

	propia, err := fixture.market.PlaceOrder(t.Context(), withPrice(buyCommand(usuario, playerID, 4), mitadDeCredito))
	if err != nil {
		t.Fatalf("la orden del usuario devolvió %v", err)
	}
	if propia.Order.Status != model.StatusPending {
		t.Fatalf("la orden del usuario quedó en %s: tiene que quedar en reposo para que haya algo que la recotización pueda romper", propia.Order.Status)
	}

	if err = fixture.marketMaker.RecordQuote(t.Context(), playerID, model.CreditsFromUnits(9), "rendimiento", "1"); err != nil {
		t.Fatalf("RecordQuote devolvió %v", err)
	}

	if estado := fixture.orderStatus(t, propia.Order.ID); estado == model.StatusCancelled {
		t.Error("la recotización canceló una orden de un usuario regular")
	}

	ordenes, err := fixture.market.ListOpenOrders(t.Context(), usuario, nil)
	if err != nil {
		t.Fatalf("ListOpenOrders devolvió %v", err)
	}
	if len(ordenes) != 1 {
		t.Errorf("al usuario le quedaron %d órdenes vivas, se esperaba la suya intacta", len(ordenes))
	}
}

func TestRecordScoreQuoteGuardaElPrecioCalculado(t *testing.T) {
	fixture := newMarketFixture(t)
	fixture.seedSuperuser(t)
	playerID := fixture.seedPlayer(t)

	if err := fixture.marketMaker.Bootstrap(t.Context()); err != nil {
		t.Fatalf("Bootstrap devolvió %v", err)
	}

	if err := fixture.marketMaker.RecordScoreQuote(t.Context(), playerID, 0.5, nil, "rateada"); err != nil {
		t.Fatalf("RecordScoreQuote devolvió %v", err)
	}

	quote, err := fixture.daos.Quote.GetLatestByPlayer(t.Context(), nil, playerID)
	if err != nil {
		t.Fatalf("GetLatestByPlayer devolvió %v", err)
	}
	expected := model.DefaultPricePolicy().Price(0.5)
	if quote.Price != expected {
		t.Errorf("el precio cotizado es %d, se esperaba %d", quote.Price, expected)
	}
	if quote.Strategy != "rateada" {
		t.Errorf("la estrategia es %q, se esperaba rateada", quote.Strategy)
	}
}

func TestRecordScoreQuoteRepreciaLaOfertaDelCreador(t *testing.T) {
	fixture := newMarketFixture(t)
	fixture.seedSuperuser(t)
	playerID := fixture.seedPlayer(t)

	if err := fixture.marketMaker.Bootstrap(t.Context()); err != nil {
		t.Fatalf("Bootstrap devolvió %v", err)
	}

	original, found := fixture.liveMarketMakerOrder(t, playerID)
	if !found {
		t.Fatal("el creador de mercado no dejó una oferta viva")
	}

	if err := fixture.marketMaker.RecordScoreQuote(t.Context(), playerID, 0.5, nil, "rateada"); err != nil {
		t.Fatalf("RecordScoreQuote devolvió %v", err)
	}

	expected := model.DefaultPricePolicy().Price(0.5)
	repreciada, found := fixture.liveMarketMakerOrder(t, playerID)
	if !found {
		t.Fatal("tras recotizar por score no quedó una oferta viva")
	}
	if repreciada.ID == original.ID {
		t.Error("la oferta vieja sigue viva: tenía que cancelarse y reponerse")
	}
	if repreciada.Price != expected {
		t.Errorf("la oferta quedó a %d, se esperaba %d", repreciada.Price, expected)
	}
	if fixture.orderStatus(t, original.ID) != model.StatusCancelled {
		t.Error("la oferta original no quedó CANCELLED")
	}
}

func TestRecordScoreQuoteConScoreCeroCobraLaBase(t *testing.T) {
	fixture := newMarketFixture(t)
	fixture.seedSuperuser(t)
	playerID := fixture.seedPlayer(t)

	if err := fixture.marketMaker.Bootstrap(t.Context()); err != nil {
		t.Fatalf("Bootstrap devolvió %v", err)
	}

	if err := fixture.marketMaker.RecordScoreQuote(t.Context(), playerID, 0, nil, "rateada"); err != nil {
		t.Fatalf("RecordScoreQuote devolvió %v", err)
	}

	order, found := fixture.liveMarketMakerOrder(t, playerID)
	if !found {
		t.Fatal("tras recotizar no quedó una oferta viva")
	}
	if order.Price != model.DefaultPricePolicy().Base {
		t.Errorf("la oferta quedó a %d, se esperaba la base %d", order.Price, model.DefaultPricePolicy().Base)
	}
}
