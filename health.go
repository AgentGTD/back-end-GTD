package encoreapp

import (
	"context"
	"time"
)

type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
	Version   string    `json:"version"`
}

// Health check endpoint - simple endpoint that doesn't require any external dependencies
// encore:api public method=GET path=/api/health
func Health(ctx context.Context) (*HealthResponse, error) {
	return &HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Message:   "Encore app is running successfully",
		Version:   "1.0.0",
	}, nil
}
