package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/dao"
)

var (
	ErrP2PDatabaseNil = errors.New("database connection is nil")
)

type P2PRepository struct {
	db          *sql.DB
	sessionDao  *dao.P2PSessionSql
	positionDao *dao.PositionSql
	accountDao  *dao.AccountSql
	tradeDao    *dao.TradeSql
	ledgerDao   *dao.LedgerSql
	auditDao    dao.AuditDAO
	playerDao   *dao.PlayerSql
	userDao     *dao.UserSql
}

func NewP2PRepository(
	db *sql.DB,
	sessionDao *dao.P2PSessionSql,
	positionDao *dao.PositionSql,
	accountDao *dao.AccountSql,
	tradeDao *dao.TradeSql,
	ledgerDao *dao.LedgerSql,
	auditDao dao.AuditDAO,
	playerDao *dao.PlayerSql,
	userDao *dao.UserSql,
) *P2PRepository {
	return &P2PRepository{
		db:          db,
		sessionDao:  sessionDao,
		positionDao: positionDao,
		accountDao:  accountDao,
		tradeDao:    tradeDao,
		ledgerDao:   ledgerDao,
		auditDao:    auditDao,
		playerDao:   playerDao,
		userDao:     userDao,
	}
}

func (r *P2PRepository) Create(ctx context.Context, session model.P2PSession, actor string) (model.P2PSession, error) {
	if r.db == nil {
		return model.P2PSession{}, ErrP2PDatabaseNil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return model.P2PSession{}, fmt.Errorf("iniciar transacción p2p create: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	active, lockErr := r.playerDao.LockPlayer(ctx, tx, session.PlayerID)
	if lockErr != nil {
		if errors.Is(lockErr, dao.ErrNotFound) {
			return model.P2PSession{}, ErrNotFound
		}
		return model.P2PSession{}, lockErr
	}
	if !active {
		return model.P2PSession{}, ErrPlayerInactive
	}

	pos, err := r.positionDao.LockByUserAndPlayer(ctx, tx, session.SellerID, session.PlayerID)
	if err != nil {
		return model.P2PSession{}, err
	}

	if err = pos.ReserveTokens(session.Quantity); err != nil {
		return model.P2PSession{}, err
	}

	if err = r.positionDao.Update(ctx, tx, pos); err != nil {
		return model.P2PSession{}, fmt.Errorf("actualizar posición vendedor: %w", err)
	}

	created, err := r.sessionDao.Insert(ctx, tx, session)
	if err != nil {
		return model.P2PSession{}, err
	}

	if r.auditDao != nil {
		if err = r.auditDao.InsertAuditLog(ctx, tx, model.AuditLog{
			EntityType: "p2p_session",
			EntityID:   created.ID,
			Action:     "PLACE",
			Actor:      actor,
			DiffNew: map[string]any{
				"sellerId": created.SellerID,
				"playerId": created.PlayerID,
				"quantity": created.Quantity,
				"price":    int64(created.Price),
				"status":   string(created.Status),
			},
		}); err != nil {
			return model.P2PSession{}, fmt.Errorf("registrar auditoría p2p create: %w", err)
		}
	}

	if r.userDao != nil {
		if seller, userErr := r.userDao.GetByID(ctx, created.SellerID); userErr == nil {
			created.SellerEmail = seller.Email
		}
	}

	if err := tx.Commit(); err != nil {
		return model.P2PSession{}, fmt.Errorf("commit transacción p2p create: %w", err)
	}

	return created, nil
}

func (r *P2PRepository) GetByToken(ctx context.Context, token string) (model.P2PSession, error) {
	return r.sessionDao.GetByToken(ctx, token)
}

func (r *P2PRepository) Accept(ctx context.Context, token string, buyerID int64, now time.Time, actor string) (model.Trade, error) {
	if r.db == nil {
		return model.Trade{}, ErrP2PDatabaseNil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return model.Trade{}, fmt.Errorf("iniciar transacción p2p accept: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	session, err := r.sessionDao.LockByToken(ctx, tx, token)
	if err != nil {
		return model.Trade{}, err
	}

	if session.Status == model.P2PStatusCompleted && session.BuyerID != nil && *session.BuyerID == buyerID {
		return r.tradeDao.GetByP2PSessionID(ctx, tx, session.ID)
	}

	if err = session.Accept(buyerID, now); err != nil {
		return model.Trade{}, err
	}

	costBasisReleased, err := r.transferTokens(ctx, tx, session, buyerID)
	if err != nil {
		return model.Trade{}, err
	}

	totalAmount := session.TotalAmount()
	if err = r.settleBalances(ctx, tx, session.SellerID, buyerID, totalAmount); err != nil {
		return model.Trade{}, err
	}

	if err = r.sessionDao.UpdateStatus(ctx, tx, session); err != nil {
		return model.Trade{}, fmt.Errorf("actualizar sesión p2p: %w", err)
	}

	storedTrade, err := r.recordTradeAndLedger(ctx, tx, session, buyerID, totalAmount, now)
	if err != nil {
		return model.Trade{}, err
	}

	if r.auditDao != nil {
		if err = r.auditDao.InsertAuditLog(ctx, tx, model.AuditLog{
			EntityType: "p2p_session",
			EntityID:   session.ID,
			Action:     "ACCEPT",
			Actor:      actor,
			DiffNew: map[string]any{
				"status":      string(session.Status),
				"buyerId":     buyerID,
				"completedAt": session.CompletedAt,
				"costBasis":   int64(costBasisReleased),
			},
		}); err != nil {
			return model.Trade{}, fmt.Errorf("registrar auditoría p2p accept: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return model.Trade{}, fmt.Errorf("commit transacción p2p accept: %w", err)
	}

	return storedTrade, nil
}

func (r *P2PRepository) transferTokens(ctx context.Context, tx *sql.Tx, session model.P2PSession, buyerID int64) (model.Credits, error) {
	var sellerPos, buyerPos model.Position
	var err error

	if buyerID < session.SellerID {
		buyerPos, err = r.lockOrCreateBuyerPosition(ctx, tx, buyerID, session.PlayerID)
		if err != nil {
			return 0, err
		}
		sellerPos, err = r.positionDao.LockByUserAndPlayer(ctx, tx, session.SellerID, session.PlayerID)
		if err != nil {
			return 0, err
		}
	} else {
		sellerPos, err = r.positionDao.LockByUserAndPlayer(ctx, tx, session.SellerID, session.PlayerID)
		if err != nil {
			return 0, err
		}
		buyerPos, err = r.lockOrCreateBuyerPosition(ctx, tx, buyerID, session.PlayerID)
		if err != nil {
			return 0, err
		}
	}

	costBasisReleased, err := sellerPos.DeliverReservedTokens(session.Quantity)
	if err != nil {
		return 0, err
	}
	if err = r.positionDao.Update(ctx, tx, sellerPos); err != nil {
		return 0, fmt.Errorf("actualizar posición vendedor: %w", err)
	}

	totalAmount := session.TotalAmount()
	if err = buyerPos.Receive(session.Quantity, totalAmount); err != nil {
		return 0, err
	}
	if err = r.positionDao.Update(ctx, tx, buyerPos); err != nil {
		return 0, fmt.Errorf("actualizar posición comprador: %w", err)
	}

	return costBasisReleased, nil
}

func (r *P2PRepository) lockOrCreateBuyerPosition(ctx context.Context, tx *sql.Tx, buyerID, playerID int64) (model.Position, error) {
	buyerPos, err := r.positionDao.LockByUserAndPlayer(ctx, tx, buyerID, playerID)
	if err == nil {
		return buyerPos, nil
	}
	if !errors.Is(err, model.ErrPositionNotFound) {
		return model.Position{}, err
	}

	initialPos := model.Position{
		UserID:   buyerID,
		PlayerID: playerID,
	}
	if _, insertErr := r.positionDao.InsertIfAbsent(ctx, tx, initialPos); insertErr != nil {
		return model.Position{}, fmt.Errorf("crear posición comprador: %w", insertErr)
	}

	return r.positionDao.LockByUserAndPlayer(ctx, tx, buyerID, playerID)
}

func (r *P2PRepository) settleBalances(ctx context.Context, tx *sql.Tx, sellerID, buyerID int64, totalAmount model.Credits) error {
	var buyerAccount, sellerAccount model.Account
	var err error

	if buyerID < sellerID {
		buyerAccount, err = r.accountDao.LockByUser(ctx, tx, buyerID)
		if err != nil {
			return err
		}
		sellerAccount, err = r.accountDao.LockByUser(ctx, tx, sellerID)
		if err != nil {
			return err
		}
	} else {
		sellerAccount, err = r.accountDao.LockByUser(ctx, tx, sellerID)
		if err != nil {
			return err
		}
		buyerAccount, err = r.accountDao.LockByUser(ctx, tx, buyerID)
		if err != nil {
			return err
		}
	}

	if buyerAccount.AvailableBalance < totalAmount {
		return model.ErrInsufficientFunds
	}
	buyerAccount.AvailableBalance -= totalAmount
	if err = r.accountDao.UpdateBalances(ctx, tx, buyerAccount); err != nil {
		return fmt.Errorf("actualizar saldo comprador: %w", err)
	}

	sellerAccount.AvailableBalance += totalAmount
	if err = r.accountDao.UpdateBalances(ctx, tx, sellerAccount); err != nil {
		return fmt.Errorf("actualizar saldo vendedor: %w", err)
	}

	return nil
}

func (r *P2PRepository) recordTradeAndLedger(ctx context.Context, tx *sql.Tx, session model.P2PSession, buyerID int64, totalAmount model.Credits, now time.Time) (model.Trade, error) {
	sessionID := session.ID
	trade := model.Trade{
		P2PSessionID: &sessionID,
		BuyerID:      buyerID,
		SellerID:     session.SellerID,
		PlayerID:     session.PlayerID,
		Quantity:     session.Quantity,
		Price:        session.Price,
		TotalAmount:  totalAmount,
		ExecutedAt:   now,
	}

	storedTrade, err := r.tradeDao.Insert(ctx, tx, trade)
	if err != nil {
		return model.Trade{}, fmt.Errorf("insertar trade p2p: %w", err)
	}

	settlementGroup, err := r.ledgerDao.NextSettlementGroup(ctx, tx)
	if err != nil {
		return model.Trade{}, fmt.Errorf("obtener settlement group: %w", err)
	}

	tradeID := storedTrade.ID
	bID := buyerID
	sID := session.SellerID

	buyerEntry := model.LedgerEntry{
		TradeID:         &tradeID,
		UserID:          &bID,
		Reason:          model.ReasonTrade,
		Description:     fmt.Sprintf("compra p2p de %d tokens del jugador %d", session.Quantity, session.PlayerID),
		SettlementGroup: settlementGroup,
		Amount:          -totalAmount,
	}
	if _, insertErr := r.ledgerDao.Insert(ctx, tx, buyerEntry); insertErr != nil {
		return model.Trade{}, insertErr
	}

	sellerEntry := model.LedgerEntry{
		TradeID:         &tradeID,
		UserID:          &sID,
		Reason:          model.ReasonTrade,
		Description:     fmt.Sprintf("venta p2p de %d tokens del jugador %d", session.Quantity, session.PlayerID),
		SettlementGroup: settlementGroup,
		Amount:          totalAmount,
	}
	if _, insertErr := r.ledgerDao.Insert(ctx, tx, sellerEntry); insertErr != nil {
		return model.Trade{}, insertErr
	}

	return storedTrade, nil
}

func (r *P2PRepository) Cancel(ctx context.Context, token string, actorID int64, now time.Time, actor string) error {
	if r.db == nil {
		return ErrP2PDatabaseNil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("iniciar transacción p2p cancel: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	session, err := r.sessionDao.LockByToken(ctx, tx, token)
	if err != nil {
		return err
	}

	if session.SellerID != actorID {
		return errors.New("sólo el vendedor puede cancelar su propia oferta")
	}

	if err = session.Cancel(now); err != nil {
		return err
	}

	pos, err := r.positionDao.LockByUserAndPlayer(ctx, tx, session.SellerID, session.PlayerID)
	if err != nil {
		return err
	}

	if err = pos.ReleaseReservedTokens(session.Quantity); err != nil {
		return err
	}
	if err = r.positionDao.Update(ctx, tx, pos); err != nil {
		return fmt.Errorf("actualizar posición vendedor: %w", err)
	}

	if err = r.sessionDao.UpdateStatus(ctx, tx, session); err != nil {
		return fmt.Errorf("actualizar sesión p2p: %w", err)
	}

	if r.auditDao != nil {
		if err = r.auditDao.InsertAuditLog(ctx, tx, model.AuditLog{
			EntityType: "p2p_session",
			EntityID:   session.ID,
			Action:     "CANCEL",
			Actor:      actor,
			DiffNew: map[string]any{
				"status":      string(session.Status),
				"completedAt": session.CompletedAt,
			},
		}); err != nil {
			return fmt.Errorf("registrar auditoría p2p cancel: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit transacción p2p cancel: %w", err)
	}

	return nil
}

func (r *P2PRepository) ExpirePending(ctx context.Context, now time.Time) error {
	if r.db == nil {
		return ErrP2PDatabaseNil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("iniciar transacción p2p expire: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	sessions, err := r.sessionDao.ListPendingExpired(ctx, tx, now)
	if err != nil {
		return err
	}

	var errs []error
	for _, session := range sessions {
		if _, savepointErr := tx.ExecContext(ctx, "SAVEPOINT p2p_expire_session"); savepointErr != nil {
			errs = append(errs, fmt.Errorf("abrir savepoint p2p expire: %w", savepointErr))
			return errors.Join(errs...)
		}

		if expireErr := r.expireSession(ctx, tx, session.Token, now); expireErr != nil {
			if _, rollbackErr := tx.ExecContext(ctx, "ROLLBACK TO SAVEPOINT p2p_expire_session"); rollbackErr != nil {
				errs = append(errs, expireErr, fmt.Errorf("revertir savepoint p2p expire: %w", rollbackErr))
				return errors.Join(errs...)
			}
			errs = append(errs, expireErr)
			continue
		}

		if _, releaseErr := tx.ExecContext(ctx, "RELEASE SAVEPOINT p2p_expire_session"); releaseErr != nil {
			errs = append(errs, fmt.Errorf("liberar savepoint p2p expire: %w", releaseErr))
			return errors.Join(errs...)
		}
	}

	if err = tx.Commit(); err != nil {
		errs = append(errs, fmt.Errorf("commit transacción p2p expire: %w", err))
	}

	return errors.Join(errs...)
}

func (r *P2PRepository) expireSession(ctx context.Context, tx *sql.Tx, token string, now time.Time) error {
	lockedSession, err := r.sessionDao.LockByToken(ctx, tx, token)
	if err != nil {
		return err
	}

	if !lockedSession.IsLive() {
		return nil
	}

	if err = lockedSession.Expire(now); err != nil {
		return err
	}

	pos, posErr := r.positionDao.LockByUserAndPlayer(ctx, tx, lockedSession.SellerID, lockedSession.PlayerID)
	if posErr != nil && !errors.Is(posErr, model.ErrPositionNotFound) {
		return posErr
	}
	if posErr == nil {
		if err = pos.ReleaseReservedTokens(lockedSession.Quantity); err != nil {
			return err
		}
		if err = r.positionDao.Update(ctx, tx, pos); err != nil {
			return fmt.Errorf("actualizar posición vendedor: %w", err)
		}
	}

	if err = r.sessionDao.UpdateStatus(ctx, tx, lockedSession); err != nil {
		return fmt.Errorf("actualizar sesión p2p: %w", err)
	}

	if r.auditDao != nil {
		if err = r.auditDao.InsertAuditLog(ctx, tx, model.AuditLog{
			EntityType: "p2p_session",
			EntityID:   lockedSession.ID,
			Action:     "EXPIRE",
			Actor:      "system/p2p-expiration",
			DiffNew: map[string]any{
				"status":      string(lockedSession.Status),
				"completedAt": lockedSession.CompletedAt,
			},
		}); err != nil {
			return fmt.Errorf("registrar auditoría p2p expire: %w", err)
		}
	}

	return nil
}
