package main

import (
	"log"

	"github.com/valentinpelus/k8flex/internal/app"
	"github.com/valentinpelus/k8flex/internal/server"
)

func main() {
	// Initialize application
	application, err := app.New()
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}

	// Log startup information
	application.LogStartupInfo()

	// Create and start HTTP server
	srv := server.New(application.Config.Port, application.Config.WebhookAuthToken, application.AlertProcessor)
	if err := srv.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
