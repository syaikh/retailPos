package handler_test

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"retailPos/internal/auth"
	"retailPos/internal/handler"
	model "retailPos/internal/model"
	"retailPos/internal/repo"
	"retailPos/internal/service"
	"retailPos/internal/ws"
)

var (
	h           *handler.Handler
	productRepo *repo.ProductRepo
	roleRepo    *repo.RoleRepo
)

// Setup func untuk inisialisasi Gin router & Dependencies
func setupServer() *gin.Engine {
	// Pindah working directory ke root project untuk memuat .env
	cwd, _ := os.Getwd()
	os.Chdir(filepath.Join(cwd, "../.."))
	if err := godotenv.Load(); err != nil {
		log.Println("No .env find in test")
	}

	db, err := repo.NewDB()
	if err != nil {
		log.Fatalf("failed to connect to database for test: %v", err)
	}

	userRepo := repo.NewUserRepo(db)
	roleRepo = repo.NewRoleRepo(db)
	productRepo = repo.NewProductRepo(db)
	productGroupRepo := repo.NewProductGroupRepo(db)
	statsRepo := repo.NewStatsRepo(db)
	salesRepo := repo.NewSalesRepo(db)
	hub := ws.NewHub()
	go hub.Run()

	authRepo := auth.NewPostgresRepo(db)
	tokenService := auth.NewTokenService("test-secret", "test-refresh-secret")
	authService := auth.NewAuthService(userRepo, authRepo, tokenService)
	salesService := service.NewSalesService(db, productRepo, userRepo, hub)
	inventoryService := service.NewInventoryService(productRepo)
	h = handler.NewHandler(authService, userRepo, roleRepo, productRepo, productGroupRepo, statsRepo, salesRepo, salesService, inventoryService)

	gin.SetMode(gin.TestMode)
	r := gin.Default()

	// Mocking middleware auth sehingga setiap request direkam sebagai role 'admin'
	r.Use(func(c *gin.Context) {
		c.Set("role", "admin")
		c.Set("user_id", 1) // Mock cashier ID
		// No store_id set - admin can see all
		c.Next()
	})

	r.POST("/api/products", h.CreateProduct)
	r.DELETE("/api/products/:id", h.DeleteProduct)
	r.POST("/api/sales", h.CreateSale)

	return r
}

func TestProductSoftDeleteAndRestoreFlow(t *testing.T) {
	r := setupServer()

	// 1. Create a Product
	skuTest := "TEST-RESTORE-123"
	pPayload := map[string]any{
		"name":  "Test Produk Alpha",
		"sku":   skuTest,
		"price": 50000,
		"stock": 0, // Dibuat 0 agar bisa di-delete langsung (validasi delete)
	}
	body, _ := json.Marshal(pPayload)

	req1, _ := http.NewRequest("POST", "/api/products", bytes.NewBuffer(body))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)

	if w1.Code != http.StatusCreated {
		t.Fatalf("Failed creating product: %v", w1.Body.String())
	}

	var createdProduct model.Product
	json.Unmarshal(w1.Body.Bytes(), &createdProduct)

	// 2. Delete the Product
	req2, _ := http.NewRequest("DELETE", "/api/products/"+strconv.Itoa(createdProduct.ID), nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("Failed deleting product: %v", w2.Body.String())
	}

	// Memastikan Delete berhasil di basis data
	deletedP, _ := productRepo.GetBySKUWithDeleted(skuTest, nil)
	if deletedP.DeletedAt == nil {
		t.Fatalf("Expected DeletedAt to be populated, got nil")
	}

	// 3. Rekreasi Product -> Harus ter-trigger RESTORE
	pPayloadRestore := map[string]any{
		"name":  "Test Produk Alpha (Restored)", // Nama diubah
		"sku":   skuTest,
		"price": 100000, // Harga diubah
		"stock": 10,     // Stok diperbarui
	}
	bodyRestore, _ := json.Marshal(pPayloadRestore)
	req3, _ := http.NewRequest("POST", "/api/products", bytes.NewBuffer(bodyRestore))
	req3.Header.Set("Content-Type", "application/json")
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)

	if w3.Code != http.StatusCreated {
		t.Fatalf("Failed restoring product: %v", w3.Body.String())
	}

	restoredP, _ := productRepo.GetByID(createdProduct.ID, nil)
	if restoredP.Name != "Test Produk Alpha (Restored)" {
		t.Fatalf("Expected product name to be restored & updated, got %s", restoredP.Name)
	}
	if restoredP.RestoredAt == nil {
		t.Fatalf("Expected RestoredAt to be not nil")
	}

	// Clean up -> Force Delete Test Data
	cleanupDB(restoredP.ID)
}

func TestDeleteProductWithStock(t *testing.T) {
	r := setupServer()

	// 1. Create a Product dengan stok > 0
	skuTest := "TEST-DEL-REJECT-999"
	pPayload := map[string]any{
		"name":  "Produk Stok Penuh",
		"sku":   skuTest,
		"price": 10000,
		"stock": 15,
	}
	body, _ := json.Marshal(pPayload)
	req, _ := http.NewRequest("POST", "/api/products", bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var p model.Product
	json.Unmarshal(w.Body.Bytes(), &p)

	// 2. Coba Hapus
	reqDel, _ := http.NewRequest("DELETE", "/api/products/"+strconv.Itoa(p.ID), nil)
	wDel := httptest.NewRecorder()
	r.ServeHTTP(wDel, reqDel)

	if wDel.Code != http.StatusFailedDependency {
		t.Fatalf("Expected status Failed Dependency (424), got %d: %s", wDel.Code, wDel.Body.String())
	}

	// Clean up
	cleanupDB(p.ID)
}

func TestSnapshotTransaction(t *testing.T) {
	r := setupServer()

	// 1. Create Product
	skuTest := "TEST-SNAP-777"
	pPayload := map[string]any{
		"name":  "Snapshot Item",
		"sku":   skuTest,
		"price": 15000,
		"stock": 50,
	}
	body, _ := json.Marshal(pPayload)
	req1, _ := http.NewRequest("POST", "/api/products", bytes.NewBuffer(body))
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	var p model.Product
	json.Unmarshal(w1.Body.Bytes(), &p)

	// 2. Lakukan transaksi Sales
	salePayload := model.Sale{
		TotalAmount:   30000,
		PaymentMethod: "cash",
		Items: []model.SaleItem{
			{
				ProductID:   p.ID,
				Quantity:    2,
				PriceAtSale: 15000,
			},
		},
	}
	saleBody, _ := json.Marshal(salePayload)
	reqSale, _ := http.NewRequest("POST", "/api/sales", bytes.NewBuffer(saleBody))
	wSale := httptest.NewRecorder()
	r.ServeHTTP(wSale, reqSale)

	if wSale.Code != http.StatusCreated {
		t.Fatalf("Failed creating sale: %s", wSale.Body.String())
	}

	// Verifikasi nama snapshot secara langsung ke database
	db, _ := repo.NewDB()
	var snapshotName string
	db.QueryRow("SELECT product_name FROM sale_items WHERE sale_id = (SELECT id FROM sales ORDER BY id DESC LIMIT 1)").Scan(&snapshotName)

	if snapshotName != "Snapshot Item" {
		t.Fatalf("Expected snapshot name 'Snapshot Item', got '%s'", snapshotName)
	}

	cleanupDB(p.ID)
}

func cleanupDB(productID int) {
	db, _ := repo.NewDB()
	// Membersihkan record agar tidak mengganggu test lain
	db.Exec("DELETE FROM sale_items WHERE product_id = $1", productID)
	// Kita force delete sepenuhnya untuk tes
	db.Exec("DELETE FROM products WHERE id = $1", productID)
}
