package dao_test

import (
	"database/sql"
	"testing"
)

func insertarOrdenCruda(t *testing.T, db *sql.DB, userID, playerID int64, side string, price, quantity int64) int64 {
	t.Helper()

	var id int64
	err := db.QueryRowContext(t.Context(),
		`INSERT INTO market_orders (user_id, player_id, side, time_in_force, status, price, quantity)
		 VALUES ($1, $2, $3, 'GTC', 'PENDING', $4, $5) RETURNING id`,
		userID, playerID, side, price, quantity,
	).Scan(&id)
	if err != nil {
		t.Fatalf("no se pudo insertar la orden: %v", err)
	}

	return id
}

func TestEsquemaUqPositionsUserPlayerImpideDosPosicionesDelMismoPar(t *testing.T) {
	db := startPostgres(t)
	userID := seedUser(t, db)
	playerID := seedPlayer(t, db)

	const query = `INSERT INTO user_positions (user_id, player_id, available_quantity) VALUES ($1, $2, 100)`

	if _, err := db.ExecContext(t.Context(), query, userID, playerID); err != nil {
		t.Fatalf("la primera posición no se pudo insertar: %v", err)
	}
	if _, err := db.ExecContext(t.Context(), query, userID, playerID); err == nil {
		t.Error("la base aceptó una segunda posición del mismo usuario en el mismo jugador")
	}
}

func TestEsquemaUqQuotesPlayerInitialImpideDosCotizacionesIniciales(t *testing.T) {
	db := startPostgres(t)
	playerID := seedPlayer(t, db)

	const query = `INSERT INTO player_quotes (player_id, price, strategy, strategy_version) VALUES ($1, 10000, 'initial', '1')`

	if _, err := db.ExecContext(t.Context(), query, playerID); err != nil {
		t.Fatalf("la primera cotización inicial no se pudo insertar: %v", err)
	}
	if _, err := db.ExecContext(t.Context(), query, playerID); err == nil {
		t.Error("la base aceptó una segunda cotización inicial del mismo jugador")
	}

	_, err := db.ExecContext(t.Context(),
		`INSERT INTO player_quotes (player_id, price, strategy, strategy_version) VALUES ($1, 20000, 'rendimiento', '1')`,
		playerID,
	)
	if err != nil {
		t.Errorf("el índice parcial bloqueó una cotización de otra estrategia: %v", err)
	}
}

func TestEsquemaUqOrdersMarketMakerLiveImpideDosOfertasVivas(t *testing.T) {
	db := startPostgres(t)
	userID := seedUser(t, db)
	playerID := seedPlayer(t, db)

	const query = `INSERT INTO market_orders (user_id, player_id, side, time_in_force, status, price, quantity, market_maker)
	               VALUES ($1, $2, 'SELL', 'GTC', 'PENDING', 10000, 100, TRUE) RETURNING id`

	var firstID int64
	if err := db.QueryRowContext(t.Context(), query, userID, playerID).Scan(&firstID); err != nil {
		t.Fatalf("la primera oferta del creador de mercado no se pudo insertar: %v", err)
	}

	var secondID int64
	if err := db.QueryRowContext(t.Context(), query, userID, playerID).Scan(&secondID); err == nil {
		t.Error("la base aceptó una segunda oferta viva del creador de mercado")
	}

	if _, err := db.ExecContext(t.Context(),
		`UPDATE market_orders SET status = 'CANCELLED' WHERE id = $1`, firstID,
	); err != nil {
		t.Fatalf("no se pudo cancelar la primera oferta: %v", err)
	}

	if err := db.QueryRowContext(t.Context(), query, userID, playerID).Scan(&secondID); err != nil {
		t.Errorf("el índice parcial bloqueó la reposición después de cancelar: %v", err)
	}
}

func TestEsquemaRechazaUnSaldoNegativo(t *testing.T) {
	db := startPostgres(t)
	userID := seedUser(t, db)

	cases := []struct {
		name      string
		available int64
		reserved  int64
	}{
		{name: "disponible negativo", available: -1, reserved: 0},
		{name: "reservado negativo", available: 0, reserved: -1},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := db.ExecContext(t.Context(),
				`INSERT INTO user_accounts (user_id, available_balance, reserved_balance) VALUES ($1, $2, $3)`,
				userID, c.available, c.reserved,
			)
			if err == nil {
				t.Error("la base aceptó un saldo negativo")
			}
		})
	}
}

func TestEsquemaRechazaUnaPosicionNegativa(t *testing.T) {
	db := startPostgres(t)
	userID := seedUser(t, db)
	playerID := seedPlayer(t, db)

	cases := []struct {
		name      string
		available int64
		reserved  int64
		costBasis int64
	}{
		{name: "disponible negativo", available: -1, reserved: 0, costBasis: 0},
		{name: "reservado negativo", available: 0, reserved: -1, costBasis: 0},
		{name: "costo negativo", available: 0, reserved: 0, costBasis: -1},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := db.ExecContext(t.Context(),
				`INSERT INTO user_positions (user_id, player_id, available_quantity, reserved_quantity, cost_basis)
				 VALUES ($1, $2, $3, $4, $5)`,
				userID, playerID, c.available, c.reserved, c.costBasis,
			)
			if err == nil {
				t.Error("la base aceptó una posición negativa")
			}
		})
	}
}

func TestEsquemaRechazaUnPrecioOUnaCantidadNoPositivos(t *testing.T) {
	db := startPostgres(t)
	userID := seedUser(t, db)
	playerID := seedPlayer(t, db)

	cases := []struct {
		name     string
		price    int64
		quantity int64
	}{
		{name: "precio cero", price: 0, quantity: 10},
		{name: "precio negativo", price: -1, quantity: 10},
		{name: "cantidad cero", price: 10000, quantity: 0},
		{name: "cantidad negativa", price: 10000, quantity: -1},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := db.ExecContext(t.Context(),
				`INSERT INTO market_orders (user_id, player_id, side, time_in_force, status, price, quantity)
				 VALUES ($1, $2, 'BUY', 'GTC', 'PENDING', $3, $4)`,
				userID, playerID, c.price, c.quantity,
			)
			if err == nil {
				t.Error("la base aceptó una orden con precio o cantidad no positivos")
			}
		})
	}
}

func TestEsquemaRechazaUnaOperacionConTotalInconsistente(t *testing.T) {
	db := startPostgres(t)
	buyerID := seedUser(t, db)
	sellerID := seedUser(t, db)
	playerID := seedPlayer(t, db)
	buyOrderID := insertarOrdenCruda(t, db, buyerID, playerID, "BUY", 10000, 10)
	sellOrderID := insertarOrdenCruda(t, db, sellerID, playerID, "SELL", 10000, 10)

	_, err := db.ExecContext(t.Context(),
		`INSERT INTO market_trades (buy_order_id, sell_order_id, buyer_id, seller_id, player_id, price, quantity, total_amount)
		 VALUES ($1, $2, $3, $4, $5, 10000, 4, 99999)`,
		buyOrderID, sellOrderID, buyerID, sellerID, playerID,
	)
	if err == nil {
		t.Error("la base aceptó una operación cuyo total no es precio por cantidad")
	}
}

func TestEsquemaRechazaUnaOperacionConsigoMismo(t *testing.T) {
	db := startPostgres(t)
	userID := seedUser(t, db)
	playerID := seedPlayer(t, db)
	buyOrderID := insertarOrdenCruda(t, db, userID, playerID, "BUY", 10000, 10)
	sellOrderID := insertarOrdenCruda(t, db, userID, playerID, "SELL", 10000, 10)

	_, err := db.ExecContext(t.Context(),
		`INSERT INTO market_trades (buy_order_id, sell_order_id, buyer_id, seller_id, player_id, price, quantity, total_amount)
		 VALUES ($1, $2, $3, $3, $4, 10000, 4, 40000)`,
		buyOrderID, sellOrderID, userID, playerID,
	)
	if err == nil {
		t.Error("la base aceptó una operación entre un usuario y sí mismo")
	}
}
