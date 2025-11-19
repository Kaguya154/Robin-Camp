package api

import (
	"os"

	"Robin-Camp/internal"
	"Robin-Camp/internal/boxoffice"

	"github.com/cloudwego/hertz/pkg/route"
)

func RegisterRoutes(h *route.RouterGroup) {
	// Build box office client from environment.
	boxClient, _ := boxoffice.NewFromEnv()

	// Read auth token from environment (for BearerAuth on POST /movies).
	authToken := os.Getenv("AUTH_TOKEN")

	handler := internal.NewHandler(internal.DB, boxClient, authToken)
	handler.RegisterRoutes(h)
}
