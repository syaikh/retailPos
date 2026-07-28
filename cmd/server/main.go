package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"retail-pos-system/internal/config"
	"retail-pos-system/internal/middleware"
	"retail-pos-system/internal/shared"
	"retail-pos-system/internal/wiring"
	"retail-pos-system/pkg/websocket"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"retail-pos-system/docs"
)

const (
	defaultMaxConns         = 25
	defaultMinConns         = 5
	defaultMaxConnLifetime  = 30 * time.Minute
	defaultMaxConnIdleTime  = 5 * time.Minute
	defaultHealthCheckPeriod = 15 * time.Second
	defaultBodyLimit         = 32 << 20
	defaultPort              = "9095"
	defaultReadTimeout       = 15 * time.Second
	defaultWriteTimeout      = 120 * time.Second
	defaultIdleTimeout       = 60 * time.Second
)

func main() {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		loc, _ = time.LoadLocation("UTC")
	}
	time.Local = loc

	cfg := config.Load()
	shared.InitLogger(cfg.Env, cfg.LogLevel)

	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		sslmode := "disable"
		if cfg.Env == "production" {
			sslmode = "require"
		}
		dbUser := os.Getenv("DB_USER")
		dbPass := os.Getenv("DB_PASSWORD")
		dbHost := os.Getenv("DB_HOST")
		dbPort := os.Getenv("DB_PORT")
		dbName := os.Getenv("DB_NAME")
		if dbHost == "" {
			slog.Error("DB_HOST environment variable is required when DATABASE_URL is not set")
			os.Exit(1)
		}
		dsn = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s&timezone=Asia/Jakarta",
			dbUser, dbPass, dbHost, dbPort, dbName, sslmode)
	}
	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		slog.Error("unable to parse database config", "error", err)
		os.Exit(1)
	}
	poolCfg.MaxConns = defaultMaxConns
	poolCfg.MinConns = defaultMinConns
	poolCfg.MaxConnLifetime = defaultMaxConnLifetime
	poolCfg.MaxConnIdleTime = defaultMaxConnIdleTime
	poolCfg.HealthCheckPeriod = defaultHealthCheckPeriod

	dbPool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		slog.Error("unable to connect to database", "error", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	if err := dbPool.Ping(context.Background()); err != nil {
		slog.Error("unable to ping database", "error", err)
		os.Exit(1)
	}
	slog.Info("connected to PostgreSQL")

	deps := wiring.Initialize(wiring.Providers{DB: dbPool, Config: cfg})

	go deps.Bus.Run()
	defer deps.Bus.Shutdown()

	go deps.Hub.Run()
	defer deps.Hub.Shutdown()

	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			if _, err := dbPool.Exec(context.Background(), "SELECT refresh_sales_mv()"); err != nil {
				slog.Error("failed to refresh materialized views", "error", err)
			} else {
				slog.Debug("materialized views refreshed")
			}
		}
	}()

	router := gin.Default()
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{cfg.CORSOrigin},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Requested-With", "X-CSRF-Token"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	router.Use(gzip.Gzip(gzip.DefaultCompression))
	router.Use(middleware.SecurityHeadersMiddleware([]string{cfg.CORSOrigin}))
	router.Use(middleware.RateLimitMiddleware())
	router.Use(middleware.BodyLimitMiddleware(defaultBodyLimit))

	authMiddleware := middleware.NewModularAuthMiddleware(deps.AuthSvc)
	permMiddleware := middleware.RequirePermission

	router.GET("/ws", middleware.WebSocketRateLimitMiddleware(), func(c *gin.Context) {
		websocket.ServeWebSocket(deps.Hub, c)
	})

	deps.SaleH.RegisterPaymentMethodsPublicRoutes(router.Group("/api"))
	deps.ProductH.RegisterPublicRoutes(router.Group("/api"))
	deps.BrandH.RegisterPublicRoutes(router.Group("/api"))
	deps.UOMH.RegisterPublicRoutes(router.Group("/api"))

	deps.AuthH.RegisterLoginRoute(router.Group("/api"), middleware.LoginRateLimitMiddleware())
	deps.AuthH.RegisterRoutes(router.Group("/api"), authMiddleware, middleware.CSRFMiddleware(), permMiddleware)
	deps.AuthH.RegisterRefreshRoute(router.Group("/api"), middleware.RefreshRateLimitMiddleware())
	deps.AuthH.RegisterChangePasswordRoute(router.Group("/api"), authMiddleware, middleware.CSRFMiddleware())
	protected := router.Group("/api")
	protected.Use(authMiddleware)
	protected.Use(middleware.CSRFMiddleware())
	noopAuth := func(c *gin.Context) { c.Next() }
	{
		deps.ProductH.RegisterRoutes(protected, noopAuth, permMiddleware)
		deps.PurchaseH.RegisterRoutes(protected, noopAuth, permMiddleware)
		deps.SaleH.RegisterRoutes(protected, noopAuth, permMiddleware)
		deps.InventoryH.RegisterRoutes(protected, noopAuth, permMiddleware)
		deps.CustomerH.RegisterRoutes(protected, noopAuth, permMiddleware)
		deps.CategoryH.RegisterRoutes(protected, noopAuth, permMiddleware)
		deps.CustomerGroupH.RegisterRoutes(protected, noopAuth, permMiddleware)
		deps.StoreH.RegisterRoutes(protected, noopAuth, permMiddleware)
		deps.UserH.RegisterRoutes(protected, noopAuth, permMiddleware)
		deps.AuditH.RegisterRoutes(protected, noopAuth, permMiddleware)
		deps.ReportH.RegisterRoutes(protected, noopAuth, permMiddleware)
		deps.BrandH.RegisterRoutes(protected, noopAuth, permMiddleware)
		deps.UOMH.RegisterRoutes(protected, noopAuth, permMiddleware)
		deps.IEH.RegisterRoutes(protected, noopAuth, permMiddleware)
		deps.PricingH.RegisterRoutes(protected, noopAuth, permMiddleware)
		deps.SupplierH.RegisterRoutes(protected, noopAuth, permMiddleware)
		deps.ShiftH.RegisterRoutes(protected, noopAuth, permMiddleware)
	}

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "timestamp": time.Now().Format(time.RFC3339)})
	})

	docs.SwaggerInfo.BasePath = "/api"
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}
	addr := ":" + port

	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  defaultReadTimeout,
		WriteTimeout: defaultWriteTimeout,
		IdleTimeout:  defaultIdleTimeout,
	}

	go func() {
		slog.Info("server starting", "addr", addr, "env", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	}

	dbPool.Close()
	slog.Info("server exited")
}
