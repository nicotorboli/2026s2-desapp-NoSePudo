package repository

import (
	"database/sql"

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
}

func NewContainer(db *sql.DB, daos *dao.Container) *Container {
	return &Container{
		Player:             NewPlayerRepository(db, daos.Player, daos.Audit),
		PlayerStatistics:   NewPlayerStatisticsRepository(db, daos.PlayerStatistics),
		User:               NewUserRepository(daos.User),
		RefreshToken:       NewRefreshTokenRepository(daos.RefreshToken),
		PlayerScore:        NewPlayerScoreRepository(db, daos.PlayerScore),
		ScoreHistogram:     NewScoreHistogramRepository(daos.ScoreHistogram),
		StrategyActivation: NewStrategyActivationRepository(daos.StrategyActivation),
		LeagueCoefficient:  NewLeagueCoefficientRepository(daos.LeagueCoefficient),
	}
}
