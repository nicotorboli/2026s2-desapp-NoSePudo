package repository

import (
	"database/sql"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/model"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/dao"
)

type Container struct {
	Player             *PlayerRepositoryImpl
	PlayerStatistics   *PlayerStatisticsRepositoryImpl
	User               *UserRepository
	RefreshToken       *RefreshTokenRepository
	PlayerScore        *PlayerScoreRepositoryImpl
	ScoreHistogram     *ScoreHistogramRepositoryImpl
	StrategyActivation *StrategyActivationRepositoryImpl
	LeagueCoefficient  *LeagueCoefficientRepositoryImpl
	ScoringStrategy    *ScoringStrategyRepositoryImpl
	SyncSchedule       *SyncScheduleRepositoryImpl
	SyncRun            *SyncRunRepositoryImpl
	Market             *MarketRepository
	Portfolio          *PortfolioRepository
	Registration       *RegistrationRepository
}

type containerSettings struct {
	initialUserBalance model.Credits
}

type ContainerOption func(*containerSettings)

func WithInitialUserBalance(balance model.Credits) ContainerOption {
	return func(settings *containerSettings) {
		settings.initialUserBalance = balance
	}
}

func NewContainer(db *sql.DB, daos *dao.Container, options ...ContainerOption) *Container {
	var settings containerSettings
	for _, option := range options {
		option(&settings)
	}

	return &Container{
		Player:             NewPlayerRepository(db, daos.Player, daos.Audit),
		PlayerStatistics:   NewPlayerStatisticsRepository(db, daos.PlayerStatistics),
		User:               NewUserRepository(daos.User),
		RefreshToken:       NewRefreshTokenRepository(daos.RefreshToken),
		PlayerScore:        NewPlayerScoreRepository(db, daos.PlayerScore),
		ScoreHistogram:     NewScoreHistogramRepository(daos.ScoreHistogram),
		StrategyActivation: NewStrategyActivationRepository(daos.StrategyActivation),
		LeagueCoefficient:  NewLeagueCoefficientRepository(daos.LeagueCoefficient),
		ScoringStrategy:    NewScoringStrategyRepository(daos.ScoringStrategy),
		SyncSchedule:       NewSyncScheduleRepository(daos.SyncSchedule),
		SyncRun:            NewSyncRunRepository(db, daos.SyncRun),
		Market: NewMarketRepository(
			db,
			daos.Player,
			daos.Account,
			daos.Position,
			daos.Order,
			daos.Trade,
			daos.Ledger,
			daos.Quote,
			daos.Audit,
			daos.User,
		),
		Portfolio:    NewPortfolioRepository(daos.Account, daos.Position, daos.Quote, daos.Trade, daos.Player),
		Registration: NewRegistrationRepository(db, daos.User, daos.Account, daos.Ledger, settings.initialUserBalance),
	}
}
