package controller_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/controller"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/httphandler"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/middleware"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/service"
)

type fakeP2PService struct {
	err          error
	player       model.Player
	gotCommand   service.CreateP2PCommand
	session      model.P2PSession
	createCalled bool
}

func (f *fakeP2PService) Create(
	_ context.Context,
	cmd service.CreateP2PCommand,
) (model.P2PSession, model.Player, error) {
	f.createCalled = true
	f.gotCommand = cmd
	return f.session, f.player, f.err
}

func (f *fakeP2PService) Get(context.Context, string) (model.P2PSession, model.Player, error) {
	return f.session, f.player, f.err
}

func (f *fakeP2PService) Accept(context.Context, string, int64, string) (model.Trade, error) {
	return model.Trade{}, f.err
}

func (f *fakeP2PService) Cancel(context.Context, string, int64, string) error {
	return f.err
}

func callP2P(
	t *testing.T,
	endpoint httphandler.Endpoint,
	actor *middleware.Actor,
	body string,
) (int, map[string]any) {
	t.Helper()

	handler := httphandler.Wrap(endpoint, slog.New(slog.NewTextHandler(io.Discard, nil)))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/p2p/sessions",
		strings.NewReader(body),
	)
	if actor != nil {
		request = request.WithContext(middleware.WithActor(request.Context(), *actor))
	}
	handler.ServeHTTP(recorder, request)

	var decoded map[string]any
	if recorder.Body.Len() > 0 {
		if err := json.Unmarshal(recorder.Body.Bytes(), &decoded); err != nil {
			t.Fatalf("la respuesta no es JSON: %v (%s)", err, recorder.Body.String())
		}
	}

	return recorder.Code, decoded
}

func TestCreateP2PSessionConvierteElPrecioDecimalACreditos(t *testing.T) {
	fake := &fakeP2PService{
		session: model.P2PSession{
			Token:     "un-token",
			Status:    model.P2PStatusPending,
			SellerID:  42,
			PlayerID:  7,
			Quantity:  5,
			Price:     model.Credits(15000),
			CreatedAt: time.Date(2026, time.November, 16, 12, 0, 0, 0, time.UTC),
			ExpiresAt: time.Date(2026, time.November, 16, 12, 5, 0, 0, time.UTC),
		},
		player: model.Player{ID: 7, Name: "Jugador", ClubName: "Club"},
	}
	actor := anActor()

	status, body := callP2P(
		t,
		controller.NewP2PController(fake).Create(),
		&actor,
		`{"playerId":7,"quantity":5,"pricePerToken":"1.5"}`,
	)

	if status != http.StatusCreated {
		t.Fatalf("se obtuvo status %d, se esperaba 201", status)
	}
	if !fake.createCalled {
		t.Fatal("el service no fue invocado")
	}
	if fake.gotCommand.Price != model.Credits(15000) {
		t.Errorf("el service recibió %d, se esperaba %d", fake.gotCommand.Price, model.Credits(15000))
	}
	if body["pricePerToken"] != "1.5000" {
		t.Errorf("pricePerToken: se obtuvo %v, se esperaba %q", body["pricePerToken"], "1.5000")
	}
	if body["totalAmount"] != "7.5000" {
		t.Errorf("totalAmount: se obtuvo %v, se esperaba %q", body["totalAmount"], "7.5000")
	}
}

func TestCreateP2PSessionRechazaUnPrecioInvalido(t *testing.T) {
	fake := &fakeP2PService{}
	actor := anActor()

	status, _ := callP2P(
		t,
		controller.NewP2PController(fake).Create(),
		&actor,
		`{"playerId":7,"quantity":5,"pricePerToken":"abc"}`,
	)

	if status != http.StatusBadRequest {
		t.Fatalf("se obtuvo status %d, se esperaba 400", status)
	}
	if fake.createCalled {
		t.Fatal("el service no debía ser invocado con un precio inválido")
	}
}
