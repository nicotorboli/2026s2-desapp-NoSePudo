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
	minBcryptCost   = 4
	maxBcryptCost   = 31

	defaultHost       = "127.0.0.1"
	defaultPort       = "8080"
	defaultAccessTTL  = "15m"
	defaultRefreshTTL = "168h"
	defaultBcryptCost = "12"

	defaultSyncRunTimeout = "30m"
	defaultSyncInterval   = "24h"

	minSyncInterval = 15 * time.Minute
)

var ErrMissingJWTSecret = errors.New("NSP_JWT_SECRET es obligatorio y debe tener al menos 32 bytes")

type Cfg struct {
	PostgresDataSource     string
	FootballDataAPIKey     string
	Host                   string
	Port                   string
	JWTSecret              string
	SuperuserEmail         string
	SuperuserPassword      string
	AccessTTL              time.Duration
	RefreshTTL             time.Duration
	BcryptCost             int
	SyncRunTimeout         time.Duration
	SyncPlayersInterval    time.Duration
	SyncStatisticsInterval time.Duration
	SyncPlayersEnabled     bool
	SyncStatisticsEnabled  bool
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

	syncRunTimeout, err := durationFromEnv("NSP_SYNC_RUN_TIMEOUT", defaultSyncRunTimeout)
	if err != nil {
		return nil, err
	}

	syncPlayersInterval, err := syncIntervalFromEnv("NSP_SYNC_PLAYERS_INTERVAL")
	if err != nil {
		return nil, err
	}

	syncStatisticsInterval, err := syncIntervalFromEnv("NSP_SYNC_STATISTICS_INTERVAL")
	if err != nil {
		return nil, err
	}

	return &Cfg{
		PostgresDataSource:     os.Getenv("NSP_PSQL_DS"),
		FootballDataAPIKey:     os.Getenv("NSP_FOOTBALLDATA_API_KEY"),
		Host:                   cmp.Or(os.Getenv("NSP_HOST"), defaultHost),
		Port:                   cmp.Or(os.Getenv("NSP_PORT"), defaultPort),
		JWTSecret:              jwtSecret,
		SuperuserEmail:         os.Getenv("NSP_SUPERUSER_EMAIL"),
		SuperuserPassword:      os.Getenv("NSP_SUPERUSER_PASSWORD"),
		AccessTTL:              accessTTL,
		RefreshTTL:             refreshTTL,
		BcryptCost:             bcryptCost,
		SyncRunTimeout:         syncRunTimeout,
		SyncPlayersInterval:    syncPlayersInterval,
		SyncStatisticsInterval: syncStatisticsInterval,
		SyncPlayersEnabled:     boolFromEnv("NSP_SYNC_PLAYERS_ENABLED"),
		SyncStatisticsEnabled:  boolFromEnv("NSP_SYNC_STATISTICS_ENABLED"),
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

// syncIntervalFromEnv lee un intervalo de sincronización y exige el mínimo.
func syncIntervalFromEnv(name string) (time.Duration, error) {
	value, err := durationFromEnv(name, defaultSyncInterval)
	if err != nil {
		return 0, err
	}
	if value < minSyncInterval {
		return 0, fmt.Errorf("%s: %q debe ser de al menos %s", name, value, minSyncInterval)
	}
	return value, nil
}

// boolFromEnv lee un flag booleano; todo lo que no sea exactamente "true" es
// false, así que un valor ausente o mal escrito deshabilita por defecto.
func boolFromEnv(name string) bool {
	return os.Getenv(name) == "true"
}
