package main

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/cfg"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/controller"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/handler"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/dao"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/persistence/repository"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/service"
)

func main() {
	cfg := &cfg.Config{
		DBUrl:  os.Getenv("DBURL"),
		DBUser: os.Getenv("DBUSER"),
		DBPass: os.Getenv("DBPASSWORD"),
	}

	playerDao := dao.NewPlayerDao(nil)
	playerRepo := repository.NewPlayerRepository(playerDao)
	playerService := service.NewPlayerService(playerRepo)
	playerController := controller.NewPlayerController(playerService)

	handler := handler.NewServer(
		slog.New(&slog.JSONHandler{}),
		cfg,
		controller.NewContainer(playerController),
		nil,
	)

	server := &http.Server{
		Addr:         "127.0.0.1:8080",
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	err := server.ListenAndServe()

	if errors.Is(err, http.ErrServerClosed) {
		fmt.Println("Server closed")
	} else if err != nil {
		fmt.Println("Error starting server:", err)
		os.Exit(1)
	}

}
