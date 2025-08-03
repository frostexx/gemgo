package server

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"pi/server/core"

	"github.com/gin-gonic/gin"
)

// Handlers now holds a reference to the main Server struct
// to access the shared bot instance and the mutex.
type Handlers struct {
	s *Server
}

func NewHandlers(s *Server) *Handlers {
	return &Handlers{s: s}
}

// ConfigureRequest defines the JSON structure your frontend should send.
type ConfigureRequest struct {
	MainWalletSeed  string `json:"main_wallet_seed" binding:"required"`
	SponsorSeed     string `json:"sponsor_wallet_seed" binding:"required"`
	UnlockTimestamp string `json:"unlock_timestamp" binding:"required"`
}

// Configure is the new handler to initialize the bot.
func (h *Handlers) Configure(c *gin.Context) {
	h.s.mu.Lock()
	defer h.s.mu.Unlock()

	var req ConfigureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	if h.s.bot != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Bot is already configured. Stop it before re-configuring."})
		return
	}

	botConfig := core.BotConfig{
		NetworkURL:      os.Getenv("NET_URL"),
		NetworkPass:     os.Getenv("NET_PASSPHRASE"),
		MainWalletSeed:  req.MainWalletSeed,
		SponsorSeed:     req.SponsorSeed,
		UnlockTimestamp: req.UnlockTimestamp,
	}

	bot, err := core.NewBot(botConfig)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to initialize bot: %v", err)})
		return
	}

	h.s.bot = bot
	go h.s.bot.Run() // Start the bot's background listener.

	log.Println("Bot has been configured successfully via API.")
	c.JSON(http.StatusOK, gin.H{"status": "Bot configured successfully."})
}

func (h *Handlers) Start(c *gin.Context) {
	h.s.mu.Lock()
	defer h.s.mu.Unlock()

	if h.s.bot == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bot is not configured. Please call /configure first."})
		return
	}

	err := h.s.bot.StartOperations()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "Bot operations started"})
}

func (h *Handlers) Stop(c *gin.Context) {
	h.s.mu.Lock()
	defer h.s.mu.Unlock()

	if h.s.bot == nil {
		c.JSON(http.StatusOK, gin.H{"status": "Bot was not running."})
		return
	}

	h.s.bot.StopOperations()
	h.s.bot = nil // Allow for re-configuration
	c.JSON(http.StatusOK, gin.H{"status": "Bot operations stopped and de-configured."})
}

func (h *Handlers) Status(c *gin.Context) {
	h.s.mu.Lock()
	defer h.s.mu.Unlock()

	if h.s.bot == nil {
		c.JSON(http.StatusOK, gin.H{"status": "Bot is not configured."})
		return
	}

	c.JSON(http.StatusOK, h.s.bot.GetStatus())
}