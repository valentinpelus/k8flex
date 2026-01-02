package server

import (
	"fmt"
	"log"
	"net/http"

	"github.com/valentinpelus/k8flex/internal/handler"
	"github.com/valentinpelus/k8flex/internal/middleware"
	"github.com/valentinpelus/k8flex/internal/processor"
)

// Server wraps the HTTP server
type Server struct {
	port           string
	webhookHandler *handler.WebhookHandler
	authMiddleware *middleware.AuthMiddleware
}

// New creates a new HTTP server
func New(port string, authToken string, alertProcessor *processor.AlertProcessor) *Server {
	return &Server{
		port:           port,
		webhookHandler: handler.NewWebhookHandler(alertProcessor),
		authMiddleware: middleware.NewAuthMiddleware(authToken),
	}
}

// SetupRoutes configures HTTP routes
func (s *Server) SetupRoutes() {
	http.HandleFunc("/webhook", s.authMiddleware.Authenticate(s.webhookHandler.HandleWebhook))
	http.HandleFunc("/health", handler.HandleHealth)
}

// Start starts the HTTP server
func (s *Server) Start() error {
	s.SetupRoutes()

	log.Printf("HTTP server listening on :%s", s.port)
	if err := http.ListenAndServe(":"+s.port, nil); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}
