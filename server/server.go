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

	// Setup routes for the new bot controller
	handlers := NewHandlers(s)
	router.POST("/configure", handlers.Configure)
	router.POST("/start", handlers.Start)
	router.POST("/stop", handlers.Stop)
	router.GET("/status", handlers.Status)

	s.httpServer = &http.Server{
		Addr:    port,
		Handler: router,
	}

	fmt.Printf("Bot control server listening on port %s\n", port)
	fmt.Println("Endpoints: POST /configure, POST /start, POST /stop, GET /status")

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