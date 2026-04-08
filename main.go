package main

import (
	"log"
	"net/http"
	"os"
	"retailPos/internal/auth"
	"retailPos/internal/handler"
	"retailPos/internal/repo"
	"retailPos/internal/service"
	"retailPos/internal/ws"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	db, err := repo.NewDB()
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize Repositories
	userRepo := repo.NewUserRepo(db)
	productRepo := repo.NewProductRepo(db)

	// Initialize Services & Hub
	hub := ws.NewHub()
	go hub.Run()

	authService := auth.NewAuthService(userRepo)
	salesService := service.NewSalesService(db, productRepo, hub)

	// Initialize Handlers
	h := handler.NewHandler(authService, productRepo, salesService)

	r := gin.Default()

	// CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// Routes
	api := r.Group("/api")
	{
		api.POST("/login", h.Login)

		// WebSocket
		api.GET("/ws", func(c *gin.Context) {
			hub.ServeHTTP(c.Writer, c.Request)
		})

		// Protected Routes
		protected := api.Group("/")
		protected.Use(handler.AuthMiddleware())
		{
			protected.GET("/products", h.GetProducts)
			protected.POST("/products", h.CreateProduct)
			protected.POST("/sales", h.CreateSale)
		}
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
