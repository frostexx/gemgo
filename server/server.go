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
		AllowOrigins:     []string{"*"}, // or restrict to specific domains
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.POST("/api/login", s.Login)
	r.GET("/ws/withdraw", s.Withdraw)
	
	// Updated to serve from src directory instead of public
	r.GET("/", func(ctx *gin.Context) {
		ctx.File("./src/index.html")
	})
	
	// Updated to serve static assets from src directory
	r.StaticFS("/assets", http.Dir("./src/assets"))
	
	// If you need to serve other static files from src, add this:
	r.Static("/static", "./src")

	fmt.Printf("running on port: %s\n", port)

	return r.Run(port)
}