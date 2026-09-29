package configuration

import (
	"fmt"
	"os"
)

type Configuration struct {
	PostgresDataSource string
	FootballDataAPIKey string
	Host               string
	Port               string
}

func Load() (*Configuration, error) {
	dsn := os.Getenv("NSPPSQLDS")
	apiKey := os.Getenv("NSPFOOTBALLDATAAPIKEY")
	host := os.Getenv("NSPHOST")
	port := os.Getenv("NSPPORT")

	if host == "" {
		host = "127.0.0.1"
	}

	if port == "" {
		port = "8080"
	}

	return &Configuration{
		PostgresDataSource: dsn,
		FootballDataAPIKey: apiKey,
		Host:               host,
		Port:               port,
	}, nil
}

func (c *Configuration) GetServerAddress() string {
	return fmt.Sprintf("%s:%s", c.Host, c.Port)
}
