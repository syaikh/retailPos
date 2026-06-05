package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"retail-pos-system/internal/auth"
	"retail-pos-system/internal/config"
	"retail-pos-system/internal/delivery/http/handler"
	"retail-pos-system/internal/middleware"
	"retail-pos-system/internal/repository"
	"retail-pos-system/pkg/websocket"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// Set timezone to Asia/Jakarta
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		loc, _ = time.LoadLocation("UTC")
	}
	time.Local = loc

	cfg := config.Load()

	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Database Connection
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		// Default local DB -- sesuaikan dengan docker compose jika berbeda
		dsn = "postgres://pos:admin123@localhost:5433/retail_pos?sslmode=disable&timezone=Asia/Jakarta"
	}
	dbPool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		panic(fmt.Sprintf("Unable to connect to database: %v\n", err))
	}

	// Verify DB connection
	if err := dbPool.Ping(context.Background()); err != nil {
		panic(fmt.Sprintf("Unable to ping database: %v\n", err))
	}
	fmt.Println("✅ Connected to PostgreSQL")

	router := gin.Default()

	// CORS
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{cfg.CORSOrigin},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

// Real Repositories
 	authRepo := repository.NewPostgresRepository(dbPool)
 	roleRepo := repository.NewPostgresRepository(dbPool)
 	productRepo := repository.NewPostgresRepository(dbPool)
 	saleRepo := repository.NewPostgresRepository(dbPool)
 	auditRepo := repository.NewPostgresRepository(dbPool)
 	categoryRepo := repository.NewPostgresRepository(dbPool)

 	// Auth service with real DB pool
 	authService := auth.NewAuthService(authRepo, dbPool)

 	// WebSocket hub
 	hub := websocket.NewHub(authService)
 	go hub.Run()

 	h := handler.NewHandler(authRepo, roleRepo, productRepo, saleRepo, authService, hub, auditRepo, categoryRepo)

	// Public routes
	public := router.Group("/api")
	public.Use(middleware.RateLimitMiddleware())
	{
		public.POST("/login", h.Login)
		public.POST("/refresh", h.RefreshToken)
		public.GET("/categories", h.ListCategories)
		public.GET("/products", h.GetProducts)
		public.GET("/stock-thresholds", h.GetStockThresholds)
		public.GET("/products/next-sku", h.GetNextSKU)
		public.GET("/brands", h.GetBrands)
		public.GET("/tax-classes", h.GetTaxClasses)
		public.GET("/units-of-measure", h.GetUnitsOfMeasure)
		public.GET("/warehouses", h.GetWarehouses)
		public.GET("/dashboard/years", h.GetAvailableYears)
	}

	// Protected routes (require authentication)
	protected := router.Group("/api")
	protected.Use(func(c *gin.Context) {
		c.Set("authService", authService)
		c.Next()
	})
	protected.Use(middleware.AuthMiddleware())
	{
		protected.POST("/validate", h.ValidateSession)
		protected.POST("/logout", h.Logout)
		protected.GET("/products/:id", h.GetProductByID)
		protected.POST("/products", h.CreateProduct)
		protected.PUT("/products/:id", h.UpdateProduct)
		protected.DELETE("/products/:id", h.DeleteProduct)

		protected.POST("/brands", h.CreateBrand)
		protected.PUT("/brands/:id", h.UpdateBrand)
		protected.DELETE("/brands/:id", h.DeleteBrand)

		protected.POST("/sales", h.CreateSale)
		protected.GET("/sales", h.GetSalesHistory)
		protected.GET("/sales/:id", h.GetSaleByID)
		protected.GET("/dashboard/stats", h.GetDashboardStats)
		protected.GET("/dashboard/chart", h.GetSalesChartData)
		protected.GET("/dashboard/chart/weekly", h.GetSalesWeeklyReport)
		protected.GET("/dashboard/chart/monthly", h.GetSalesMonthlyReport)
		protected.GET("/dashboard/comparison", h.GetPeriodComparison)

		// Admin
		protected.GET("/admin/users", h.ListUsers)
		protected.POST("/admin/users", h.CreateUser)
		protected.PUT("/admin/users/:id", h.UpdateUser)
		protected.DELETE("/admin/users/:id", h.DeleteUser)

		protected.GET("/admin/roles", h.ListRoles)
		protected.POST("/admin/roles", h.CreateRole)
		protected.PUT("/admin/roles/:id/permissions", h.UpdateRolePermissions)
		protected.DELETE("/admin/roles/:id", h.DeleteRole)
		protected.GET("/admin/permissions", h.ListPermissions)

		// Category Management
		protected.GET("/categories/manage", h.ListCategoriesManagement)
		protected.POST("/categories", h.CreateCategoryHandler)
		protected.PUT("/categories/:id", h.UpdateCategoryHandler)
		protected.DELETE("/categories/:id", h.DeleteCategoryHandler)

		protected.GET("/audit-logs", h.ListAuditLogs)
	}

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "timestamp": time.Now().Format(time.RFC3339)})
	})

	// WebSocket
	router.GET("/ws", h.ServeWS)

	// Server
	port := os.Getenv("PORT")
	if port == "" {
		port = "9095"
	}
	addr := ":" + port

	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		println("Server starting on " + addr + " (env: " + cfg.Env + ")")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		fmt.Printf("Server forced to shutdown: %v\n", err)
	}

	// Close WebSocket connections
	hub.Shutdown()

	// Close database connection
	dbPool.Close()

	println("Server exited")
}
