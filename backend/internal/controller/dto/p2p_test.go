package dto_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/controller/dto"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

func TestCreateP2PSessionRequestParseaPrecioEnCreditos(t *testing.T) {
	casos := map[string]struct {
		raw      string
		esperado model.Credits
	}{
		"entero":             {raw: "3", esperado: model.CreditsFromUnits(3)},
		"un decimal y medio": {raw: "1.5", esperado: model.Credits(15000)},
		"cuatro decimales":   {raw: "0.0001", esperado: model.Credits(1)},
		"con ceros finales":  {raw: "2.0500", esperado: model.Credits(20500)},
	}

	for nombre, caso := range casos {
		t.Run(nombre, func(t *testing.T) {
			request := dto.CreateP2PSessionRequest{PlayerID: 1, Quantity: 1, PricePerToken: caso.raw}

			precio, err := request.ParsedPrice()
			if err != nil {
				t.Fatalf("no se pudo parsear %q: %v", caso.raw, err)
			}
			if precio != caso.esperado {
				t.Errorf("se obtuvo %d, se esperaba %d", precio, caso.esperado)
			}
			if err := request.Validate(); err != nil {
				t.Errorf("la petición válida fue rechazada: %v", err)
			}
		})
	}
}

func TestCreateP2PSessionRequestRechazaPreciosInvalidos(t *testing.T) {
	casos := map[string]string{
		"vacío":           "",
		"texto":           "abc",
		"cero":            "0",
		"negativo":        "-1",
		"cinco decimales": "1.23456",
	}

	for nombre, raw := range casos {
		t.Run(nombre, func(t *testing.T) {
			request := dto.CreateP2PSessionRequest{PlayerID: 1, Quantity: 1, PricePerToken: raw}

			if err := request.Validate(); err == nil {
				t.Fatalf("se aceptó el precio inválido %q", raw)
			}
			if _, err := request.ParsedPrice(); err == nil {
				t.Fatalf("se parseó el precio inválido %q", raw)
			}
		})
	}
}

func TestP2PSessionResponseRindePrecioYTotalEnCreditos(t *testing.T) {
	session := model.P2PSession{
		Token:     "un-token",
		Status:    model.P2PStatusPending,
		SellerID:  42,
		PlayerID:  7,
		Quantity:  5,
		Price:     model.Credits(15000),
		CreatedAt: time.Date(2026, time.November, 16, 12, 0, 0, 0, time.UTC),
		ExpiresAt: time.Date(2026, time.November, 16, 12, 5, 0, 0, time.UTC),
	}
	player := model.Player{ID: 7, Name: "Jugador", ClubName: "Club"}

	response := dto.P2PSessionResponseDesdeModelo(session, player, "")

	if response.PricePerToken != "1.5000" {
		t.Errorf("pricePerToken: se obtuvo %q, se esperaba %q", response.PricePerToken, "1.5000")
	}
	if response.TotalAmount != "7.5000" {
		t.Errorf("totalAmount: se obtuvo %q, se esperaba %q", response.TotalAmount, "7.5000")
	}

	raw, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("no se pudo serializar la respuesta: %v", err)
	}
	if !json.Valid(raw) {
		t.Fatal("la respuesta no es JSON válido")
	}

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("no se pudo deserializar la respuesta: %v", err)
	}
	if decoded["pricePerToken"] != "1.5000" {
		t.Errorf("el JSON expone %v, se esperaba la cadena %q", decoded["pricePerToken"], "1.5000")
	}
}

func TestP2PTradeResponseRindePrecioYTotalEnCreditos(t *testing.T) {
	trade := model.Trade{
		ID:          9,
		BuyerID:     43,
		SellerID:    42,
		PlayerID:    7,
		Quantity:    5,
		Price:       model.Credits(15000),
		TotalAmount: model.Credits(75000),
		ExecutedAt:  time.Date(2026, time.November, 16, 12, 1, 0, 0, time.UTC),
	}

	response := dto.P2PTradeResponseDesdeModelo(trade, "un-token")

	if response.PricePerToken != "1.5000" {
		t.Errorf("pricePerToken: se obtuvo %q, se esperaba %q", response.PricePerToken, "1.5000")
	}
	if response.TotalAmount != "7.5000" {
		t.Errorf("totalAmount: se obtuvo %q, se esperaba %q", response.TotalAmount, "7.5000")
	}
}
