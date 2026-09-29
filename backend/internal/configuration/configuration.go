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
	minSecretLength = 32
	minBcryptCost = 4
	maxBcryptCost = 31

	defaultHost = "127.0.0.1"
	defaultPort = "8080"
	defaultAccessTTL = "15m"
	defaultRefreshTTL = "168h"
	defaultBcryptCost = "12"
)


var ErrMissingJWTSecret = errors.New("NSP_JWT_SECRET es obligatorio y debe tener al menos 32 bytes")

type Cfg struct {
	PostgresDataSource string
	Host string
	Port string
	JWTSecret string
	SuperuserEmail string
	SuperuserPassword string
	AccessTTL time.Duration
	RefreshTTL time.Duration
	BcryptCost int
}


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
		Host: cmp.Or(os.Getenv("NSPHOST"), defaultHost),
		Port: cmp.Or(os.Getenv("NSPPORT"), defaultPort),
		JWTSecret: jwtSecret,
		SuperuserEmail: os.Getenv("NSP_SUPERUSER_EMAIL"),
		SuperuserPassword: os.Getenv("NSP_SUPERUSER_PASSWORD"),
		AccessTTL: accessTTL,
		RefreshTTL: refreshTTL,
		BcryptCost: bcryptCost,
	}, nil
}

func (c *Cfg) GetServerAddress() string {
	return fmt.Sprintf("%s:%s", c.Host, c.Port)
}


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

