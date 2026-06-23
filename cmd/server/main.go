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
	"retail-pos-system/internal/service"
	"retail-pos-system/pkg/websocket"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		loc, _ = time.LoadLocation("UTC")
	}
	time.Local = loc

	cfg := config.Load()

	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://pos:admin123@localhost:5433/retail_pos?sslmode=disable&timezone=Asia/Jakarta"
	}
	dbPool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		panic(fmt.Sprintf("Unable to connect to database: %v\n", err))
	}

	if err := dbPool.Ping(context.Background()); err != nil {
		panic(fmt.Sprintf("Unable to ping database: %v\n", err))
	}
	fmt.Println(" Connected to PostgreSQL")

	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{cfg.CORSOrigin},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	authRepo := repository.NewPostgresRepository(dbPool)
	roleRepo := repository.NewPostgresRepository(dbPool)
	productRepo := repository.NewPostgresRepository(dbPool)
	paymentRepo := repository.NewPostgresRepository(dbPool)
	saleRepo := repository.NewPostgresRepository(dbPool)
	customerRepo := repository.NewPostgresRepository(dbPool)
	auditRepo := repository.NewPostgresRepository(dbPool)
	categoryRepo := repository.NewPostgresRepository(dbPool)

	authService := auth.NewAuthService(authRepo, dbPool)
	hub := websocket.NewHub(authService)
	go hub.Run()

	excelService := service.NewExcelService(saleRepo)

	h := handler.NewHandler(authRepo, roleRepo, productRepo, paymentRepo, saleRepo, customerRepo, authService, hub, auditRepo, categoryRepo, excelService)

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
		public.GET("/payment-methods", h.ListPaymentMethods)
		public.GET("/payment-methods/:code", h.GetPaymentMethodByCode)
	}

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
		protected.GET("/product-stock/:id", h.GetProductStockByID)
		protected.POST("/products", middleware.RequirePermission("product:create"), h.CreateProduct)
		protected.PUT("/products/:id", middleware.RequirePermission("product:update"), h.UpdateProduct)
		protected.DELETE("/products/:id", middleware.RequirePermission("product:delete"), h.DeleteProduct)
		protected.POST("/products/bulk/status", middleware.RequirePermission("product:update"), h.BulkUpdateProductStatus)

		protected.POST("/brands", middleware.RequirePermission("product:create"), h.CreateBrand)
		protected.PUT("/brands/:id", middleware.RequirePermission("product:update"), h.UpdateBrand)
		protected.DELETE("/brands/:id", middleware.RequirePermission("product:delete"), h.DeleteBrand)

		protected.POST("/units-of-measure", middleware.RequirePermission("product:create"), h.CreateUnitOfMeasure)
		protected.PUT("/units-of-measure/:id", middleware.RequirePermission("product:update"), h.UpdateUnitOfMeasure)
		protected.DELETE("/units-of-measure/:id", middleware.RequirePermission("product:delete"), h.DeleteUnitOfMeasure)

		protected.POST("/sales", middleware.RequirePermission("sale:create"), h.CreateSale)
		protected.GET("/sales", middleware.RequirePermission("sale:read"), h.GetSalesHistory)
		protected.GET("/sales/export", middleware.RequirePermission("report:read"), h.ExportSales)
		protected.GET("/sales/:id", middleware.RequirePermission("sale:read"), h.GetSaleByID)

		protected.GET("/dashboard/stats", middleware.RequirePermission("dashboard:read"), h.GetDashboardStats)
		protected.GET("/dashboard/live", middleware.RequirePermission("dashboard:read"), h.GetLiveDashboardStats)
		protected.GET("/dashboard/chart", middleware.RequirePermission("report:read"), h.GetSalesChartData)
		protected.GET("/dashboard/chart/weekly", middleware.RequirePermission("report:read"), h.GetSalesWeeklyReport)
		protected.GET("/dashboard/chart/monthly", middleware.RequirePermission("report:read"), h.GetSalesMonthlyReport)
		protected.GET("/dashboard/comparison", middleware.RequirePermission("report:read"), h.GetPeriodComparison)
		protected.POST("/dashboard/export", middleware.RequirePermission("report:read"), h.ExportDashboard)

		protected.GET("/admin/users", middleware.RequirePermission("user:read"), h.ListUsers)
		protected.POST("/admin/users", middleware.RequirePermission("user:create"), h.CreateUser)
		protected.PUT("/admin/users/:id", middleware.RequirePermission("user:update"), h.UpdateUser)
		protected.DELETE("/admin/users/:id", middleware.RequirePermission("user:delete"), h.DeleteUser)

		protected.GET("/admin/roles", middleware.RequirePermission("role:read"), h.ListRoles)
		protected.POST("/admin/roles", middleware.RequirePermission("role:create"), h.CreateRole)
		protected.PUT("/admin/roles/:id", middleware.RequirePermission("role:update"), h.UpdateRole)
		protected.PUT("/admin/roles/:id/permissions", middleware.RequirePermission("role:update"), h.UpdateRolePermissions)
		protected.DELETE("/admin/roles/:id", middleware.RequirePermission("role:delete"), h.DeleteRole)
		protected.GET("/admin/permissions", middleware.RequirePermission("role:read"), h.ListPermissions)

		protected.GET("/categories/manage", middleware.RequirePermission("category:read"), h.ListCategoriesManagement)
		protected.POST("/categories", middleware.RequirePermission("category:create"), h.CreateCategoryHandler)
		protected.PUT("/categories/:id", middleware.RequirePermission("category:update"), h.UpdateCategoryHandler)
		protected.DELETE("/categories/:id", middleware.RequirePermission("category:delete"), h.DeleteCategoryHandler)

		protected.POST("/inventory/adjust", middleware.RequirePermission("inventory:adjust"), h.AdjustStock)

	protected.GET("/customers", middleware.RequirePermission("customer:read"), h.GetCustomers)
	protected.GET("/customers/:id", middleware.RequirePermission("customer:read"), h.GetCustomerByID)
	protected.POST("/customers", middleware.RequirePermission("customer:create"), h.CreateCustomer)
	protected.PUT("/customers/:id", middleware.RequirePermission("customer:update"), h.UpdateCustomer)
	protected.DELETE("/customers/:id", middleware.RequirePermission("customer:delete"), h.DeleteCustomer)
	protected.POST("/customers/bulk/status", middleware.RequirePermission("customer:update"), h.BulkUpdateCustomerStatus)
	protected.POST("/customers/bulk/delete", middleware.RequirePermission("customer:delete"), h.BulkDeleteCustomers)

		protected.GET("/audit-logs", middleware.RequirePermission("audit:read"), h.ListAuditLogs)
		protected.GET("/audit-logs/export", middleware.RequirePermission("audit:read"), h.ExportAuditLogs)
	}

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "timestamp": time.Now().Format(time.RFC3339)})
	})

	router.GET("/ws", h.ServeWS)

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

	go func() {
		println("Server starting on " + addr + " (env: " + cfg.Env + ")")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		fmt.Printf("Server forced to shutdown: %v\n", err)
	}

	hub.Shutdown()
	dbPool.Close()
	println("Server exited")
}
