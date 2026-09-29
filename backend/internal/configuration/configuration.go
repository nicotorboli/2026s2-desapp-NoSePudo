package configuration

import (
	"fmt"
	"os"
)

type Config struct {
	PostgresDataSource string
	FootballDataAPIKey string
	Host               string
	Port               string
}

// Load loads configuration from environment variables.
func Load() (*Config, error) {
	dsn := os.Getenv("NSPPSQLDS")
	if dsn == "" {
		// Fallback for standard PG environment variable if set
		dsn = os.Getenv("DATABASE_URL")
	}

	apiKey := os.Getenv("NSPFOOTBALLDATAAPIKEY")
	host := os.Getenv("NSPHOST")
	port := os.Getenv("NSPPORT")
	if port == "" {
		port = os.Getenv("PORT")
	}
	if port == "" {
		port = "8080"
	}
	if host == "" {
		host = "127.0.0.1"
	}

	return &Config{
		PostgresDataSource: dsn,
		FootballDataAPIKey: apiKey,
		Host:               host,
		Port:               port,
	}, nil
}

// LoadCfg maintains backward compatibility with the initial stub.
func LoadCfg() *Config {
	cfg, _ := Load()
	return cfg
}

func (c *Config) GetServerAddress() string {
	host := c.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := c.Port
	if port == "" {
		port = "8080"
	}
	return fmt.Sprintf("%s:%s", host, port)
}
