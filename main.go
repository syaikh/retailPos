package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"retailPos/internal/auth"
	"retailPos/internal/handler"
	"retailPos/internal/repo"
	"retailPos/internal/service"
	"retailPos/internal/ws"
	"syscall"
	"time"

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
	productGroupRepo := repo.NewProductGroupRepo(db)
	statsRepo := repo.NewStatsRepo(db)
	salesRepo := repo.NewSalesRepo(db)

	// Initialize Services & Hub
	hub := ws.NewHub()
	go hub.Run()

	authService := auth.NewAuthService(userRepo)
	salesService := service.NewSalesService(db, productRepo, hub)

	// Initialize Handlers
	h := handler.NewHandler(authService, productRepo, productGroupRepo, statsRepo, salesRepo, salesService)

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

		// Debug: unprotected chart endpoint
		api.GET("/sales/chart", h.GetSalesChartData)

		// WebSocket
		api.GET("/ws", func(c *gin.Context) {
			hub.ServeHTTP(c.Writer, c.Request)
		})

		// Protected Routes
		protected := api.Group("/")
		protected.Use(handler.AuthMiddleware())
		{
			protected.GET("/product-groups", h.GetProductGroups)
			protected.POST("/product-groups", h.CreateProductGroup)
			protected.PUT("/product-groups/:id", h.UpdateProductGroup)
			protected.DELETE("/product-groups/:id", h.DeleteProductGroup)

			protected.GET("/products", h.GetProducts)
			protected.POST("/products", h.CreateProduct)
			protected.PUT("/products/:id", h.UpdateProduct)
			protected.DELETE("/products/:id", h.DeleteProduct)
			protected.GET("/stats", h.GetDashboardStats)
			protected.GET("/sales", h.GetSalesHistory)
			protected.POST("/sales", h.CreateSale)
		}
	}

	// Resolve build directory relative to the working directory first,
	// then fall back to the executable path for installed binaries.
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("failed to get working directory: %v", err)
	}
	buildDir := filepath.Join(cwd, "web", "build")
	if _, err := os.Stat(buildDir); os.IsNotExist(err) {
		exePath, err := os.Executable()
		if err != nil {
			log.Fatalf("failed to determine executable path: %v", err)
		}
		buildDir = filepath.Join(filepath.Dir(exePath), "web", "build")
	}

	// Serve built frontend static files
	r.Static("/_app", filepath.Join(buildDir, "_app"))
	r.StaticFile("/index.html", filepath.Join(buildDir, "index.html"))

	// Root page and SPA fallback for client-side routing
	r.GET("/", func(c *gin.Context) {
		c.File(filepath.Join(buildDir, "index.html"))
	})
	r.NoRoute(func(c *gin.Context) {
		if c.Request.Method != "GET" {
			c.JSON(404, gin.H{"error": "Not found"})
			return
		}
		c.File(filepath.Join(buildDir, "404.html"))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	go func() {
		log.Printf("Server starting on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}
