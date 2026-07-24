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

	"retail-pos-system/internal/audit"
	"retail-pos-system/internal/brand"
	"retail-pos-system/internal/category"
	"retail-pos-system/internal/config"
	"retail-pos-system/internal/customer"
	"retail-pos-system/internal/customergroup"
	"retail-pos-system/internal/eventbus"
	"retail-pos-system/internal/inventory"
	"retail-pos-system/internal/middleware"
	"retail-pos-system/internal/pricing"
	"retail-pos-system/internal/platform/importexport"
	"retail-pos-system/internal/platform/importexport/export"
	ieh "retail-pos-system/internal/platform/importexport/handler"
	"retail-pos-system/internal/platform/importexport/history"
	importer "retail-pos-system/internal/platform/importexport/import"
	"retail-pos-system/internal/platform/importexport/progress"
	"retail-pos-system/internal/platform/importexport/schema"
	"retail-pos-system/internal/platform/importexport/template"
	"retail-pos-system/internal/platform/importexport/validation"
	"retail-pos-system/internal/product"
	"retail-pos-system/internal/report"
	"retail-pos-system/internal/sale"
	"retail-pos-system/internal/shared"
	"retail-pos-system/internal/shift"
	"retail-pos-system/internal/store"
	"retail-pos-system/internal/supplier"
	"retail-pos-system/internal/uom"
	"retail-pos-system/internal/user"
	"retail-pos-system/pkg/cache"
	"retail-pos-system/pkg/websocket"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"retail-pos-system/docs"
)

type authAdapter struct {
	svc *user.AuthService
}

func (a *authAdapter) ValidateToken(tokenString string) (*websocket.Claims, error) {
	claims, err := a.svc.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}
	return &websocket.Claims{
		ID:       claims.ID,
		Role:     claims.Role,
		StoreID:  claims.StoreID,
		Username: claims.Username,
	}, nil
}

type productLookupAdapter struct {
	repo *product.Repository
}

func (a *productLookupAdapter) GetProductByID(ctx context.Context, id int) (string, string, int, *int, error) {
	p, err := a.repo.GetProductByID(ctx, id, nil)
	if err != nil {
		return "", "", 0, nil, err
	}
	return p.SKU, p.Name, p.Stock, p.StoreID, nil
}

type productPriceAdapter struct {
	repo *product.Repository
}

func (a *productPriceAdapter) GetProductPrice(ctx context.Context, productID int) (int, error) {
	return a.repo.GetProductPrice(ctx, productID)
}

const (
	defaultMaxConns         = 25
	defaultMinConns         = 5
	defaultMaxConnLifetime  = 30 * time.Minute
	defaultMaxConnIdleTime  = 5 * time.Minute
	defaultHealthCheckPeriod = 15 * time.Second
	defaultBodyLimit         = 1 << 20
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

	appCache := cache.New(10*time.Minute, 30*time.Second)

	bus := eventbus.New()
	bus.SetDeadLetterStore(eventbus.NewPgDeadLetterStore(dbPool))
	go bus.Run()
	defer bus.Shutdown()

	userRepo := user.NewRepository(dbPool)
	userRepo.SetCache(appCache)
	productRepo := product.NewRepository(dbPool)
	productRepo.SetCache(appCache)
	saleRepo := sale.NewRepository(dbPool)
	inventoryRepo := inventory.NewRepository(dbPool)
	customerRepo := customer.NewRepository(dbPool)
	categoryRepo := category.NewRepository(dbPool)
	categoryRepo.SetCache(appCache)
	brandRepo := brand.NewRepository(dbPool)
	brandRepo.SetCache(appCache)
	uomRepo := uom.NewRepository(dbPool)
	uomRepo.SetCache(appCache)
	auditRepo := audit.NewRepository(dbPool)
	reportRepo := report.NewRepository(dbPool)
	reportRepo.SetCache(appCache)
	pricingRepo := pricing.NewRepository(dbPool)
	supplierRepo := supplier.NewRepository(dbPool)
	customerGroupRepo := customergroup.NewRepository(dbPool)
	storeRepo := store.NewRepository(dbPool)
	shiftRepo := shift.NewRepository(dbPool)

	userSvc := user.NewService(userRepo)
	authSvc := user.NewAuthService(userRepo)
	productSvc := product.NewService(productRepo, categoryRepo, brandRepo, uomRepo, bus)
	saleSvc := sale.NewService(saleRepo, bus)
	saleSvc.SetPriceStore(&productPriceAdapter{repo: productRepo})
	pricingResolver := pricing.NewResolver(pricingRepo)
	saleSvc.SetPriceResolver(pricingResolver)
	inventorySvc := inventory.NewService(inventoryRepo, bus)
	customerSvc := customer.NewService(customerRepo)
	categorySvc := category.NewService(categoryRepo)
	brandSvc := brand.NewService(brandRepo)
	uomSvc := uom.NewService(uomRepo)
	auditSvc := audit.NewService(auditRepo)
	reportSvc := report.NewService(reportRepo, bus)
	pricingSvc := pricing.NewService(pricingRepo)
	supplierSvc := supplier.NewService(supplierRepo)
	customerGroupSvc := customergroup.NewService(customerGroupRepo)
	storeSvc := store.NewService(storeRepo)
	shiftSvc := shift.NewService(shiftRepo)

	userH := user.NewHandler(userSvc, auditSvc)
	authH := user.NewAuthHandler(authSvc, auditSvc)
	productH := product.NewHandler(productSvc, auditSvc)
	saleH := sale.NewHandler(saleSvc, auditSvc)
	inventoryH := inventory.NewHandler(inventorySvc, auditSvc)
	customerH := customer.NewHandler(customerSvc, auditSvc)
	categoryH := category.NewHandler(categorySvc, auditSvc)
	brandH := brand.NewHandler(brandSvc, auditSvc)
	uomH := uom.NewHandler(uomSvc, auditSvc)
	auditH := audit.NewHandler(auditSvc)
	reportH := report.NewHandler(reportSvc)
	pricingH := pricing.NewHandler(pricingSvc, pricingResolver, auditSvc)
	pricingH.SetProductSearcher(pricingRepo)
	supplierH := supplier.NewHandler(supplierSvc, auditSvc)
	customerGroupH := customergroup.NewHandler(customerGroupSvc, auditSvc)
	storeH := store.NewHandler(storeSvc, auditSvc)
	shiftH := shift.NewHandler(shiftSvc, auditSvc)

	schemaReg := schema.NewRegistry()
	_ = schemaReg.Register(category.Schema)
	_ = schemaReg.Register(brand.Schema)
	_ = schemaReg.Register(uom.Schema)
	_ = schemaReg.Register(customer.Schema)
	_ = schemaReg.Register(product.Schema)
	_ = schemaReg.Register(store.Schema)
	_ = schemaReg.Register(customergroup.Schema)
	_ = schemaReg.Register(pricing.Schema)
	_ = schemaReg.Register(supplier.Schema)

	adapterReg := importexport.NewAdapterRegistry()
	_ = adapterReg.Register(category.NewAdapter(categoryRepo))
	_ = adapterReg.Register(brand.NewAdapter(brandRepo))
	_ = adapterReg.Register(uom.NewAdapter(uomRepo))
	_ = adapterReg.Register(customer.NewAdapter(customerRepo))
	_ = adapterReg.Register(product.NewAdapter(productRepo, categoryRepo, brandRepo, uomRepo))
	_ = adapterReg.Register(store.NewAdapter(storeRepo))
	_ = adapterReg.Register(customergroup.NewAdapter(customerGroupRepo))
	_ = adapterReg.Register(pricing.NewAdapter(pricingRepo))
	_ = adapterReg.Register(supplier.NewAdapter(supplierRepo))

	valPipeline := validation.NewDefaultPipeline()
	progStore := progress.NewPgRepository(dbPool)
	progEng := progress.NewEngine(progStore)
	historyStore := history.NewStore(dbPool)
	importEng := importer.NewEngine(schemaReg, valPipeline, adapterReg, progEng, historyStore)
	exportEng := export.NewEngine()
	templateEng := template.NewEngine()
	ieH := ieh.NewHandler(schemaReg, adapterReg, importEng, exportEng, templateEng, progEng, historyStore)

	hub := websocket.NewHub(&authAdapter{authSvc})
	go hub.Run()
	defer hub.Shutdown()

	wsProductLookup := &productLookupAdapter{repo: productRepo}
	bus.Subscribe(websocket.NewSaleCreatedListener(hub))
	bus.Subscribe(websocket.NewProductUpdatedListener(hub))
	bus.Subscribe(websocket.NewStockAdjustedListener(hub, wsProductLookup))
	bus.Subscribe(reportRepo.NewSaleCreatedListener())

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

	authMiddleware := middleware.NewModularAuthMiddleware(authSvc)
	permMiddleware := middleware.RequirePermission

	router.GET("/ws", middleware.WebSocketRateLimitMiddleware(), func(c *gin.Context) {
		websocket.ServeWebSocket(hub, c)
	})

	saleH.RegisterPaymentMethodsPublicRoutes(router.Group("/api"))
	productH.RegisterPublicRoutes(router.Group("/api"))
	brandH.RegisterPublicRoutes(router.Group("/api"))
	uomH.RegisterPublicRoutes(router.Group("/api"))

	authH.RegisterLoginRoute(router.Group("/api"), middleware.LoginRateLimitMiddleware())
	authH.RegisterRoutes(router.Group("/api"), authMiddleware, middleware.CSRFMiddleware(), permMiddleware)
	authH.RegisterRefreshRoute(router.Group("/api"), middleware.RefreshRateLimitMiddleware())
	authH.RegisterChangePasswordRoute(router.Group("/api"), authMiddleware, middleware.CSRFMiddleware())
	protected := router.Group("/api")
	protected.Use(authMiddleware)
	protected.Use(middleware.CSRFMiddleware())
	noopAuth := func(c *gin.Context) { c.Next() }
	{
		productH.RegisterRoutes(protected, noopAuth, permMiddleware)
		saleH.RegisterRoutes(protected, noopAuth, permMiddleware)
		inventoryH.RegisterRoutes(protected, noopAuth, permMiddleware)
		customerH.RegisterRoutes(protected, noopAuth, permMiddleware)
		categoryH.RegisterRoutes(protected, noopAuth, permMiddleware)
		customerGroupH.RegisterRoutes(protected, noopAuth, permMiddleware)
		storeH.RegisterRoutes(protected, noopAuth, permMiddleware)
		userH.RegisterRoutes(protected, noopAuth, permMiddleware)
		auditH.RegisterRoutes(protected, noopAuth, permMiddleware)
		reportH.RegisterRoutes(protected, noopAuth, permMiddleware)
		brandH.RegisterRoutes(protected, noopAuth, permMiddleware)
		uomH.RegisterRoutes(protected, noopAuth, permMiddleware)
		ieH.RegisterRoutes(protected, noopAuth, permMiddleware)
		pricingH.RegisterRoutes(protected, noopAuth, permMiddleware)
		supplierH.RegisterRoutes(protected, noopAuth, permMiddleware)
		shiftH.RegisterRoutes(protected, noopAuth, permMiddleware)
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
