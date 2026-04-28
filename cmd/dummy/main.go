package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"retail-pos-system/internal/domain"
	"retail-pos-system/internal/repository"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	// Seed random number generator
	rand.Seed(time.Now().UnixNano())

	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// DB connection pool with retry
	dbPool, err := repository.NewDBConnection()
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer dbPool.Close()

	log.Println("✅ Connected to PostgreSQL")

	// Debug: Check current database
	var dbName string
	err = dbPool.QueryRow(context.Background(), "SELECT current_database()").Scan(&dbName)
	if err != nil {
		log.Printf("Failed to get database name: %v", err)
	} else {
		log.Printf("Connected to database: %s", dbName)
	}

	// Generate comprehensive dummy data
	if err := generateDummyData(dbPool, userRepo); err != nil {
		log.Fatalf("Failed to generate dummy data: %v", err)
	}

	log.Println("🎉 Dummy data generation completed successfully!")
	log.Println("📊 Database now contains realistic test data for development and testing")
}

func generateDummyData(dbPool *pgxpool.Pool, userRepo repository.UserRepository) error {
	ctx := context.Background()

	log.Println("🏪 Generating stores...")
	if err := generateStores(ctx, dbPool); err != nil {
		return fmt.Errorf("failed to generate stores: %w", err)
	}

	log.Println("👥 Generating users...")
	if err := generateUsers(ctx, dbPool, userRepo); err != nil {
		return fmt.Errorf("failed to generate users: %w", err)
	}

	log.Println("📦 Generating products...")
	if err := generateProducts(ctx, dbPool); err != nil {
		return fmt.Errorf("failed to generate products: %w", err)
	}

	log.Println("🛒 Generating sales transactions...")
	if err := generateSales(ctx, dbPool); err != nil {
		return fmt.Errorf("failed to generate sales: %w", err)
	}

	return nil
}

func generateStores(ctx context.Context, db *pgxpool.Pool) error {
	stores := []struct {
		id      int
		name    string
		address string
		phone   string
	}{
		{1, "Toko Utama Jakarta", "Jl. Sudirman No. 123, Jakarta Pusat", "021-1234567"},
		{2, "Cabang Bandung", "Jl. Asia Afrika No. 45, Bandung", "022-7654321"},
		{3, "Cabang Surabaya", "Jl. Tunjungan No. 78, Surabaya", "031-9876543"},
		{4, "Mini Market Bogor", "Jl. Pajajaran No. 12, Bogor", "0251-112233"},
		{5, "Kiosk Depok", "Jl. Margonda No. 34, Depok", "021-445566"},
	}

	for _, store := range stores {
		result, err := db.Exec(ctx, `
			INSERT INTO stores (id, name, address, phone, is_active, created_at)
			VALUES ($1, $2, $3, $4, true, NOW())
			ON CONFLICT (id) DO NOTHING`,
			store.id, store.name, store.address, store.phone)
		if err != nil {
			log.Printf("Error inserting store %s: %v", store.name, err)
			return err
		}
		rowsAffected := result.RowsAffected()
		log.Printf("Inserted store %s, rows affected: %d", store.name, rowsAffected)
	}

	// Verify stores were inserted
	var count int
	err := db.QueryRow(ctx, "SELECT COUNT(*) FROM stores").Scan(&count)
	if err != nil {
		return err
	}
	log.Printf("✅ Created %d stores (verified: %d in DB)", len(stores), count)
	return nil
}

func generateUsers(ctx context.Context, db *pgxpool.Pool, userRepo repository.UserRepository) error {
	// Password hash for 'admin123'
	passwordHash := "$2a$10$cV3/py5RCcZT.B8Hh1HTeO.07eOjtQXYqm6G.KQ/YOTHfKJouRznq"

	users := []struct {
		username string
		email    string
		roleID   int
		storeID  *int
	}{
		// Store managers
		{"manager_jkt", "manager.jkt@retailpos.local", 3, intPtr(1)},
		{"manager_bdg", "manager.bdg@retailpos.local", 3, intPtr(2)},
		{"manager_sby", "manager.sby@retailpos.local", 3, intPtr(3)},

		// Cashiers for each store
		{"cashier_jkt1", "cashier1.jkt@retailpos.local", 4, intPtr(1)},
		{"cashier_jkt2", "cashier2.jkt@retailpos.local", 4, intPtr(1)},
		{"cashier_bdg1", "cashier1.bdg@retailpos.local", 4, intPtr(2)},
		{"cashier_bdg2", "cashier2.bdg@retailpos.local", 4, intPtr(2)},
		{"cashier_sby1", "cashier1.sby@retailpos.local", 4, intPtr(3)},
		{"cashier_bgr1", "cashier1.bgr@retailpos.local", 4, intPtr(4)},
		{"cashier_dpk1", "cashier1.dpk@retailpos.local", 4, intPtr(5)},
	}

	for _, userData := range users {
		// Check if user exists
		var existingID int
		err := db.QueryRow(ctx, "SELECT id FROM users WHERE username = $1", userData.username).Scan(&existingID)
		if err == nil {
			continue // User already exists
		}

		// Insert user directly to avoid scanning issues
		_, err = db.Exec(ctx, `
			INSERT INTO users (username, email, password_hash, role_id, store_id, is_active, created_at)
			VALUES ($1, $2, $3, $4, $5, true, NOW())`,
			userData.username, userData.email, passwordHash, userData.roleID, userData.storeID)

		if err != nil {
			log.Printf("Warning: Failed to create user %s: %v", userData.username, err)
		}
	}

	log.Printf("✅ Created %d additional users", len(users))
	return nil
}

func generateProducts(ctx context.Context, db *pgxpool.Pool) error {
	// Get existing categories
	rows, err := db.Query(ctx, "SELECT id, name FROM categories ORDER BY id")
	if err != nil {
		return err
	}
	defer rows.Close()

	var categories []struct {
		id   int
		name string
	}
	for rows.Next() {
		var cat struct {
			id   int
			name string
		}
		if err := rows.Scan(&cat.id, &cat.name); err != nil {
			return err
		}
		categories = append(categories, cat)
	}

	// Product templates by category
	productTemplates := map[string][]struct {
		name  string
		price int
		cost  int
		stock int
	}{
		"Makanan Instant": {
			{"Indomie Goreng Original", 5000, 4000, 100},
			{"Indomie Kari Ayam", 5000, 4000, 120},
			{"Indomie Soto", 5000, 4000, 95},
			{"Mie Sedap Goreng", 4500, 3600, 110},
			{"Supermi Ayam Bawang", 6000, 4800, 85},
		},
		"Minuman": {
			{"Teh Botol Sosro", 5000, 4500, 200},
			{"Teh Pucuk Harum", 5500, 4950, 180},
			{"Pocari Sweat", 6500, 5850, 150},
			{"Kopiko 78°C", 8000, 6500, 120},
			{"Good Day Freeze", 7000, 6300, 130},
		},
		"Snack": {
			{"Chitato BBQ", 10000, 8500, 75},
			{"Chitato Lite", 9500, 8075, 80},
			{"Lays Rumput Laut", 8500, 7225, 90},
			{"Doritos Nacho", 12000, 10200, 65},
			{"Beng-Beng", 2500, 2000, 300},
		},
	}

	// Generate products for each category
	productID := 6 // Start after existing products
	for _, category := range categories {
		templates, exists := productTemplates[category.name]
		if !exists {
			continue
		}

		for _, template := range templates {
			sku := fmt.Sprintf("SKU-%03d", productID)

			_, err := db.Exec(ctx, `
				INSERT INTO products (id, sku, name, category_id, price, cost, stock, stock_min, stock_max, is_active, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, true, NOW(), NOW())
				ON CONFLICT (id) DO NOTHING`,
				productID, sku, template.name, category.id, template.price, template.cost,
				template.stock, template.stock/10, template.stock*2)

			if err != nil {
				return err
			}
			productID++
		}
	}

	totalProducts := productID - 1
	log.Printf("✅ Created %d additional products across %d categories", totalProducts-5, len(categories))
	return nil
}

func generateSales(ctx context.Context, db *pgxpool.Pool) error {
	// Get all products and users
	products, err := getAllProducts(ctx, db)
	if err != nil {
		return err
	}

	cashiers, err := getCashiers(ctx, db)
	if err != nil {
		return err
	}

	if len(products) == 0 || len(cashiers) == 0 {
		return fmt.Errorf("no products or cashiers found")
	}

	// Generate sales for the past month
	now := time.Now()
	startDate := now.AddDate(0, -1, 0)

	saleID := 1
	totalSales := 0

	for current := startDate; current.Before(now); current = current.AddDate(0, 0, 1) {
		// Generate 3-8 sales per day
		dailySales := 3 + rand.Intn(6)

		for i := 0; i < dailySales; i++ {
			if err := generateSingleSale(ctx, db, saleID, products, cashiers, current); err != nil {
				return err
			}
			saleID++
			totalSales++
		}
	}

	log.Printf("✅ Generated %d sales transactions over 1 month", totalSales)
	return nil
}

func generateSingleSale(ctx context.Context, db *pgxpool.Pool, saleID int, products []domain.Product, cashiers []domain.User, saleDate time.Time) error {
	// Random cashier
	cashier := cashiers[rand.Intn(len(cashiers))]

	// Random number of items (1-5)
	itemCount := 1 + rand.Intn(5)

	// Select random products
	selectedProducts := make([]domain.Product, 0, itemCount)
	totalAmount := 0

	for i := 0; i < itemCount; i++ {
		product := products[rand.Intn(len(products))]
		quantity := 1 + rand.Intn(3) // 1-3 items per product

		// Avoid duplicate products in same sale
		duplicate := false
		for _, p := range selectedProducts {
			if p.ID == product.ID {
				duplicate = true
				break
			}
		}
		if duplicate {
			i--
			continue
		}

		selectedProducts = append(selectedProducts, product)
		totalAmount += product.Price * quantity
	}

	// Random payment method
	paymentMethods := []string{"cash", "card", "qris"}
	paymentMethod := paymentMethods[rand.Intn(len(paymentMethods))]

	// Create sale record
	_, err := db.Exec(ctx, `
		INSERT INTO sales (id, invoice_number, total_amount, payment_method, cashier_id, store_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO NOTHING`,
		saleID, fmt.Sprintf("INV-%06d", saleID), totalAmount, paymentMethod, cashier.ID, cashier.StoreID, saleDate)

	if err != nil {
		return err
	}

	// Create sale items
	for _, product := range selectedProducts {
		quantity := 1 + rand.Intn(3)
		unitPrice := product.Price

		_, err := db.Exec(ctx, `
			INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT DO NOTHING`,
			saleID, product.ID, quantity, unitPrice, unitPrice*quantity)

		if err != nil {
			return err
		}
	}

	return nil
}

// Helper functions

func intPtr(i int) *int {
	return &i
}

func getAllProducts(ctx context.Context, db *pgxpool.Pool) ([]domain.Product, error) {
	rows, err := db.Query(ctx, "SELECT id, name, price FROM products WHERE is_active = true ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []domain.Product
	for rows.Next() {
		var product domain.Product
		if err := rows.Scan(&product.ID, &product.Name, &product.Price); err != nil {
			return nil, err
		}
		products = append(products, product)
	}

	return products, nil
}

func getCashiers(ctx context.Context, db *pgxpool.Pool) ([]domain.User, error) {
	rows, err := db.Query(ctx, "SELECT id, store_id FROM users WHERE role_id = 4 AND is_active = true ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cashiers []domain.User
	for rows.Next() {
		var cashier domain.User
		if err := rows.Scan(&cashier.ID, &cashier.StoreID); err != nil {
			return nil, err
		}
		cashiers = append(cashiers, cashier)
	}

	return cashiers, nil
}