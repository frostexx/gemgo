package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"pi/server/core"
	"time"

	"github.com/gin-gonic/gin"
)

type Server struct {
	httpServer *http.Server
	bot        *core.Bot
}

func New() *Server {
	// Initialize the bot with configuration from environment variables
	botConfig := core.BotConfig{
		NetworkURL:      os.Getenv("NET_URL"),
		NetworkPass:     os.Getenv("NET_PASSPHRASE"),
		MainWalletSeed:  os.Getenv("MAIN_WALLET_SEED"),
		SponsorSeed:     os.Getenv("SPONSOR_WALLET_SEED"),
		UnlockTimestamp: os.Getenv("UNLOCK_TIMESTAMP"),
	}

	bot, err := core.NewBot(botConfig)
	if err != nil {
		log.Fatalf("Failed to initialize bot: %v", err)
	}

	return &Server{
		bot: bot,
	}
}

func (s *Server) Run(port string) error {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	// Setup routes
	handlers := NewHandlers(s.bot)
	router.POST("/start", handlers.Start)
	router.POST("/stop", handlers.Stop)
	router.GET("/status", handlers.Status)

	s.httpServer = &http.Server{
		Addr:    port,
		Handler: router,
	}

	fmt.Printf("Bot control server listening on port %s\n", port)
	fmt.Println("Endpoints: POST /start, POST /stop, GET /status")

	// Start the bot's background processes
	go s.bot.Run()

	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown() error {
	if s.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.bot.Stop() // Stop the bot logic
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}