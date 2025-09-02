package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gravadigital/telescopio-api/internal/config"
	"github.com/gravadigital/telescopio-api/internal/logger"
	"github.com/gravadigital/telescopio-api/internal/server"
	"github.com/gravadigital/telescopio-api/internal/storage/postgres"
)

func main() {
	// Inicializar logger
	logger.Init()
	l := logger.Get()

	// Cargar configuración
	cfg, err := config.Load()
	if err != nil {
		l.Fatal("Failed to load configuration", "error", err)
	}

	// Inicializar base de datos
	db, err := postgres.NewDatabase(cfg.Database)
	if err != nil {
		l.Fatal("Failed to connect to database", "error", err)
	}

	// Crear servidor
	srv := server.New(cfg, db)

	// Configurar graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Iniciar servidor en goroutine
	go func() {
		if err := srv.Start(); err != nil {
			l.Error("Server failed to start", "error", err)
		}
	}()

	l.Info("Server started successfully", "port", cfg.Port)

	// Esperar señal de terminación
	<-done
	l.Info("Server is shutting down...")

	// Graceful shutdown con timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Stop(ctx); err != nil {
		l.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	// Cerrar conexión de base de datos
	if err := postgres.Close(); err != nil {
		l.Error("Error closing database connection", "error", err)
	}

	l.Info("Server exited properly")
}
