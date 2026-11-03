package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
)

type P2PSessionSql struct {
	Db *sql.DB
}

func NewP2PSessionDao(db *sql.DB) *P2PSessionSql {
	return &P2PSessionSql{Db: db}
}

func (dao *P2PSessionSql) Insert(ctx context.Context, tx *sql.Tx, session model.P2PSession) (model.P2PSession, error) {
	const query = `
		INSERT INTO p2p_sessions (seller_id, player_id, quantity, price, status, created_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, token, created_at, expires_at`

	querier, err := querierFor(tx, dao.Db)
	if err != nil {
		return model.P2PSession{}, err
	}

	status := session.Status
	if status == "" {
		status = model.P2PStatusPending
	}

	createdAt := session.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	expiresAt := session.ExpiresAt
	if expiresAt.IsZero() {
		expiresAt = createdAt.Add(5 * time.Minute)
	}

	err = querier.QueryRowContext(ctx, query,
		session.SellerID,
		session.PlayerID,
		session.Quantity,
		int64(session.Price),
		string(status),
		createdAt,
		expiresAt,
	).Scan(&session.ID, &session.Token, &session.CreatedAt, &session.ExpiresAt)
	if err != nil {
		return model.P2PSession{}, fmt.Errorf("insertar la sesión p2p: %w", err)
	}

	session.Status = status
	return session, nil
}

func (dao *P2PSessionSql) GetByToken(ctx context.Context, token string) (model.P2PSession, error) {
	const query = `
		SELECT s.id, s.token, s.seller_id, u_seller.email,
		       s.buyer_id, COALESCE(u_buyer.email, ''),
		       s.player_id, s.quantity, s.price, s.status,
		       s.created_at, s.expires_at, s.completed_at
		FROM p2p_sessions s
		JOIN users u_seller ON u_seller.id = s.seller_id
		LEFT JOIN users u_buyer ON u_buyer.id = s.buyer_id
		WHERE s.token = $1`

	if dao.Db == nil {
		return model.P2PSession{}, errNilDatabase
	}

	return scanP2PSession(dao.Db.QueryRowContext(ctx, query, token))
}

func (dao *P2PSessionSql) LockByToken(ctx context.Context, tx *sql.Tx, token string) (model.P2PSession, error) {
	const query = `
		SELECT s.id, s.token, s.seller_id, u_seller.email,
		       s.buyer_id, COALESCE(u_buyer.email, ''),
		       s.player_id, s.quantity, s.price, s.status,
		       s.created_at, s.expires_at, s.completed_at
		FROM p2p_sessions s
		JOIN users u_seller ON u_seller.id = s.seller_id
		LEFT JOIN users u_buyer ON u_buyer.id = s.buyer_id
		WHERE s.token = $1
		FOR UPDATE OF s`

	if tx == nil {
		return model.P2PSession{}, errLockNeedsTransaction
	}

	return scanP2PSession(tx.QueryRowContext(ctx, query, token))
}

func (dao *P2PSessionSql) UpdateStatus(ctx context.Context, tx *sql.Tx, session model.P2PSession) error {
	const query = `
		UPDATE p2p_sessions
		SET status = $2, buyer_id = $3, completed_at = $4
		WHERE id = $1`

	result, err := execOn(ctx, tx, dao.Db, query,
		session.ID,
		string(session.Status),
		session.BuyerID,
		session.CompletedAt,
	)
	if err != nil {
		return fmt.Errorf("actualizar el estado de la sesión p2p: %w", err)
	}

	return requireOneRow(result, model.ErrP2PSessionNotFound)
}

func (dao *P2PSessionSql) ListPendingExpired(ctx context.Context, now time.Time) ([]model.P2PSession, error) {
	const query = `
		SELECT s.id, s.token, s.seller_id, u_seller.email,
		       s.buyer_id, COALESCE(u_buyer.email, ''),
		       s.player_id, s.quantity, s.price, s.status,
		       s.created_at, s.expires_at, s.completed_at
		FROM p2p_sessions s
		JOIN users u_seller ON u_seller.id = s.seller_id
		LEFT JOIN users u_buyer ON u_buyer.id = s.buyer_id
		WHERE s.status = 'PENDING' AND s.expires_at <= $1
		ORDER BY s.expires_at ASC`

	if dao.Db == nil {
		return nil, errNilDatabase
	}

	rows, err := dao.Db.QueryContext(ctx, query, now)
	if err != nil {
		return nil, fmt.Errorf("listar sesiones p2p vencidas: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var sessions []model.P2PSession
	for rows.Next() {
		session, scanErr := scanP2PSessionFromRows(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		sessions = append(sessions, session)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("recorrer sesiones p2p vencidas: %w", err)
	}

	return sessions, nil
}

func scanP2PSession(row *sql.Row) (model.P2PSession, error) {
	var (
		session     model.P2PSession
		price       int64
		status      string
		buyerID     sql.NullInt64
		completedAt sql.NullTime
	)

	err := row.Scan(
		&session.ID,
		&session.Token,
		&session.SellerID,
		&session.SellerEmail,
		&buyerID,
		&session.BuyerEmail,
		&session.PlayerID,
		&session.Quantity,
		&price,
		&status,
		&session.CreatedAt,
		&session.ExpiresAt,
		&completedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.P2PSession{}, model.ErrP2PSessionNotFound
		}
		return model.P2PSession{}, fmt.Errorf("leer sesión p2p: %w", err)
	}

	session.Price = model.Credits(price)
	session.Status = model.P2PSessionStatus(status)
	if buyerID.Valid {
		id := buyerID.Int64
		session.BuyerID = &id
	}
	if completedAt.Valid {
		t := completedAt.Time
		session.CompletedAt = &t
	}

	return session, nil
}

func scanP2PSessionFromRows(rows *sql.Rows) (model.P2PSession, error) {
	var (
		session     model.P2PSession
		price       int64
		status      string
		buyerID     sql.NullInt64
		completedAt sql.NullTime
	)

	err := rows.Scan(
		&session.ID,
		&session.Token,
		&session.SellerID,
		&session.SellerEmail,
		&buyerID,
		&session.BuyerEmail,
		&session.PlayerID,
		&session.Quantity,
		&price,
		&status,
		&session.CreatedAt,
		&session.ExpiresAt,
		&completedAt,
	)
	if err != nil {
		return model.P2PSession{}, fmt.Errorf("leer sesión p2p: %w", err)
	}

	session.Price = model.Credits(price)
	session.Status = model.P2PSessionStatus(status)
	if buyerID.Valid {
		id := buyerID.Int64
		session.BuyerID = &id
	}
	if completedAt.Valid {
		t := completedAt.Time
		session.CompletedAt = &t
	}

	return session, nil
}
