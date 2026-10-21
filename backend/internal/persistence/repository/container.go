package repository

import (
	"database/sql"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/dao"
)

type Container struct {
	Player       PlayerRepository
	User         *UserRepository
	RefreshToken *RefreshTokenRepository
	Market       *MarketRepository
	Portfolio    *PortfolioRepository
}

func NewContainer(db *sql.DB, daos *dao.Container) *Container {
	return &Container{
		Player:       NewPlayerRepository(db, daos.Player, daos.Audit),
		User:         NewUserRepository(daos.User),
		RefreshToken: NewRefreshTokenRepository(daos.RefreshToken),
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
		),
		Portfolio: NewPortfolioRepository(daos.Account, daos.Position, daos.Quote, daos.Trade),
	}
}
