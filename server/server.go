package server

import (
	"context"
	"fmt"
	"net/http"
	"pi/server/core"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type Server struct {
	httpServer *http.Server
	bot        *core.Bot
	// Mutex to protect access to the bot instance,
	// as it will be configured and created via a concurrent HTTP request.
	mu sync.Mutex
}

func New() *Server {
	// The bot is not initialized at startup anymore.
	// It will be created when the /configure endpoint is called.
	return &Server{}
}

func (s *Server) Run(port string) error {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	// Define the specific API routes. This creates a route for /api/...
	api := router.Group("/api")
	{
		handlers := NewHandlers(s)
		api.POST("/configure", handlers.Configure)
		api.POST("/start", handlers.Start)
		api.POST("/stop", handlers.Stop)
		api.GET("/status", handlers.Status)
	}

	// --- FIX: Use StaticFile to map the root URL "/" to the index.html file. ---
	// This creates a single, non-conflicting route for the frontend.
	// It does NOT create a greedy wildcard.
	router.StaticFile("/", "./public/index.html")

	s.httpServer = &http.Server{
		Addr:    port,
		Handler: router,
	}

	fmt.Printf("Bot control server listening on port %s\n", port)
	fmt.Println("Serving frontend from './public/index.html' and API from '/api'")

	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown() error {
	if s.httpServer != nil {
		s.mu.Lock()
		defer s.mu.Unlock()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if s.bot != nil {
			s.bot.Stop() // Stop the bot logic if it exists
		}
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}