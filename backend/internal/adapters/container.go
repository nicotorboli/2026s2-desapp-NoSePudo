package adapters

import (
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/adapters/footballdata"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/configuration"
)

// footballDataRequestDelay espacia los pedidos a football-data.org para no
// pasarse del rate limit del plan gratuito.
const footballDataRequestDelay = 6 * time.Second

// Container arma los adaptadores de la capa con la configuración inyectada.
// Es el único lugar donde el secreto de firma, los dos tiempos de vida, el
// factor de costo de bcrypt y la API key de football-data se aplican.
type Container struct {
	Password     *Password
	JWT          *JWT
	FootballData *footballdata.FootballDataClient
}

func NewContainer(cfg *configuration.Cfg) *Container {
	return &Container{
		Password:     NewPassword(cfg.BcryptCost),
		JWT:          NewJWT(cfg.JWTSecret, cfg.AccessTTL, cfg.RefreshTTL, time.Now),
		FootballData: footballdata.NewClient(cfg.FootballDataAPIKey, footballDataRequestDelay),
	}
}
