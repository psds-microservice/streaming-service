package cmd

import (
	"fmt"
	"log"

	"github.com/psds-microservice/streaming-service/internal/database"
	"github.com/spf13/cobra"
)

var commandCmd = &cobra.Command{
	Use:   "command [name]",
	Short: "Run one-time command (migrate, migrate-create)",
	RunE:  runCommand,
}

func init() {
	rootCmd.AddCommand(commandCmd)
}

func runCommand(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		fmt.Println("available: migrate, migrate-create")
		return nil
	}
	name := args[0]
	switch name {
	case "migrate":
		return runMigrateUp(cmd, nil)
	case "migrate-create":
		migrationName := ""
		if len(args) > 1 {
			migrationName = args[1]
		} else {
			fmt.Print("Enter migration name: ")
			_, _ = fmt.Scanln(&migrationName)
		}
		if migrationName == "" {
			log.Fatal("migration name required")
		}
		return database.CreateMigration(migrationName)
	default:
		return fmt.Errorf("unknown command: %s", name)
	}
}
