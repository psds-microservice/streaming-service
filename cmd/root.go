package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "streaming-service",
	Short: "Streaming service: session lifecycle, WebSocket stream relay",
	Long:  `HTTP + WebSocket API. Commands: api, migrate, seed.`,
	RunE:  runAPI, // default: run API (same as "streaming-service api")
}

func init() {
	rootCmd.AddCommand(apiCmd)
	rootCmd.AddCommand(migrateCmd)
	rootCmd.AddCommand(seedCmd)
}

// Execute runs the root command and returns the error (for main to log.Fatal).
func Execute() error {
	return rootCmd.Execute()
}
