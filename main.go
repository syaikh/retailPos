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
	"strings"
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
	roleRepo := repo.NewRoleRepo(db)
	productRepo := repo.NewProductRepo(db)
	productGroupRepo := repo.NewProductGroupRepo(db)
	statsRepo := repo.NewStatsRepo(db)
	salesRepo := repo.NewSalesRepo(db)

	// Initialize Services & Hub
	hub := ws.NewHub()
	go hub.Run()

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}
	refreshSecret := os.Getenv("JWT_REFRESH_SECRET")
	if refreshSecret == "" {
		log.Fatal("JWT_REFRESH_SECRET environment variable is required")
	}
	authRepo := auth.NewPostgresRepo(db)
	tokenService := auth.NewTokenService(secret, refreshSecret)
	authService := auth.NewAuthService(userRepo, authRepo, tokenService)
	salesService := service.NewSalesService(db, productRepo, userRepo, hub)
	inventoryService := service.NewInventoryService(productRepo)

	// Initialize Handlers
	h := handler.NewHandler(authService, userRepo, roleRepo, productRepo, productGroupRepo, statsRepo, salesRepo, salesService, inventoryService)

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
		api.POST("/logout", h.Logout)
		api.POST("/refresh", h.RefreshToken)

		// Debug: unprotected chart endpoint
		api.GET("/sales/chart", h.GetSalesChartData)

		// WebSocket
		api.GET("/ws", func(c *gin.Context) {
			hub.ServeHTTP(c.Writer, c.Request)
		})

		// Protected Routes
		protected := api.Group("/")
		protected.Use(auth.AuthMiddleware(tokenService, roleRepo, userRepo))
		{
			// Auth: get current user with permissions
			protected.GET("/auth/validate", h.ValidateSession)

			// Admin-only routes
			adminRoutes := protected.Group("/")
			adminRoutes.Use(auth.RoleMiddleware("admin"))
			{
				adminRoutes.POST("/product-groups", h.CreateProductGroup)
				adminRoutes.PUT("/product-groups/:id", h.UpdateProductGroup)
				adminRoutes.DELETE("/product-groups/:id", h.DeleteProductGroup)
				adminRoutes.POST("/products", h.CreateProduct)
				adminRoutes.PUT("/products/:id", h.UpdateProduct)
				adminRoutes.DELETE("/products/:id", h.DeleteProduct)
				adminRoutes.GET("/inventory/export", h.ExportInventory)

				// Admin management endpoints
				adminRoutes.GET("/permissions", h.ListPermissions)
				adminRoutes.GET("/roles", h.ListRoles)
				adminRoutes.POST("/roles", h.CreateRole)
				adminRoutes.PUT("/roles/:id/permissions", h.UpdateRolePermissions)
				adminRoutes.DELETE("/roles/:id", h.DeleteRole)
				adminRoutes.GET("/users", h.ListUsers)
				adminRoutes.PUT("/users/:id/role", h.UpdateUserRole)
			}

			// Protected routes (admin + cashier)
			protected.GET("/product-groups", h.GetProductGroups)
			protected.GET("/products", h.GetProducts)
			protected.GET("/products/:id", h.GetProductByID)
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

	// Root page
	r.GET("/", func(c *gin.Context) {
		c.File(filepath.Join(buildDir, "index.html"))
	})

	// SPA fallback: serve index.html for client-side routes
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		// API routes that didn't match (except /api prefix which is handled by api group)
		if strings.HasPrefix(path, "/api") {
			c.JSON(404, gin.H{"error": "Not found"})
			return
		}

		// Static asset requests (contain a file extension) that missed -> 404
		if strings.Contains(path, ".") && c.Request.Method == "GET" {
			c.JSON(404, gin.H{"error": "Not found"})
			return
		}

		// All other GET requests are SPA routes -> serve index.html
		if c.Request.Method == "GET" {
			c.File(filepath.Join(buildDir, "index.html"))
			return
		}

		// Non-GET to non-API paths -> 404
		c.JSON(404, gin.H{"error": "Not found"})
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
