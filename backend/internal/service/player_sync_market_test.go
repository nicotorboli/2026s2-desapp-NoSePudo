package service_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/service"
)

type bootstrapSpy struct {
	err   error
	calls int
}

func (s *bootstrapSpy) Bootstrap(ctx context.Context) error {
	s.calls++

	return s.err
}

func TestUnaSincronizacionRepetidaNoEmiteTokensNuevos(t *testing.T) {
	fixture := newMarketFixture(t)
	fixture.seedSuperuser(t)
	playerID := fixture.seedPlayer(t)

	for corrida := range 3 {
		if err := fixture.marketMaker.Bootstrap(t.Context()); err != nil {
			t.Fatalf("la sincronización %d devolvió %v", corrida+1, err)
		}
	}

	if total := fixture.tokensOfPlayer(t, playerID); total != 100 {
		t.Errorf("tras tres sincronizaciones hay %d tokens, se esperaban 100", total)
	}
	if quotes := fixture.countQuotes(t, playerID); quotes != 1 {
		t.Errorf("tras tres sincronizaciones hay %d cotizaciones, se esperaba una sola inicial", quotes)
	}
}

func TestUnJugadorNuevoEntraAlMercadoEnLaSiguienteSincronizacion(t *testing.T) {
	fixture := newMarketFixture(t)
	fixture.seedSuperuser(t)
	primero := fixture.seedPlayer(t)

	if err := fixture.marketMaker.Bootstrap(t.Context()); err != nil {
		t.Fatalf("la primera sincronización devolvió %v", err)
	}

	segundo := fixture.seedPlayer(t)

	if err := fixture.marketMaker.Bootstrap(t.Context()); err != nil {
		t.Fatalf("la segunda sincronización devolvió %v", err)
	}

	for _, playerID := range []int64{primero, segundo} {
		if total := fixture.tokensOfPlayer(t, playerID); total != 100 {
			t.Errorf("el jugador %d tiene %d tokens, se esperaban 100", playerID, total)
		}
		if _, found := fixture.liveMarketMakerOrder(t, playerID); !found {
			t.Errorf("el jugador %d quedó sin oferta del creador de mercado", playerID)
		}
	}
}

func TestUnJugadorDesactivadoNoEntraAlMercado(t *testing.T) {
	fixture := newMarketFixture(t)
	fixture.seedSuperuser(t)
	inactivo := fixture.seedPlayer(t)
	fixture.deactivatePlayer(t, inactivo)

	if err := fixture.marketMaker.Bootstrap(t.Context()); err != nil {
		t.Fatalf("Bootstrap devolvió %v", err)
	}

	if total := fixture.tokensOfPlayer(t, inactivo); total != 0 {
		t.Errorf("se emitieron %d tokens de un jugador inactivo, se esperaba cero", total)
	}
	if quotes := fixture.countQuotes(t, inactivo); quotes != 0 {
		t.Errorf("se cotizó %d veces a un jugador inactivo, se esperaba cero", quotes)
	}
}

func newSyncServiceWithBootstrap(spy *bootstrapSpy) *service.PlayerSyncServiceImpl {
	adapter := &mockFootballDataClient{
		fetchLeaguePlayersFn: func(ctx context.Context, leagueCode string) ([]model.Player, error) {
			return []model.Player{{ExternalID: 1, Name: "Jugador", LeagueCode: leagueCode}}, nil
		},
	}

	return service.NewPlayerSyncService(
		adapter,
		&mockPlayerRepository{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		service.WithMarketBootstrap(spy),
	)
}

func TestLaSincronizacionDisparaElBootstrapDelMercado(t *testing.T) {
	spy := &bootstrapSpy{}

	if _, err := newSyncServiceWithBootstrap(spy).SyncPlayers(t.Context(), "actor@test.local"); err != nil {
		t.Fatalf("SyncPlayers devolvió %v", err)
	}

	if spy.calls != 1 {
		t.Errorf("el bootstrap se llamó %d veces, se esperaba exactamente 1 al final de la sincronización", spy.calls)
	}
}

func TestElFalloDelBootstrapNoRompeLaSincronizacion(t *testing.T) {
	spy := &bootstrapSpy{err: errors.New("el mercado no se pudo preparar")}

	response, err := newSyncServiceWithBootstrap(spy).SyncPlayers(t.Context(), "actor@test.local")
	if err != nil {
		t.Fatalf("SyncPlayers devolvió %v: un bootstrap fallido no puede tumbar la sincronización", err)
	}
	if response.Status != "success" {
		t.Errorf("la sincronización informó %q, se esperaba success", response.Status)
	}
	if spy.calls != 1 {
		t.Errorf("el bootstrap se llamó %d veces, se esperaba 1", spy.calls)
	}
}

func TestSinBootstrapConfiguradoLaSincronizacionSigueAndando(t *testing.T) {
	adapter := &mockFootballDataClient{
		fetchLeaguePlayersFn: func(ctx context.Context, leagueCode string) ([]model.Player, error) {
			return []model.Player{{ExternalID: 1, Name: "Jugador", LeagueCode: leagueCode}}, nil
		},
	}

	sync := service.NewPlayerSyncService(
		adapter,
		&mockPlayerRepository{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)

	if _, err := sync.SyncPlayers(t.Context(), "actor@test.local"); err != nil {
		t.Fatalf("SyncPlayers sin creador de mercado devolvió %v", err)
	}
}
