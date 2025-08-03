package server

import (
	"net/http"
	"pi/server/core"

	"github.com/gin-gonic/gin"
)

type Handlers struct {
	bot *core.Bot
}

func NewHandlers(bot *core.Bot) *Handlers {
	return &Handlers{bot: bot}
}

func (h *Handlers) Start(c *gin.Context) {
	err := h.bot.StartOperations()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "Bot operations started"})
}

func (h *Handlers) Stop(c *gin.Context) {
	h.bot.StopOperations()
	c.JSON(http.StatusOK, gin.H{"status": "Bot operations stopped"})
}

func (h *Handlers) Status(c *gin.Context) {
	c.JSON(http.StatusOK, h.bot.GetStatus())
}