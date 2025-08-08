package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	// We will need to import config and other packages later
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	// TODO: Load config from ../../server/internal/config/config.json

	// TODO: Initialize WebSocket client to connect to the hub

	// TODO: Initialize database connection

	// Setup HTTP API endpoints
	mux := http.NewServeMux()
	mux.HandleFunc("/api/nodes", handleGetNodes)

	log.Info().Msg("Manager service starting on :8090")
	if err := http.ListenAndServe(":8090", mux); err != nil {
		log.Fatal().Err(err).Msg("Failed to start manager service")
	}
}

func handleGetNodes(w http.ResponseWriter, r *http.Request) {
	// TODO: Fetch node data from the database or from the hub via WebSocket
	// and return as JSON
	fmt.Fprintln(w, "Node data will be here")
}
