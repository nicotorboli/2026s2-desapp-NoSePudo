package main

import (
	"fmt"
	"os"

	_ "github.com/lib/pq"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/cfg"
)

func main() {
	cfg := &cfg.Config{
		DBUrl:  os.Getenv("DBURL"),
		DBUser: os.Getenv("DBUSER"),
		DBPass: os.Getenv("DBPASSWORD"),
	}

	fmt.Println(cfg.DBUrl)
}
