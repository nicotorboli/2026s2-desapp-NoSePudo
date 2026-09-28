package configuration

import (
	"cmp"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	// minSecretLength es lo que "usable" significa para una clave HMAC-SHA256:
	// un secreto más corto es una clave más débil, y en silencio.
	minSecretLength = 32

	// Los límites del factor de costo de bcrypt. Se escriben acá en vez de
	// importar la librería para que la capa de configuración no dependa de
	// cómo se hashea.
	minBcryptCost = 4
	maxBcryptCost = 31

	defaultHost       = "127.0.0.1"
	defaultPort       = "8080"
	defaultAccessTTL  = "15m"
	defaultRefreshTTL = "168h"
	defaultBcryptCost = "12"
)

// ErrMissingJWTSecret es la razón por la que el servicio se niega a arrancar
// antes que servir credenciales que nadie va a poder verificar.
var ErrMissingJWTSecret = errors.New("NSP_JWT_SECRET es obligatorio y debe tener al menos 32 bytes")

// Cfg es la configuración inyectada. Los campos van de mayor a menor tamaño
// porque govet corre con fieldalignment.
type Cfg struct {
	PostgresDataSource string
	Host               string
	Port               string
	JWTSecret          string
	SuperuserEmail     string
	SuperuserPassword  string
	AccessTTL          time.Duration
	RefreshTTL         time.Duration
	BcryptCost         int
}

// LoadCfg lee la configuración del entorno y falla cuando algo que el
// servicio necesita no está o no sirve. Devolver el error en vez de entrar en
// pánico es lo que le deja a main la salida limpia que ya tiene, y lo que hace
// que esto se pueda testear sin levantar un servidor.
func LoadCfg() (*Cfg, error) {
	jwtSecret := os.Getenv("NSP_JWT_SECRET")
	if len(jwtSecret) < minSecretLength {
		return nil, ErrMissingJWTSecret
	}

	accessTTL, err := durationFromEnv("NSP_ACCESS_TTL", defaultAccessTTL)
	if err != nil {
		return nil, err
	}

	refreshTTL, err := durationFromEnv("NSP_REFRESH_TTL", defaultRefreshTTL)
	if err != nil {
		return nil, err
	}

	bcryptCost, err := bcryptCostFromEnv()
	if err != nil {
		return nil, err
	}

	return &Cfg{
		PostgresDataSource: os.Getenv("NSPPSQLDS"),
		Host:               cmp.Or(os.Getenv("NSPHOST"), defaultHost),
		Port:               cmp.Or(os.Getenv("NSPPORT"), defaultPort),
		JWTSecret:          jwtSecret,
		SuperuserEmail:     os.Getenv("NSP_SUPERUSER_EMAIL"),
		SuperuserPassword:  os.Getenv("NSP_SUPERUSER_PASSWORD"),
		AccessTTL:          accessTTL,
		RefreshTTL:         refreshTTL,
		BcryptCost:         bcryptCost,
	}, nil
}

func (c *Cfg) GetServerAddress() string {
	return fmt.Sprintf("%s:%s", c.Host, c.Port)
}

// HasSuperuserCredentials indica si hay con qué aprovisionar el superusuario.
// Que falten no es un error: el arranque simplemente se saltea ese paso.
func (c *Cfg) HasSuperuserCredentials() bool {
	return c.SuperuserEmail != "" && c.SuperuserPassword != ""
}

func durationFromEnv(name, fallback string) (time.Duration, error) {
	raw := cmp.Or(os.Getenv(name), fallback)

	value, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("%s: %q no es una duración válida: %w", name, raw, err)
	}
	if value <= 0 {
		return 0, fmt.Errorf("%s: %q debe ser una duración positiva", name, raw)
	}

	return value, nil
}

func bcryptCostFromEnv() (int, error) {
	const name = "NSP_BCRYPT_COST"
	raw := cmp.Or(os.Getenv(name), defaultBcryptCost)

	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s: %q no es un entero: %w", name, raw, err)
	}
	if value < minBcryptCost || value > maxBcryptCost {
		return 0, fmt.Errorf("%s: %d está fuera del rango [%d, %d]", name, value, minBcryptCost, maxBcryptCost)
	}

	return value, nil
}
