package encoreapp

import (
	"context"
)

type TestResponse struct {
	Message string `json:"message"`
	Success bool   `json:"success"`
}

// Simple test endpoint - no external dependencies
// encore:api public method=GET path=/test
func Test(ctx context.Context) (*TestResponse, error) {
	return &TestResponse{
		Message: "Test endpoint working!",
		Success: true,
	}, nil
}
