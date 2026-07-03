package server

import (
	"log"
	"os"
	"strings"

	"github.com/MonuChaudhary14/Archon/internal/auth"
	"github.com/MonuChaudhary14/Archon/internal/cache"
	"github.com/MonuChaudhary14/Archon/internal/database"
	"github.com/MonuChaudhary14/Archon/pkg/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type Server struct {
	router *gin.Engine
}

func NewServer() (*Server, error) {
	db, err := database.NewPostgresPool(os.Getenv("DATABASE_URL"))
	if err != nil {
		return nil, err
	}

	redisClient := cache.NewRedisClient()
	if err := cache.Ping(redisClient); err != nil {
		return nil, err
	}

	log.Println("PostgreSQL Connected")
	log.Println("Redis Connected")

	router := gin.Default()

	allowedOrigins := os.Getenv("FRONTEND_URL")
	if allowedOrigins == "" {
		allowedOrigins = "http://localhost:3000"
	}

	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = strings.Split(allowedOrigins, ",")
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	corsConfig.AllowCredentials = true
	router.Use(cors.New(corsConfig))

	router.Use(middleware.MetricsMiddleware())
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Setup modular routing
	authGroup := router.Group("/")
	auth.Setup(db, redisClient, authGroup)

	// Swagger setup
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return &Server{
		router: router,
	}, nil
}

func (s *Server) Run(addr string) error {
	return s.router.Run(addr)
}
