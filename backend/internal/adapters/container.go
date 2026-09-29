package adapters

import (
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/configuration"
)

// Container arma los adaptadores de la capa con la configuración inyectada.
// Es el único lugar donde el secreto de firma, los dos tiempos de vida y el
// factor de costo de bcrypt se aplican.
type Container struct {
	Password *Password
	JWT *JWT
}

func NewContainer(cfg *configuration.Cfg) *Container {
	return &Container{
		Password: NewPassword(cfg.BcryptCost),
		JWT: NewJWT(cfg.JWTSecret, cfg.AccessTTL, cfg.RefreshTTL, time.Now),
	}
}

