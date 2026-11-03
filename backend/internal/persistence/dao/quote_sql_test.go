package dao_test

import (
	"database/sql"
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

func seedPlayerScore(t *testing.T, db *sql.DB, playerID int64) int64 {
	t.Helper()

	var statisticsID int64
	if err := db.QueryRowContext(t.Context(),
		`INSERT INTO player_statistics (player_id, goals, assists, shots, key_passes, dribbles, tackles, interceptions, fouls, clears, blocks, was_dribbled, rating)
		 VALUES ($1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 7.0) RETURNING id`, playerID).Scan(&statisticsID); err != nil {
		t.Fatalf("no se pudieron sembrar las estadísticas: %v", err)
	}

	var histogramID int64
	if err := db.QueryRowContext(t.Context(),
		`INSERT INTO scoring_histograms (position, league_code, buckets) VALUES ('Offence', 'PL', '{0,1}') RETURNING id`).Scan(&histogramID); err != nil {
		t.Fatalf("no se pudo sembrar un histograma: %v", err)
	}

	var scoreID int64
	if err := db.QueryRowContext(t.Context(),
		`INSERT INTO player_scores (player_id, statistics_id, strategy_id, weights, raw_score, percentile, histogram_id)
		 VALUES ($1, $2, 1, '{}', 0.5, 0.5, $3) RETURNING id`,
		playerID, statisticsID, histogramID).Scan(&scoreID); err != nil {
		t.Fatalf("no se pudo sembrar un score: %v", err)
	}

	return scoreID
}

func TestQuoteSqlInsertPersisteElScoreID(t *testing.T) {
	db := startPostgres(t)
	quoteDao := dao.NewQuoteDao(db)
	playerID := seedPlayer(t, db)
	scoreID := seedPlayerScore(t, db, playerID)

	if _, err := quoteDao.Insert(t.Context(), nil, model.PlayerQuote{
		Strategy:        "rateada",
		StrategyVersion: "1",
		PlayerID:        playerID,
		Price:           model.CreditsFromUnits(51),
		ScoreID:         &scoreID,
	}); err != nil {
		t.Fatalf("Insert devolvió %v", err)
	}

	quote, err := quoteDao.ListByPlayer(t.Context(), playerID)
	if err != nil {
		t.Fatalf("ListByPlayer devolvió %v", err)
	}
	if len(quote) != 1 {
		t.Fatalf("ListByPlayer devolvió %d cotizaciones, se esperaba 1", len(quote))
	}
	if quote[0].ScoreID == nil || *quote[0].ScoreID != scoreID {
		t.Errorf("ScoreID = %v, se esperaba %d", quote[0].ScoreID, scoreID)
	}
}

func TestQuoteSqlInsertSinScoreIDLoDejaNulo(t *testing.T) {
	db := startPostgres(t)
	quoteDao := dao.NewQuoteDao(db)
	playerID := seedPlayer(t, db)

	if _, err := quoteDao.Insert(t.Context(), nil, model.PlayerQuote{
		Strategy:        "rateada",
		StrategyVersion: "1",
		PlayerID:        playerID,
		Price:           model.CreditsFromUnits(51),
	}); err != nil {
		t.Fatalf("Insert devolvió %v", err)
	}

	quotes, err := quoteDao.ListByPlayer(t.Context(), playerID)
	if err != nil {
		t.Fatalf("ListByPlayer devolvió %v", err)
	}
	if len(quotes) != 1 {
		t.Fatalf("ListByPlayer devolvió %d cotizaciones, se esperaba 1", len(quotes))
	}
	if quotes[0].ScoreID != nil {
		t.Errorf("ScoreID = %v, se esperaba nulo", *quotes[0].ScoreID)
	}
}

func TestQuoteSqlListByPlayerOrdenaLaMasRecientePrimero(t *testing.T) {
	db := startPostgres(t)
	quoteDao := dao.NewQuoteDao(db)
	playerID := seedPlayer(t, db)

	if _, err := quoteDao.InsertInitialIfAbsent(t.Context(), nil, playerID, model.CreditsFromUnits(1)); err != nil {
		t.Fatalf("InsertInitialIfAbsent devolvió %v", err)
	}
	for _, strategy := range []string{"primera", "segunda"} {
		if _, err := quoteDao.Insert(t.Context(), nil, model.PlayerQuote{
			Strategy:        strategy,
			StrategyVersion: "1",
			PlayerID:        playerID,
			Price:           model.CreditsFromUnits(2),
		}); err != nil {
			t.Fatalf("Insert de %q devolvió %v", strategy, err)
		}
	}

	quotes, err := quoteDao.ListByPlayer(t.Context(), playerID)
	if err != nil {
		t.Fatalf("ListByPlayer devolvió %v", err)
	}
	if len(quotes) != 3 {
		t.Fatalf("ListByPlayer devolvió %d cotizaciones, se esperaban 3", len(quotes))
	}
	if quotes[0].Strategy != "segunda" || quotes[2].Strategy != model.InitialQuoteStrategy {
		t.Errorf("el orden es %q ... %q, se esperaba segunda ... initial", quotes[0].Strategy, quotes[2].Strategy)
	}
}

func TestQuoteSqlListByPlayerSinCotizacionesDevuelveVacio(t *testing.T) {
	db := startPostgres(t)
	quoteDao := dao.NewQuoteDao(db)

	quotes, err := quoteDao.ListByPlayer(t.Context(), seedPlayer(t, db))
	if err != nil {
		t.Fatalf("ListByPlayer devolvió %v", err)
	}
	if len(quotes) != 0 {
		t.Errorf("ListByPlayer devolvió %d cotizaciones, se esperaba vacío", len(quotes))
	}
}
