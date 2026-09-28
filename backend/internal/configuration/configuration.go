package configuration

import (
	"fmt"
	"os"
)

type Cfg struct {
	PostgresDataSource string
	Host               string
	Port               string
}

func LoadCfg() *Cfg {
	return &Cfg{
		PostgresDataSource: os.Getenv("NSPPSQLDS"),
		Host:               os.Getenv("NSPHOST"),
		Port:               os.Getenv("NSPPORT"),
	}
}

func (c *Cfg) GetServerAddress() string {

	if c.Host == "" {
		c.Host = "127.0.0.1"
	}
	if c.Port == "" {
		c.Port = "8080"
	}

	return fmt.Sprintf("%s:%s", c.Host, c.Port)
}
