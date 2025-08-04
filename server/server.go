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

	// First, serve static files from the "public" directory.
	// This will handle the root path "/" and serve index.html.
	router.Static("/", "./public")

	// --- FIX: Group all API endpoints under the "/api" prefix ---
	// This resolves the routing conflict.
	api := router.Group("/api")
	{
		handlers := NewHandlers(s)
		api.POST("/configure", handlers.Configure)
		api.POST("/start", handlers.Start)
		api.POST("/stop", handlers.Stop)
		api.GET("/status", handlers.Status)
	}

	s.httpServer = &http.Server{
		Addr:    port,
		Handler: router,
	}

	fmt.Printf("Bot control server listening on port %s\n", port)
	fmt.Println("Serving frontend from './public' and API from '/api'")

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