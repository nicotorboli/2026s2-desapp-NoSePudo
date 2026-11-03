package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// errNilDatabase es la guarda compartida por los DAOs: sin conexión no hay
// consulta que hacer, y conviene decirlo acá y no dejar que explote adentro
// del driver.
var errNilDatabase = errors.New("la conexión a la base de datos es nil")

var errLockNeedsTransaction = errors.New("tomar un lock de fila necesita una transacción")

type rowQuerier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func querierFor(tx *sql.Tx, db *sql.DB) (rowQuerier, error) {
	if tx != nil {
		return tx, nil
	}
	if db == nil {
		return nil, errNilDatabase
	}
	return db, nil
}

func execOn(ctx context.Context, tx *sql.Tx, db *sql.DB, query string, args ...any) (sql.Result, error) {
	querier, err := querierFor(tx, db)
	if err != nil {
		return nil, err
	}

	return querier.ExecContext(ctx, query, args...)
}

func requireOneRow(result sql.Result, absent error) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("contar las filas afectadas: %w", err)
	}
	if affected == 0 {
		return absent
	}

	return nil
}

type Container struct {
	Player             *PlayerSql
	PlayerStatistics   *PlayerStatisticsSql
	Audit              *AuditSql
	User               *UserSql
	RefreshToken       *RefreshTokenSql
	Schema             *SchemaSql
	PlayerScore        *PlayerScoreSql
	ScoreHistogram     *ScoreHistogramSql
	StrategyActivation *StrategyActivationSql
	LeagueCoefficient  *LeagueCoefficientSql
	ScoringStrategy    *ScoringStrategySql
	SyncSchedule       *SyncScheduleSql
	SyncRun            *SyncRunSql
	Account            *AccountSql
	Position           *PositionSql
	Order              *OrderSql
	Trade              *TradeSql
	Ledger             *LedgerSql
	Quote              *QuoteSql
}

func NewContainer(db *sql.DB) *Container {
	return &Container{
		Player:             NewPlayerDao(db),
		PlayerStatistics:   NewPlayerStatisticsDao(db),
		Audit:              NewAuditDao(db),
		User:               NewUserDao(db),
		RefreshToken:       NewRefreshTokenDao(db),
		Schema:             NewSchemaDao(db),
		PlayerScore:        NewPlayerScoreDao(db),
		ScoreHistogram:     NewScoreHistogramDao(db),
		StrategyActivation: NewStrategyActivationDao(db),
		LeagueCoefficient:  NewLeagueCoefficientDao(db),
		ScoringStrategy:    NewScoringStrategyDao(db),
		SyncSchedule:       NewSyncScheduleDao(db),
		SyncRun:            NewSyncRunDao(db),
		Account:            NewAccountDao(db),
		Position:           NewPositionDao(db),
		Order:              NewOrderDao(db),
		Trade:              NewTradeDao(db),
		Ledger:             NewLedgerDao(db),
		Quote:              NewQuoteDao(db),
	}
}
