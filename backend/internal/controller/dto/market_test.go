package dto_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/controller/dto"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/service"
)

func TestCreditsADecimalRindeCuatroDecimales(t *testing.T) {
	casos := map[string]struct {
		esperado string
		credits  model.Credits
	}{
		"cero":                 {"0.0000", 0},
		"un crédito":           {"1.0000", model.CreditsFromUnits(1)},
		"mil créditos":         {"1000.0000", model.CreditsFromUnits(1000)},
		"medio crédito":        {"0.5000", model.Credits(5000)},
		"una diezmilésima":     {"0.0001", model.Credits(1)},
		"con parte fraccional": {"1.2345", model.Credits(12345)},
		"negativo":             {"-1.2345", model.Credits(-12345)},
	}

	for nombre, caso := range casos {
		t.Run(nombre, func(t *testing.T) {
			if rendered := dto.CreditsADecimal(caso.credits); rendered != caso.esperado {
				t.Errorf("se rindió %q, se esperaba %q", rendered, caso.esperado)
			}
		})
	}
}

func TestCreditsVanYVuelvenSinPerderPrecision(t *testing.T) {
	valores := []model.Credits{
		0,
		model.Credits(1),
		model.Credits(9999),
		model.CreditsFromUnits(1),
		model.CreditsFromUnits(1000),
		model.Credits(12345678),
		model.Credits(-42),
	}

	for _, original := range valores {
		rendered := dto.CreditsADecimal(original)

		parsed, err := dto.CreditsDesdeDecimal(rendered)
		if err != nil {
			t.Fatalf("no se pudo volver a leer %q: %v", rendered, err)
		}
		if parsed != original {
			t.Errorf("%d se rindió como %q y volvió como %d", original, rendered, parsed)
		}
	}
}

func TestCreditsDesdeDecimalAceptaMenosDeCuatroDecimales(t *testing.T) {
	casos := map[string]model.Credits{
		"1":      model.CreditsFromUnits(1),
		"1.5":    model.Credits(15000),
		"1.25":   model.Credits(12500),
		"0.001":  model.Credits(10),
		"0.0001": model.Credits(1),
	}

	for raw, esperado := range casos {
		t.Run(raw, func(t *testing.T) {
			parsed, err := dto.CreditsDesdeDecimal(raw)
			if err != nil {
				t.Fatalf("CreditsDesdeDecimal(%q) devolvió %v", raw, err)
			}
			if parsed != esperado {
				t.Errorf("%q se leyó como %d, se esperaba %d", raw, parsed, esperado)
			}
		})
	}
}

func TestCreditsDesdeDecimalRechazaLoQueNoPuedeRepresentar(t *testing.T) {
	casos := map[string]error{
		"":        dto.ErrPriceNotANumber,
		"abc":     dto.ErrPriceNotANumber,
		"1.00001": dto.ErrPriceTooPrecise,
	}

	for raw, esperado := range casos {
		t.Run(raw, func(t *testing.T) {
			if _, err := dto.CreditsDesdeDecimal(raw); !errors.Is(err, esperado) {
				t.Errorf("CreditsDesdeDecimal(%q) devolvió %v, se esperaba %v", raw, err, esperado)
			}
		})
	}
}

func precio(valor string) *string {
	return &valor
}

func TestPlaceOrderRequestValida(t *testing.T) {
	casos := map[string]struct {
		esperado error
		request  dto.PlaceOrderRequest
	}{
		"sin jugador":        {dto.ErrPlayerNotIdentified, dto.PlaceOrderRequest{Quantity: 1}},
		"cantidad cero":      {dto.ErrQuantityNotPositive, dto.PlaceOrderRequest{PlayerID: 1}},
		"cantidad negativa":  {dto.ErrQuantityNotPositive, dto.PlaceOrderRequest{PlayerID: 1, Quantity: -2}},
		"precio cero":        {dto.ErrPriceNotPositive, dto.PlaceOrderRequest{PlayerID: 1, Quantity: 1, Price: precio("0")}},
		"precio no numérico": {dto.ErrPriceNotANumber, dto.PlaceOrderRequest{PlayerID: 1, Quantity: 1, Price: precio("gratis")}},
		"vigencia inventada": {dto.ErrUnknownTimeInForce, dto.PlaceOrderRequest{PlayerID: 1, Quantity: 1, TimeInForce: precio("YOLO")}},
	}

	for nombre, caso := range casos {
		t.Run(nombre, func(t *testing.T) {
			if err := caso.request.Validate(); !errors.Is(err, caso.esperado) {
				t.Errorf("Validate devolvió %v, se esperaba %v", err, caso.esperado)
			}
		})
	}
}

func TestPlaceOrderRequestAceptaLaOrdenMinima(t *testing.T) {
	request := dto.PlaceOrderRequest{PlayerID: 7, Quantity: 10}

	if err := request.Validate(); err != nil {
		t.Fatalf("Validate devolvió %v para una orden sin precio ni vigencia", err)
	}

	price, err := request.ParsedPrice()
	if err != nil {
		t.Fatalf("ParsedPrice devolvió %v", err)
	}
	if price != nil {
		t.Errorf("el precio es %v, se esperaba nil: sin precio se usa la cotización vigente", *price)
	}
	if request.ParsedTimeInForce() != nil {
		t.Error("la vigencia es distinta de nil: el servicio aplica el default")
	}
}

func TestOrderResponseSerializaEnCamelCase(t *testing.T) {
	order := model.Order{
		CreatedAt:        time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC),
		Side:             model.SideBuy,
		TimeInForce:      model.GoodTillCancelled,
		Status:           model.StatusPartiallyFilled,
		ID:               42,
		PlayerID:         7,
		Quantity:         10,
		ExecutedQuantity: 4,
		Price:            model.CreditsFromUnits(2),
	}

	trades := []model.Trade{{
		ExecutedAt:  order.CreatedAt,
		Quantity:    4,
		Price:       model.CreditsFromUnits(2),
		TotalAmount: model.CreditsFromUnits(8),
	}}

	encoded, err := json.Marshal(dto.OrderResponseDesdeModelo(order, trades))
	if err != nil {
		t.Fatalf("Marshal devolvió %v", err)
	}

	var decoded map[string]any
	if err = json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal devolvió %v", err)
	}

	for _, campo := range []string{"playerId", "executedQuantity", "remainingQuantity", "timeInForce", "createdAt", "trades"} {
		if _, present := decoded[campo]; !present {
			t.Errorf("falta el campo %q en la respuesta", campo)
		}
	}
	if decoded["remainingQuantity"] != float64(6) {
		t.Errorf("remainingQuantity es %v, se esperaban 6", decoded["remainingQuantity"])
	}
	if decoded["price"] != "2.0000" {
		t.Errorf("price es %v, se esperaba la cadena 2.0000", decoded["price"])
	}
}

func TestCancelOrderResponseOmiteLoQueNoSeLibero(t *testing.T) {
	soloCreditos := dto.CancelOrderResponseDesdeModelo(service.CancelledOrder{
		Order:           model.Order{ID: 1, Status: model.StatusCancelled},
		ReleasedCredits: model.CreditsFromUnits(10),
	})
	if soloCreditos.ReleasedCredits == nil || *soloCreditos.ReleasedCredits != "10.0000" {
		t.Errorf("releasedCredits es %v, se esperaba 10.0000", soloCreditos.ReleasedCredits)
	}
	if soloCreditos.ReleasedTokens != nil {
		t.Error("una cancelación de compra informó tokens liberados")
	}

	soloTokens := dto.CancelOrderResponseDesdeModelo(service.CancelledOrder{
		Order:          model.Order{ID: 2, Status: model.StatusCancelled},
		ReleasedTokens: 5,
	})
	if soloTokens.ReleasedTokens == nil || *soloTokens.ReleasedTokens != 5 {
		t.Errorf("releasedTokens es %v, se esperaban 5", soloTokens.ReleasedTokens)
	}
	if soloTokens.ReleasedCredits != nil {
		t.Error("una cancelación de venta informó créditos liberados")
	}
}

func TestTransactionPageEspejaElSobreYaPublicado(t *testing.T) {
	page := model.PageResult[service.TransactionEntry]{
		Items: []service.TransactionEntry{{
			ExecutedAt:  time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC),
			PlayerName:  "Jugador",
			Side:        model.SideBuy,
			ID:          1,
			PlayerID:    7,
			Quantity:    4,
			Price:       model.CreditsFromUnits(2),
			TotalAmount: model.CreditsFromUnits(8),
		}},
		Total:      3,
		Page:       1,
		Limit:      2,
		TotalPages: 2,
	}

	encoded, err := json.Marshal(dto.TransactionPageResponseDesdeModelo(page))
	if err != nil {
		t.Fatalf("Marshal devolvió %v", err)
	}

	var decoded map[string]any
	if err = json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal devolvió %v", err)
	}

	for _, campo := range []string{"items", "total", "page", "limit", "totalPages"} {
		if _, present := decoded[campo]; !present {
			t.Errorf("falta el campo %q: el sobre tiene que ser el mismo que el de jugadores", campo)
		}
	}
}

func TestPortfolioResponseRindeTodoComoDecimal(t *testing.T) {
	portfolio := model.Portfolio{
		Positions: []model.PortfolioPosition{{
			PlayerName:       "Jugador",
			ClubName:         "Arsenal FC",
			PlayerPosition:   "Offence",
			PlayerID:         7,
			Quantity:         10,
			AveragePrice:     model.CreditsFromUnits(1),
			CurrentPrice:     model.CreditsFromUnits(3),
			CurrentValuation: model.CreditsFromUnits(30),
			UnrealizedResult: model.CreditsFromUnits(20),
		}},
		UserID:           5,
		AvailableCash:    model.CreditsFromUnits(990),
		TokensValuation:  model.CreditsFromUnits(30),
		TotalValue:       model.CreditsFromUnits(1020),
		UnrealizedResult: model.CreditsFromUnits(20),
	}

	response := dto.PortfolioResponseDesdeModelo(portfolio)

	if response.TotalValue != "1020.0000" {
		t.Errorf("totalValue es %q, se esperaba 1020.0000", response.TotalValue)
	}
	if len(response.Positions) != 1 {
		t.Fatalf("hay %d posiciones, se esperaba 1", len(response.Positions))
	}
	if response.Positions[0].CurrentValuation != "30.0000" {
		t.Errorf("currentValuation es %q, se esperaba 30.0000", response.Positions[0].CurrentValuation)
	}
	if response.Positions[0].ClubName != "Arsenal FC" {
		t.Errorf("clubName es %q", response.Positions[0].ClubName)
	}
}
