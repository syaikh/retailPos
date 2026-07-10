package main

import (
	"context"
	"fmt"
	"log"
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
	"retail-pos-system/internal/eventbus"
	"retail-pos-system/internal/inventory"
	"retail-pos-system/internal/middleware"
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
	"retail-pos-system/internal/uom"
	"retail-pos-system/internal/user"
	"retail-pos-system/pkg/websocket"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
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
		ID:      claims.ID,
		Role:    claims.Role,
		StoreID: claims.StoreID,
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
			log.Fatal("FATAL: DB_HOST environment variable is required when DATABASE_URL is not set")
		}
		dsn = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s&timezone=Asia/Jakarta",
			dbUser, dbPass, dbHost, dbPort, dbName, sslmode)
	}
	dbPool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer dbPool.Close()

	if err := dbPool.Ping(context.Background()); err != nil {
		log.Fatalf("Unable to ping database: %v\n", err)
	}
	fmt.Println("Connected to PostgreSQL")

	bus := eventbus.New()
	go bus.Run()
	defer bus.Shutdown()

	userRepo := user.NewRepository(dbPool)
	productRepo := product.NewRepository(dbPool)
	saleRepo := sale.NewRepository(dbPool)
	inventoryRepo := inventory.NewRepository(dbPool)
	bus.Subscribe(inventory.NewStockDeductListener(inventoryRepo))
	customerRepo := customer.NewRepository(dbPool)
	categoryRepo := category.NewRepository(dbPool)
	brandRepo := brand.NewRepository(dbPool)
	uomRepo := uom.NewRepository(dbPool)
	auditRepo := audit.NewRepository(dbPool)
	reportRepo := report.NewRepository(dbPool)

	userSvc := user.NewService(userRepo, bus)
	authSvc := user.NewAuthService(userRepo, bus)
	productSvc := product.NewService(productRepo, categoryRepo, brandRepo, uomRepo, bus)
	saleSvc := sale.NewService(saleRepo, bus)
	saleSvc.SetPriceStore(&productPriceAdapter{repo: productRepo})
	inventorySvc := inventory.NewService(inventoryRepo, bus)
	customerSvc := customer.NewService(customerRepo, bus)
	categorySvc := category.NewService(categoryRepo, bus)
	brandSvc := brand.NewService(brandRepo, bus)
	uomSvc := uom.NewService(uomRepo, bus)
	auditSvc := audit.NewService(auditRepo, bus)
	bus.Subscribe(audit.NewAuditListener(auditSvc))
	reportSvc := report.NewService(reportRepo, bus)

	userH := user.NewHandler(userSvc)
	authH := user.NewAuthHandler(authSvc)
	productH := product.NewHandler(productSvc)
	saleH := sale.NewHandler(saleSvc)
	inventoryH := inventory.NewHandler(inventorySvc)
	customerH := customer.NewHandler(customerSvc)
	categoryH := category.NewHandler(categorySvc)
	brandH := brand.NewHandler(brandSvc)
	uomH := uom.NewHandler(uomSvc)
	auditH := audit.NewHandler(auditSvc)
	reportH := report.NewHandler(reportSvc)

	schemaReg := schema.NewRegistry()
	_ = schemaReg.Register(category.Schema)
	_ = schemaReg.Register(brand.Schema)
	_ = schemaReg.Register(uom.Schema)
	_ = schemaReg.Register(customer.Schema)
	_ = schemaReg.Register(product.Schema)

	adapterReg := importexport.NewAdapterRegistry()
	_ = adapterReg.Register(category.NewAdapter(categoryRepo))
	_ = adapterReg.Register(brand.NewAdapter(brandRepo))
	_ = adapterReg.Register(uom.NewAdapter(uomRepo))
	_ = adapterReg.Register(customer.NewAdapter(customerRepo))
	_ = adapterReg.Register(product.NewAdapter(productRepo, categoryRepo, brandRepo, uomRepo))

	valPipeline := validation.NewDefaultPipeline()
	progStore := progress.NewPgRepository(dbPool)
	progEng := progress.NewEngine(progStore)
	historyStore := history.NewStore(dbPool)
	importEng := importer.NewEngine(schemaReg, valPipeline, adapterReg, progEng, historyStore, bus)
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

	router := gin.Default()
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{cfg.CORSOrigin},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Requested-With", "X-CSRF-Token"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	router.Use(middleware.SecurityHeadersMiddleware([]string{cfg.CORSOrigin}))
	router.Use(middleware.RateLimitMiddleware())

	authMiddleware := middleware.NewModularAuthMiddleware(authSvc)
	permMiddleware := middleware.RequirePermission

	router.GET("/ws", func(c *gin.Context) {
		websocket.ServeWebSocket(hub, c)
	})

	saleH.RegisterPaymentMethodsPublicRoutes(router.Group("/api"))
	productH.RegisterPublicRoutes(router.Group("/api"))
	brandH.RegisterPublicRoutes(router.Group("/api"))
	uomH.RegisterPublicRoutes(router.Group("/api"))

	authH.RegisterLoginRoute(router.Group("/api"), middleware.LoginRateLimitMiddleware())
	authH.RegisterRoutes(router.Group("/api"), authMiddleware, permMiddleware)
	authH.RegisterRefreshRoute(router.Group("/api"), middleware.CSRFMiddleware())
	protected := router.Group("/api")
	protected.Use(authMiddleware)
	protected.Use(middleware.CSRFMiddleware())
	{
		productH.RegisterRoutes(protected, authMiddleware, permMiddleware)
		saleH.RegisterRoutes(protected, authMiddleware, permMiddleware)
		inventoryH.RegisterRoutes(protected, authMiddleware, permMiddleware)
		customerH.RegisterRoutes(protected, authMiddleware, permMiddleware)
		categoryH.RegisterRoutes(protected, authMiddleware, permMiddleware)
		userH.RegisterRoutes(protected, authMiddleware, permMiddleware)
		auditH.RegisterRoutes(protected, authMiddleware, permMiddleware)
		reportH.RegisterRoutes(protected, authMiddleware, permMiddleware)
		brandH.RegisterRoutes(protected, authMiddleware, permMiddleware)
		uomH.RegisterRoutes(protected, authMiddleware, permMiddleware)
		ieH.RegisterRoutes(protected, authMiddleware, permMiddleware)
	}

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "timestamp": time.Now().Format(time.RFC3339)})
	})

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

	dbPool.Close()
	println("Server exited")
}
