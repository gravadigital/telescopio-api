package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/gravadigital/telescopio-api/internal/config"
	"github.com/gravadigital/telescopio-api/internal/logger"
	"github.com/gravadigital/telescopio-api/internal/storage/migrations"
	"github.com/gravadigital/telescopio-api/internal/storage/postgres"
)

func main() {
	cfg := config.Load()

	logger.Initialize("info")
	log := logger.Migration()

	rollback := flag.Bool("rollback", false, "Rollback the last migration")
	flag.Parse()

	log.Info("Starting migration process", "rollback", *rollback)

	db, err := postgres.Connect(cfg)
	if err != nil {
		log.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}

	if *rollback {
		log.Info("Rolling back migrations...")
		if err := migrations.RollbackMigration(db); err != nil {
			log.Error("Migration rollback failed", "error", err)
			os.Exit(1)
		}
		log.Info("Migration rollback completed successfully")
	} else {
		log.Info("Running migrations...")
		if err := migrations.RunMigrations(db); err != nil {
			log.Error("Migration failed", "error", err)
			os.Exit(1)
		}
		log.Info("Migrations completed successfully")
	}

	fmt.Println("Migration process completed!")
}
