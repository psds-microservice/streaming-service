package cmd

import (
	"fmt"

	"github.com/joho/godotenv"
	"github.com/psds-microservice/streaming-service/internal/config"
	"github.com/psds-microservice/streaming-service/internal/database"
	"github.com/spf13/cobra"
)

var seedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Run migrations and seeds (migrate up, then database/seeds/*.sql)",
	RunE:  runSeed,
}

func runSeed(cmd *cobra.Command, args []string) error {
	_ = godotenv.Load(".env")
	_ = godotenv.Load("../.env")
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("config: %w", err)
	}
	if err := database.MigrateUp(cfg.DatabaseURL()); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	db, err := database.Open(cfg.DSN())
	if err != nil {
		return fmt.Errorf("database: %w", err)
	}
	if err := database.RunSeeds(db); err != nil {
		return fmt.Errorf("seed: %w", err)
	}
	return nil
}
