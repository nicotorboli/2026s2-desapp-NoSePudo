package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/controller"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/repository"
	"github.com/nicotorboli/2026s2-desapp-NoSePudo/backend/internal/service"
)

// setupRoutes wires the layered architecture components and returns an http.Handler.
func setupRoutes() http.Handler {
	// Layered architecture instantiation (Principle III)
	healthRepo := repository.NewHealthRepository()
	healthSvc := service.NewHealthService(healthRepo)
	healthCtrl := controller.NewHealthController(healthSvc)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthCtrl.HandleHealth)
	return mux
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      setupRoutes(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("Server listening on port %s\n", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server listen error: %v\n", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v\n", err)
	}
	log.Println("Server gracefully stopped")
}
