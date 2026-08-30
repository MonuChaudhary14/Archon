package server

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/MonuChaudhary14/Archon/internal/ai"
	"github.com/MonuChaudhary14/Archon/internal/auth"
	"github.com/MonuChaudhary14/Archon/internal/cache"
	"github.com/MonuChaudhary14/Archon/internal/database"
	"github.com/MonuChaudhary14/Archon/internal/diagram"
	"github.com/MonuChaudhary14/Archon/internal/interview"
	"github.com/MonuChaudhary14/Archon/pkg/config"
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

func NewServer(cfg *config.Config) (*Server, error) {
	db, err := database.NewPostgresPool(cfg.DatabaseURL)
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

	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = cfg.FrontendURLs
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	corsConfig.AllowCredentials = true
	router.Use(cors.New(corsConfig))

	router.Use(middleware.MetricsMiddleware())
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Setup modular routing
	authGroup := router.Group("/")
	userRepo, _ := auth.Setup(db, redisClient, authGroup)

	diagRepo := diagram.NewRepository(db)

	jwtSecret := os.Getenv("JWT_SECRET")

	interviewRepo := interview.NewRepository(db)
	interviewService := interview.NewService(interviewRepo)
	interviewHandler := interview.NewHandler(interviewService)

	verifyOwner := func(ctx context.Context, userID int, interviewID string) (bool, error) {
		_, err := interviewRepo.GetInterviewByID(ctx, userID, interviewID)
		if err != nil {
			return false, err
		}
		return true, nil
	}

	aiGroup := router.Group("/")
	kafkaSvc := ai.Setup(aiGroup, redisClient, diagRepo, jwtSecret, userRepo, verifyOwner)

	outboxWorker := interview.NewOutboxWorker(db, kafkaSvc, 2*time.Second)
	go outboxWorker.Start(context.Background())

	diagramHandler := diagram.NewHandler(diagRepo, verifyOwner)

	interviewGroup := router.Group("/api/v1/interviews")
	interviewGroup.Use(auth.AuthMiddleware(jwtSecret, userRepo))
	interviewGroup.POST("/start", interviewHandler.StartInterview)
	interviewGroup.GET("/questions", interviewHandler.ListQuestions)
	interviewGroup.GET("/:id/report", interviewHandler.GetInterviewReport)
	interviewGroup.POST("/:id/submit", interviewHandler.SubmitInterview)
	interviewGroup.GET("/:id/diagram", diagramHandler.GetDiagram)

	// Swagger setup
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return &Server{
		router: router,
	}, nil
}

func (s *Server) Run(addr string) error {
	return s.router.Run(addr)
}
