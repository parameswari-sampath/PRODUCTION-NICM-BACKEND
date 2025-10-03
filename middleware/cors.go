package middleware

import (
	"net/http"

	"github.com/rs/cors"
)

// SetupCORS returns CORS middleware that allows all origins
func SetupCORS() func(http.Handler) http.Handler {
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: false,
		MaxAge:           300, // 5 minutes preflight cache
	})

	return c.Handler
}
