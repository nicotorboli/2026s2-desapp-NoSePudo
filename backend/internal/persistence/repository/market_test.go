package repository_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/dao"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/repository"
)

var marketSeedCounter int64

func newMarketRepository(db *sql.DB) *repository.MarketRepository {
	return repository.NewMarketRepository(
		db,
		dao.NewPlayerDao(db),
		dao.NewAccountDao(db),
		dao.NewPositionDao(db),
		dao.NewOrderDao(db),
		dao.NewTradeDao(db),
		dao.NewLedgerDao(db),
		dao.NewQuoteDao(db),
		dao.NewAuditDao(db),
	)
}

func seedMarketUser(t *testing.T, db *sql.DB) int64 {
	t.Helper()

	marketSeedCounter++

	var id int64
	err := db.QueryRowContext(t.Context(),
		`INSERT INTO users (email, password_hash, privilege, active) VALUES ($1, $2, $3, TRUE) RETURNING id`,
		fmt.Sprintf("mercado-%d@test.local", marketSeedCounter),
		fakeBcryptHash,
		int16(model.PrivilegeUser),
	).Scan(&id)
	if err != nil {
		t.Fatalf("no se pudo sembrar un usuario: %v", err)
	}

	return id
}

func seedMarketPlayer(t *testing.T, db *sql.DB) int64 {
	t.Helper()

	marketSeedCounter++

	var id int64
	err := db.QueryRowContext(t.Context(),
		`INSERT INTO players (external_id, name, club_name, league_name, league_code, position)
		 VALUES ($1, $2, 'Arsenal FC', 'Premier League', 'PL', 'Offence') RETURNING id`,
		marketSeedCounter,
		fmt.Sprintf("Jugador %d", marketSeedCounter),
	).Scan(&id)
	if err != nil {
		t.Fatalf("no se pudo sembrar un jugador: %v", err)
	}

	return id
}

func seedMarketAccount(t *testing.T, db *sql.DB, account model.Account) {
	t.Helper()

	if err := dao.NewAccountDao(db).Insert(t.Context(), nil, account); err != nil {
		t.Fatalf("no se pudo sembrar la cuenta: %v", err)
	}
}

func seedMarketPosition(t *testing.T, db *sql.DB, position model.Position) {
	t.Helper()

	if _, err := dao.NewPositionDao(db).InsertIfAbsent(t.Context(), nil, position); err != nil {
		t.Fatalf("no se pudo sembrar la posición: %v", err)
	}
}

func seedMarketOrder(t *testing.T, db *sql.DB, order model.Order) model.Order {
	t.Helper()

	stored, err := dao.NewOrderDao(db).Insert(t.Context(), nil, order)
	if err != nil {
		t.Fatalf("no se pudo sembrar la orden: %v", err)
	}

	return stored
}

func seedMarketQuote(t *testing.T, db *sql.DB, playerID int64, price model.Credits) {
	t.Helper()

	if _, err := dao.NewQuoteDao(db).InsertInitialIfAbsent(t.Context(), nil, playerID, price); err != nil {
		t.Fatalf("no se pudo sembrar la cotización: %v", err)
	}
}

func aRestingOrder(userID, playerID int64, side model.OrderSide, price model.Credits, quantity int64) model.Order {
	return model.Order{
		Side:        side,
		TimeInForce: model.GoodTillCancelled,
		Status:      model.StatusPending,
		UserID:      userID,
		PlayerID:    playerID,
		Quantity:    quantity,
		Price:       price,
	}
}

func TestMarketRepositoryTransactionBoundary(t *testing.T) {
	db := startPostgres(t)
	repo := newMarketRepository(db)

	t.Run("lee la cuenta, la posicion, la cotizacion y el libro opuesto bajo el lock", func(t *testing.T) {
		assertReadsTheLockedState(t, db, repo)
	})

	t.Run("el libro opuesto deja afuera lo que no se cruza", func(t *testing.T) {
		assertBookExcludesWhatDoesNotCross(t, db, repo)
	})

	t.Run("un jugador inexistente no abre la operacion", func(t *testing.T) {
		assertUnknownPlayerNeverRunsTheOperation(t, repo)
	})

	t.Run("el error de la operacion se propaga tal cual", func(t *testing.T) {
		assertOperationErrorIsPropagated(t, db, repo)
	})

	t.Run("la cuenta, la posicion y la cotizacion ausentes devuelven sus centinelas", func(t *testing.T) {
		assertAbsentStateReturnsItsSentinels(t, db, repo)
	})

	t.Run("el lock del jugador serializa a quien compite y se libera al terminar", func(t *testing.T) {
		assertPlayerLockSerializesCompetingTransactions(t, db, repo)
	})
}

func assertReadsTheLockedState(t *testing.T, db *sql.DB, repo *repository.MarketRepository) {
	t.Helper()

	buyer := seedMarketUser(t, db)
	seller := seedMarketUser(t, db)
	playerID := seedMarketPlayer(t, db)

	account := model.Account{
		UserID:           buyer,
		AvailableBalance: model.CreditsFromUnits(990),
		ReservedBalance:  model.CreditsFromUnits(10),
	}
	position := model.Position{
		UserID:            seller,
		PlayerID:          playerID,
		AvailableQuantity: 7,
		ReservedQuantity:  3,
		CostBasis:         model.CreditsFromUnits(12),
	}

	seedMarketAccount(t, db, account)
	seedMarketPosition(t, db, position)
	seedMarketQuote(t, db, playerID, model.CreditsFromUnits(1))
	resting := seedMarketOrder(t, db, aRestingOrder(seller, playerID, model.SideSell, model.CreditsFromUnits(2), 5))

	err := repo.WithPlayerLocked(t.Context(), playerID, func(ctx context.Context, session *repository.MarketSession) error {
		locked, accountErr := session.LockAccount(ctx, buyer)
		if accountErr != nil {
			return fmt.Errorf("LockAccount: %w", accountErr)
		}
		if locked != account {
			t.Errorf("la cuenta bloqueada es %+v, se esperaba %+v", locked, account)
		}

		held, positionErr := session.LockPosition(ctx, seller, playerID)
		if positionErr != nil {
			return fmt.Errorf("LockPosition: %w", positionErr)
		}
		if held != position {
			t.Errorf("la posición bloqueada es %+v, se esperaba %+v", held, position)
		}

		quote, quoteErr := session.LatestQuote(ctx, playerID)
		if quoteErr != nil {
			return fmt.Errorf("LatestQuote: %w", quoteErr)
		}
		if quote.Price != model.CreditsFromUnits(1) {
			t.Errorf("la cotización vigente es %d, se esperaba un crédito", quote.Price)
		}

		book, bookErr := session.LockOppositeBook(ctx, playerID, model.SideBuy)
		if bookErr != nil {
			return fmt.Errorf("LockOppositeBook: %w", bookErr)
		}
		if len(book) != 1 || book[0].ID != resting.ID {
			t.Errorf("el libro opuesto es %+v, se esperaba sólo la orden %d", book, resting.ID)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("WithPlayerLocked devolvió %v", err)
	}
}

func assertBookExcludesWhatDoesNotCross(t *testing.T, db *sql.DB, repo *repository.MarketRepository) {
	t.Helper()

	seller := seedMarketUser(t, db)
	playerID := seedMarketPlayer(t, db)
	otherPlayerID := seedMarketPlayer(t, db)

	live := seedMarketOrder(t, db, aRestingOrder(seller, playerID, model.SideSell, model.CreditsFromUnits(1), 4))
	sameSide := seedMarketOrder(t, db, aRestingOrder(seller, playerID, model.SideBuy, model.CreditsFromUnits(1), 4))
	otherPlayer := seedMarketOrder(t, db, aRestingOrder(seller, otherPlayerID, model.SideSell, model.CreditsFromUnits(1), 4))

	cancelled := seedMarketOrder(t, db, aRestingOrder(seller, playerID, model.SideSell, model.CreditsFromUnits(1), 4))
	cancelled.Status = model.StatusCancelled
	if err := dao.NewOrderDao(db).Update(t.Context(), nil, cancelled); err != nil {
		t.Fatalf("no se pudo cancelar la orden sembrada: %v", err)
	}

	err := repo.WithPlayerLocked(t.Context(), playerID, func(ctx context.Context, session *repository.MarketSession) error {
		book, bookErr := session.LockOppositeBook(ctx, playerID, model.SideBuy)
		if bookErr != nil {
			return bookErr
		}

		if len(book) != 1 || book[0].ID != live.ID {
			t.Errorf("el libro opuesto es %+v, se esperaba sólo la orden %d y no %d, %d ni %d",
				book, live.ID, sameSide.ID, otherPlayer.ID, cancelled.ID)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("WithPlayerLocked devolvió %v", err)
	}
}

func assertUnknownPlayerNeverRunsTheOperation(t *testing.T, repo *repository.MarketRepository) {
	t.Helper()

	reached := false
	err := repo.WithPlayerLocked(t.Context(), 999999, func(context.Context, *repository.MarketSession) error {
		reached = true
		return nil
	})

	if !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("WithPlayerLocked devolvió %v, se esperaba ErrNotFound", err)
	}
	if reached {
		t.Error("la operación corrió con un jugador que no existe")
	}
}

func assertOperationErrorIsPropagated(t *testing.T, db *sql.DB, repo *repository.MarketRepository) {
	t.Helper()

	playerID := seedMarketPlayer(t, db)
	failure := errors.New("la operación no se pudo completar")

	err := repo.WithPlayerLocked(t.Context(), playerID, func(context.Context, *repository.MarketSession) error {
		return failure
	})
	if !errors.Is(err, failure) {
		t.Errorf("WithPlayerLocked devolvió %v, se esperaba el error de la operación", err)
	}
}

func assertAbsentStateReturnsItsSentinels(t *testing.T, db *sql.DB, repo *repository.MarketRepository) {
	t.Helper()

	userID := seedMarketUser(t, db)
	playerID := seedMarketPlayer(t, db)

	err := repo.WithPlayerLocked(t.Context(), playerID, func(ctx context.Context, session *repository.MarketSession) error {
		if _, accountErr := session.LockAccount(ctx, userID); !errors.Is(accountErr, model.ErrAccountNotFound) {
			t.Errorf("LockAccount devolvió %v, se esperaba ErrAccountNotFound", accountErr)
		}
		if _, positionErr := session.LockPosition(ctx, userID, playerID); !errors.Is(positionErr, model.ErrPositionNotFound) {
			t.Errorf("LockPosition devolvió %v, se esperaba ErrPositionNotFound", positionErr)
		}
		if _, quoteErr := session.LatestQuote(ctx, playerID); !errors.Is(quoteErr, model.ErrQuoteNotFound) {
			t.Errorf("LatestQuote devolvió %v, se esperaba ErrQuoteNotFound", quoteErr)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("WithPlayerLocked devolvió %v", err)
	}
}

func assertPlayerLockSerializesCompetingTransactions(t *testing.T, db *sql.DB, repo *repository.MarketRepository) {
	t.Helper()

	playerDao := dao.NewPlayerDao(db)
	playerID := seedMarketPlayer(t, db)

	err := repo.WithPlayerLocked(t.Context(), playerID, func(context.Context, *repository.MarketSession) error {
		blockedCtx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
		defer cancel()

		competing, beginErr := db.BeginTx(blockedCtx, nil)
		if beginErr != nil {
			return fmt.Errorf("abrir la transacción que compite: %w", beginErr)
		}
		defer func() { _ = competing.Rollback() }()

		_, lockErr := playerDao.LockPlayer(blockedCtx, competing, playerID)
		if lockErr == nil {
			t.Error("la transacción que compite tomó el lock mientras la operación lo tenía")
		}
		if !errors.Is(blockedCtx.Err(), context.DeadlineExceeded) {
			t.Errorf("la transacción que compite falló con %v antes de vencer el plazo: se esperaba que quedara esperando el lock", lockErr)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("WithPlayerLocked devolvió %v", err)
	}

	freed, beginErr := db.BeginTx(t.Context(), nil)
	if beginErr != nil {
		t.Fatalf("no se pudo abrir la transacción posterior: %v", beginErr)
	}
	defer func() { _ = freed.Rollback() }()

	if _, lockErr := playerDao.LockPlayer(t.Context(), freed, playerID); lockErr != nil {
		t.Errorf("tomar el lock después de la operación devolvió %v: la transacción no se cerró", lockErr)
	}
}

const settlementQuantity = int64(4)

type settlementFixture struct {
	settlement repository.ExecutionSettlement
	buyer      int64
	seller     int64
	playerID   int64
	total      model.Credits
}

func mustSucceed(t *testing.T, what string, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("%s: %v", what, err)
	}
}

func newSettlementFixture(t *testing.T, db *sql.DB) settlementFixture {
	t.Helper()

	price := model.CreditsFromUnits(2)
	total := price.Times(settlementQuantity)

	buyer := seedMarketUser(t, db)
	seller := seedMarketUser(t, db)
	playerID := seedMarketPlayer(t, db)

	buyerAccount := model.Account{UserID: buyer, AvailableBalance: model.CreditsFromUnits(1000)}
	sellerAccount := model.Account{UserID: seller}
	buyerPosition := model.Position{UserID: buyer, PlayerID: playerID}
	sellerPosition := model.Position{
		UserID:            seller,
		PlayerID:          playerID,
		AvailableQuantity: 10,
		CostBasis:         model.CreditsFromUnits(10),
	}

	seedMarketAccount(t, db, buyerAccount)
	seedMarketAccount(t, db, sellerAccount)
	seedMarketPosition(t, db, sellerPosition)

	buyOrder := seedMarketOrder(t, db, aRestingOrder(buyer, playerID, model.SideBuy, price, settlementQuantity))
	sellOrder := seedMarketOrder(t, db, aRestingOrder(seller, playerID, model.SideSell, price, settlementQuantity))

	executedAt := time.Now().UTC()

	filledBuyOrder := buyOrder
	mustSucceed(t, "ejecutar la orden de compra", filledBuyOrder.Execute(settlementQuantity, executedAt))

	filledSellOrder := sellOrder
	mustSucceed(t, "ejecutar la orden de venta", filledSellOrder.Execute(settlementQuantity, executedAt))

	settledBuyerAccount := buyerAccount
	mustSucceed(t, "reservar los créditos del comprador", settledBuyerAccount.Reserve(total))
	mustSucceed(t, "liquidar la reserva del comprador", settledBuyerAccount.SettleReservation(total))

	creditedSellerAccount := sellerAccount
	mustSucceed(t, "acreditar al vendedor", creditedSellerAccount.Credit(total))

	deliveredSellerPosition := sellerPosition
	mustSucceed(t, "reservar los tokens del vendedor", deliveredSellerPosition.ReserveTokens(settlementQuantity))
	_, deliverErr := deliveredSellerPosition.DeliverReservedTokens(settlementQuantity)
	mustSucceed(t, "entregar los tokens del vendedor", deliverErr)

	receivedBuyerPosition := buyerPosition
	mustSucceed(t, "acreditar los tokens al comprador", receivedBuyerPosition.Receive(settlementQuantity, total))

	return settlementFixture{
		settlement: repository.ExecutionSettlement{
			BuyOrder:       repository.OrderChange{Before: buyOrder, After: filledBuyOrder},
			SellOrder:      repository.OrderChange{Before: sellOrder, After: filledSellOrder},
			BuyerPosition:  repository.PositionChange{Before: buyerPosition, After: receivedBuyerPosition},
			SellerPosition: repository.PositionChange{Before: sellerPosition, After: deliveredSellerPosition},
			BuyerAccount:   repository.AccountChange{Before: buyerAccount, After: settledBuyerAccount},
			SellerAccount:  repository.AccountChange{Before: sellerAccount, After: creditedSellerAccount},
			Actor:          fmt.Sprintf("comprador-%d@test.local", buyer),
			Quantity:       settlementQuantity,
			Price:          price,
		},
		buyer:    buyer,
		seller:   seller,
		playerID: playerID,
		total:    total,
	}
}

func TestMarketRepositorySettlement(t *testing.T) {
	db := startPostgres(t)
	repo := newMarketRepository(db)

	t.Run("una ejecucion deja las ordenes, el dinero y los tokens donde corresponde", func(t *testing.T) {
		assertSettlementMovesEverything(t, db, repo)
	})

	t.Run("los asientos de la ejecucion suman cero", func(t *testing.T) {
		assertLedgerSettlementIsZeroSum(t, db, repo)
	})

	t.Run("la ejecucion queda auditada entidad por entidad", func(t *testing.T) {
		assertSettlementIsAudited(t, db, repo)
	})

	t.Run("una operacion que falla despues de liquidar no deja nada escrito", func(t *testing.T) {
		assertFailedOperationWritesNothing(t, db, repo)
	})
}

func settle(t *testing.T, repo *repository.MarketRepository, fixture settlementFixture) model.Trade {
	t.Helper()

	var trade model.Trade
	err := repo.WithPlayerLocked(t.Context(), fixture.playerID, func(ctx context.Context, session *repository.MarketSession) error {
		settled, settleErr := session.SettleExecution(ctx, fixture.settlement)
		trade = settled
		return settleErr
	})
	if err != nil {
		t.Fatalf("WithPlayerLocked devolvió %v", err)
	}

	return trade
}

func assertSettlementMovesEverything(t *testing.T, db *sql.DB, repo *repository.MarketRepository) {
	t.Helper()

	fixture := newSettlementFixture(t, db)
	trade := settle(t, repo, fixture)

	if trade.ID == 0 || trade.ExecutedAt.IsZero() {
		t.Errorf("la operación registrada es %+v: la base no le asignó identidad", trade)
	}
	if trade.TotalAmount != fixture.total {
		t.Errorf("el total de la operación es %d, se esperaba %d", trade.TotalAmount, fixture.total)
	}

	assertAccount(t, db, fixture.settlement.BuyerAccount.After)
	assertAccount(t, db, fixture.settlement.SellerAccount.After)
	assertPosition(t, db, fixture.settlement.BuyerPosition.After)
	assertPosition(t, db, fixture.settlement.SellerPosition.After)
	assertOrder(t, db, fixture.settlement.BuyOrder.After)
	assertOrder(t, db, fixture.settlement.SellOrder.After)
}

func assertLedgerSettlementIsZeroSum(t *testing.T, db *sql.DB, repo *repository.MarketRepository) {
	t.Helper()

	fixture := newSettlementFixture(t, db)
	trade := settle(t, repo, fixture)

	unbalanced, err := dao.NewLedgerDao(db).UnbalancedSettlementGroups(t.Context())
	if err != nil {
		t.Fatalf("UnbalancedSettlementGroups devolvió %v", err)
	}
	if len(unbalanced) != 0 {
		t.Errorf("los grupos de liquidación %v no suman cero", unbalanced)
	}

	var (
		entries int
		groups  int
		debit   int64
		credit  int64
	)
	err = db.QueryRowContext(t.Context(),
		`SELECT COUNT(*), COUNT(DISTINCT settlement_group),
		        COALESCE(SUM(amount) FILTER (WHERE user_id = $2), 0),
		        COALESCE(SUM(amount) FILTER (WHERE user_id = $3), 0)
		 FROM ledger_entries WHERE trade_id = $1`,
		trade.ID, fixture.buyer, fixture.seller,
	).Scan(&entries, &groups, &debit, &credit)
	if err != nil {
		t.Fatalf("no se pudieron leer los asientos: %v", err)
	}

	if entries != 2 || groups != 1 {
		t.Errorf("la ejecución dejó %d asientos en %d grupos, se esperaban 2 en uno solo", entries, groups)
	}
	if debit != -int64(fixture.total) {
		t.Errorf("al comprador se le asentaron %d créditos, se esperaban %d", debit, -int64(fixture.total))
	}
	if credit != int64(fixture.total) {
		t.Errorf("al vendedor se le asentaron %d créditos, se esperaban %d", credit, int64(fixture.total))
	}
}

func assertSettlementIsAudited(t *testing.T, db *sql.DB, repo *repository.MarketRepository) {
	t.Helper()

	fixture := newSettlementFixture(t, db)
	settle(t, repo, fixture)

	for _, entity := range []string{"market_order", "user_account", "user_position"} {
		var audited int
		err := db.QueryRowContext(t.Context(),
			`SELECT COUNT(*) FROM player_audit_logs WHERE entity_type = $1 AND actor = $2`,
			entity, fixture.settlement.Actor,
		).Scan(&audited)
		if err != nil {
			t.Fatalf("no se pudieron contar las auditorías de %s: %v", entity, err)
		}
		if audited != 2 {
			t.Errorf("la ejecución dejó %d auditorías de %s, se esperaban 2", audited, entity)
		}
	}
}

func assertFailedOperationWritesNothing(t *testing.T, db *sql.DB, repo *repository.MarketRepository) {
	t.Helper()

	fixture := newSettlementFixture(t, db)
	failure := errors.New("la operación se arrepintió después de liquidar")

	err := repo.WithPlayerLocked(t.Context(), fixture.playerID, func(ctx context.Context, session *repository.MarketSession) error {
		if _, settleErr := session.SettleExecution(ctx, fixture.settlement); settleErr != nil {
			return settleErr
		}
		return failure
	})
	if !errors.Is(err, failure) {
		t.Fatalf("WithPlayerLocked devolvió %v, se esperaba el error de la operación", err)
	}

	assertAccount(t, db, fixture.settlement.BuyerAccount.Before)
	assertAccount(t, db, fixture.settlement.SellerAccount.Before)
	assertPosition(t, db, fixture.settlement.SellerPosition.Before)
	assertOrder(t, db, fixture.settlement.BuyOrder.Before)

	trades, err := dao.NewTradeDao(db).CountByParticipant(t.Context(), fixture.buyer)
	if err != nil {
		t.Fatalf("CountByParticipant devolvió %v", err)
	}
	if trades != 0 {
		t.Errorf("quedaron %d operaciones registradas después del rollback", trades)
	}

	if _, positionErr := dao.NewPositionDao(db).LockByUserAndPlayer(t.Context(), beginTxFor(t, db), fixture.buyer, fixture.playerID); !errors.Is(positionErr, model.ErrPositionNotFound) {
		t.Errorf("la posición del comprador sobrevivió al rollback: %v", positionErr)
	}
}

func beginTxFor(t *testing.T, db *sql.DB) *sql.Tx {
	t.Helper()

	tx, err := db.BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatalf("no se pudo abrir la transacción: %v", err)
	}
	t.Cleanup(func() { _ = tx.Rollback() })

	return tx
}

func assertAccount(t *testing.T, db *sql.DB, expected model.Account) {
	t.Helper()

	stored, err := dao.NewAccountDao(db).GetByUser(t.Context(), expected.UserID)
	if err != nil {
		t.Fatalf("GetByUser devolvió %v", err)
	}
	if stored != expected {
		t.Errorf("la cuenta guardada es %+v, se esperaba %+v", stored, expected)
	}
}

func assertPosition(t *testing.T, db *sql.DB, expected model.Position) {
	t.Helper()

	positions, err := dao.NewPositionDao(db).ListByUser(t.Context(), expected.UserID)
	if err != nil {
		t.Fatalf("ListByUser devolvió %v", err)
	}

	for _, position := range positions {
		if position.PlayerID != expected.PlayerID {
			continue
		}
		if position != expected {
			t.Errorf("la posición guardada es %+v, se esperaba %+v", position, expected)
		}
		return
	}

	t.Errorf("no quedó guardada una posición del usuario %d en el jugador %d", expected.UserID, expected.PlayerID)
}

func assertOrder(t *testing.T, db *sql.DB, expected model.Order) {
	t.Helper()

	stored, err := dao.NewOrderDao(db).GetByID(t.Context(), nil, expected.ID)
	if err != nil {
		t.Fatalf("GetByID devolvió %v", err)
	}
	if stored.Status != expected.Status || stored.ExecutedQuantity != expected.ExecutedQuantity {
		t.Errorf("la orden %d quedó en %s con %d ejecutados, se esperaba %s con %d",
			expected.ID, stored.Status, stored.ExecutedQuantity, expected.Status, expected.ExecutedQuantity)
	}
}
