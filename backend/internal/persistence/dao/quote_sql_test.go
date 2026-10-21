package dao_test

import (
	"errors"
	"testing"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/dao"
)

func TestQuoteSqlInsertInitialIfAbsentEscribeLaCotizacionDeMomentoCero(t *testing.T) {
	db := startPostgres(t)
	quoteDao := dao.NewQuoteDao(db)
	playerID := seedPlayer(t, db)

	inserted, err := quoteDao.InsertInitialIfAbsent(t.Context(), nil, playerID, model.CreditsFromUnits(1))
	if err != nil {
		t.Fatalf("InsertInitialIfAbsent devolvió %v", err)
	}
	if !inserted {
		t.Fatal("InsertInitialIfAbsent informó que no insertó la cotización inicial")
	}

	quote, err := quoteDao.GetLatestByPlayer(t.Context(), nil, playerID)
	if err != nil {
		t.Fatalf("GetLatestByPlayer devolvió %v", err)
	}
	if quote.Price != model.CreditsFromUnits(1) {
		t.Errorf("Price = %d, se esperaba 1 crédito", quote.Price)
	}
	if quote.Strategy != model.InitialQuoteStrategy || quote.StrategyVersion != model.InitialQuoteStrategyVersion {
		t.Errorf("la estrategia es %q versión %q", quote.Strategy, quote.StrategyVersion)
	}
}

func TestQuoteSqlInsertInitialIfAbsentEsIdempotente(t *testing.T) {
	db := startPostgres(t)
	quoteDao := dao.NewQuoteDao(db)
	playerID := seedPlayer(t, db)

	if _, err := quoteDao.InsertInitialIfAbsent(t.Context(), nil, playerID, model.CreditsFromUnits(1)); err != nil {
		t.Fatalf("el primer InsertInitialIfAbsent devolvió %v", err)
	}

	inserted, err := quoteDao.InsertInitialIfAbsent(t.Context(), nil, playerID, model.CreditsFromUnits(5))
	if err != nil {
		t.Fatalf("el segundo InsertInitialIfAbsent devolvió %v", err)
	}
	if inserted {
		t.Error("el segundo InsertInitialIfAbsent escribió una segunda cotización inicial")
	}

	quote, err := quoteDao.GetLatestByPlayer(t.Context(), nil, playerID)
	if err != nil {
		t.Fatalf("GetLatestByPlayer devolvió %v", err)
	}
	if quote.Price != model.CreditsFromUnits(1) {
		t.Errorf("Price = %d: la cotización inicial no se pisa", quote.Price)
	}
}

func TestQuoteSqlGetLatestByPlayerDevuelveLaMasReciente(t *testing.T) {
	db := startPostgres(t)
	quoteDao := dao.NewQuoteDao(db)
	playerID := seedPlayer(t, db)

	if _, err := quoteDao.InsertInitialIfAbsent(t.Context(), nil, playerID, model.CreditsFromUnits(1)); err != nil {
		t.Fatalf("InsertInitialIfAbsent devolvió %v", err)
	}
	if _, err := quoteDao.Insert(t.Context(), nil, model.PlayerQuote{
		Strategy:        "rendimiento",
		StrategyVersion: "1",
		PlayerID:        playerID,
		Price:           model.CreditsFromUnits(2),
	}); err != nil {
		t.Fatalf("Insert devolvió %v", err)
	}

	quote, err := quoteDao.GetLatestByPlayer(t.Context(), nil, playerID)
	if err != nil {
		t.Fatalf("GetLatestByPlayer devolvió %v", err)
	}
	if quote.Price != model.CreditsFromUnits(2) || quote.Strategy != "rendimiento" {
		t.Errorf("la cotización vigente es %+v, se esperaba la de rendimiento a 2 créditos", quote)
	}
}

func TestQuoteSqlGetLatestByPlayerSinCotizacion(t *testing.T) {
	db := startPostgres(t)
	quoteDao := dao.NewQuoteDao(db)

	_, err := quoteDao.GetLatestByPlayer(t.Context(), nil, seedPlayer(t, db))
	if !errors.Is(err, model.ErrQuoteNotFound) {
		t.Errorf("GetLatestByPlayer devolvió %v, se esperaba ErrQuoteNotFound", err)
	}
}

func TestQuoteSqlInsertRechazaUnaSegundaCotizacionInicial(t *testing.T) {
	db := startPostgres(t)
	quoteDao := dao.NewQuoteDao(db)
	playerID := seedPlayer(t, db)

	if _, err := quoteDao.InsertInitialIfAbsent(t.Context(), nil, playerID, model.CreditsFromUnits(1)); err != nil {
		t.Fatalf("InsertInitialIfAbsent devolvió %v", err)
	}

	_, err := quoteDao.Insert(t.Context(), nil, model.PlayerQuote{
		Strategy:        model.InitialQuoteStrategy,
		StrategyVersion: model.InitialQuoteStrategyVersion,
		PlayerID:        playerID,
		Price:           model.CreditsFromUnits(9),
	})
	if !errors.Is(err, model.ErrQuoteAlreadyExists) {
		t.Errorf("Insert devolvió %v, se esperaba ErrQuoteAlreadyExists", err)
	}
}

func TestQuoteSqlLatestPriceForPlayersResuelveVariosJugadores(t *testing.T) {
	db := startPostgres(t)
	quoteDao := dao.NewQuoteDao(db)
	primero := seedPlayer(t, db)
	segundo := seedPlayer(t, db)
	sinCotizar := seedPlayer(t, db)

	if _, err := quoteDao.InsertInitialIfAbsent(t.Context(), nil, primero, model.CreditsFromUnits(1)); err != nil {
		t.Fatalf("InsertInitialIfAbsent devolvió %v", err)
	}
	if _, err := quoteDao.InsertInitialIfAbsent(t.Context(), nil, segundo, model.CreditsFromUnits(1)); err != nil {
		t.Fatalf("InsertInitialIfAbsent devolvió %v", err)
	}
	if _, err := quoteDao.Insert(t.Context(), nil, model.PlayerQuote{
		Strategy:        "rendimiento",
		StrategyVersion: "1",
		PlayerID:        segundo,
		Price:           model.CreditsFromUnits(3),
	}); err != nil {
		t.Fatalf("Insert devolvió %v", err)
	}

	prices, err := quoteDao.LatestPriceForPlayers(t.Context(), []int64{primero, segundo, sinCotizar})
	if err != nil {
		t.Fatalf("LatestPriceForPlayers devolvió %v", err)
	}
	if prices[primero] != model.CreditsFromUnits(1) {
		t.Errorf("el primer jugador cotiza %d, se esperaba 1 crédito", prices[primero])
	}
	if prices[segundo] != model.CreditsFromUnits(3) {
		t.Errorf("el segundo jugador cotiza %d, se esperaban 3 créditos", prices[segundo])
	}
	if _, found := prices[sinCotizar]; found {
		t.Error("se devolvió un precio para un jugador sin cotización")
	}
}

func TestQuoteSqlLatestPriceForPlayersSinJugadoresNoConsulta(t *testing.T) {
	quoteDao := dao.NewQuoteDao(nil)

	prices, err := quoteDao.LatestPriceForPlayers(t.Context(), nil)
	if err != nil {
		t.Fatalf("LatestPriceForPlayers devolvió %v", err)
	}
	if len(prices) != 0 {
		t.Errorf("LatestPriceForPlayers devolvió %v, se esperaba vacío", prices)
	}
}
