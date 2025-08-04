package server

import (
	"fmt"
	"net/http"
	"pi/wallet"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type Server struct {
	wallet *wallet.Wallet
}

func New() *Server {
	return &Server{
		wallet: wallet.New(),
	}
}

func (s *Server) Run(port string) error {
	gin.SetMode(gin.ReleaseMode)

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// API routes
	r.POST("/api/login", s.Login)
	r.GET("/ws/withdraw", s.Withdraw)
	
	// Serve static files from dist directory (built React app)
	r.StaticFS("/assets", http.Dir("./dist/assets"))
	r.Static("/static", "./dist")
	
	// Serve index.html for all non-API routes (SPA routing)
	r.NoRoute(func(ctx *gin.Context) {
		ctx.File("./dist/index.html")
	})

	fmt.Printf("running on port: %s\n", port)

	return r.Run(port)
}