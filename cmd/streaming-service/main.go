// Package main — точка входа streaming-service (HTTP + WebSocket).
package main

import (
	"log"

	"github.com/psds-microservice/streaming-service/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
