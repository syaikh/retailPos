package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	_ "github.com/lib/pq"
)

var (
	adjectives = []string{"Premium", "Basic", "Advanced", "Organic", "Natural", "Turbo", "Lite", "Pro", "Smart", "Eco", "Deluxe", "Standard"}
	nouns      = []string{"Coffee", "Bottle", "Notebook", "Keyboard", "Mouse", "Monitor", "Chair", "Desk", "Lamp", "Bag", "Phone", "Cable"}
	models     = []string{"X1", "Pro Max", "v2", "S-Series", "Edition", "Blue", "Red", "Large", "Medium", "Small"}
	methods    = []string{"Cash", "QRIS", "Debit", "Credit"}
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Seeder failed: %v", err)
	}
}

func run() error {
	// Parse flags
	truncateFlag := flag.Bool("truncate", true, "Truncate tables before seeding")
	flag.Parse()

	ctx := context.Background()
	rand.Seed(time.Now().UnixNano())

	// Connect to database
	db, err := sql.Open("postgres", getDSN())
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}
	fmt.Println("✅ Connected to database: retail_pos")

	if *truncateFlag {
		fmt.Println("🗑️  Truncating existing sales and products data...")
		// Disable triggers to avoid FK issues, truncate in correct order, then re-enable
		_, err = db.ExecContext(ctx, `
			TRUNCATE TABLE sale_items, sales, inventory_movements, products 
				RESTART IDENTITY CASCADE;
		`)
		if err != nil {
			return fmt.Errorf("failed to truncate tables: %w", err)
		}
		fmt.Println("✅ Tables truncated successfully.")
	}

	// 1. Ensure Categories exist (minimal setup)
	fmt.Printf("🔧 Checking category data...\n")
	var catCount int
	db.QueryRowContext(ctx, "SELECT COUNT(*) FROM categories").Scan(&catCount)
	if catCount == 0 {
		fmt.Printf("   ⚠️  No categories found. Inserting default categories...\n")
		_, err = db.ExecContext(ctx, `
			INSERT INTO categories (name, description, is_active) VALUES 
			('Electronics', 'Gadgets and devices', true),
			('Groceries', 'Daily needs and food', true),
			('Stationery', 'Office and school supplies', true)
		`)
		if err != nil {
			return fmt.Errorf("failed to insert categories: %w", err)
		}
		catCount = 3
	}
	fmt.Printf("   Found %d categories.\n", catCount)

	// 2. Fetch category IDs
	rows, err := db.QueryContext(ctx, "SELECT id FROM categories")
	if err != nil {
		return err
	}
	var categoryIDs []int
	for rows.Next() {
		var id int
		rows.Scan(&id)
		categoryIDs = append(categoryIDs, id)
	}
	rows.Close()

	// 3. Inject Products
	productCount := 2000
	fmt.Printf("🚀 Injecting %d products...\n", productCount)
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	stmt, err := tx.PrepareContext(ctx, 
		`INSERT INTO products (sku, name, price, cost, stock, category_id, is_active, created_at) 
		 VALUES ($1, $2, $3, $4, $5, $6, true, $7)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for i := 1; i <= productCount; i++ {
		name := fmt.Sprintf("%s %s %s #%d", randElem(adjectives), randElem(nouns), randElem(models), i)
		sku := fmt.Sprintf("SKU-%d-%05d", time.Now().Unix()%1000, i)
		price := (rand.Intn(200) + 10) * 1000 // 10k to 200k
		cost := price - (rand.Intn(50)+10)*100 // Margin 10k-50k
		stock := rand.Intn(1000)
		catID := randElemInt(categoryIDs)
		createdAt := time.Now().AddDate(0, 0, -rand.Intn(90))

		_, err = stmt.ExecContext(ctx, sku, name, price, cost, stock, catID, createdAt)
		if err != nil {
			fmt.Printf("  Warning: failed to insert product %d: %v\n", i, err)
		}

		if i%500 == 0 {
			fmt.Printf("  ...%d products injected\n", i)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit products: %w", err)
	}
	fmt.Printf("✅ %d products injected successfully.\n", productCount)

	// 4. Inject Transactions (Sales)
	transactionCount := 5000
	fmt.Printf("🚀 Injecting %d transactions...\n", transactionCount)

	// Get active user IDs (cashiers)
	userRows, _ := db.QueryContext(ctx, "SELECT id FROM users WHERE is_active = true")
	var userIDs []int
	for userRows.Next() {
		var uid int
		userRows.Scan(&uid)
		userIDs = append(userIDs, uid)
	}
	userRows.Close()
	if len(userIDs) == 0 {
		return fmt.Errorf("no active users found. Please seed users first using 003_seed_data.sql")
	}

	// Get current product IDs and prices
	prodRows, _ := db.QueryContext(ctx, "SELECT id, price FROM products")
	type prodInfo struct{ id, price int }
	var allProds []prodInfo
	for prodRows.Next() {
		var p prodInfo
		prodRows.Scan(&p.id, &p.price)
		allProds = append(allProds, p)
	}
	prodRows.Close()

	tx2, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	saleStmt, _ := tx2.PrepareContext(ctx, 
		`INSERT INTO sales (invoice_number, cashier_id, payment_method, status, subtotal, discount, tax, total_amount, created_at) 
		  VALUES ($1, $2, $3, 'completed', $4, $5, $6, $7, $8) RETURNING id`)
	itemStmt, _ := tx2.PrepareContext(ctx, 
		`INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal) VALUES ($1, $2, $3, $4, $5)`)
	
	for i := 1; i <= transactionCount; i++ {
		invoice := fmt.Sprintf("INV-%d-%06d", time.Now().Year(), i)
		cashierID := randElemInt(userIDs)
		method := randElem(methods)
		numItems := rand.Intn(5) + 1
		createdAt := time.Now().AddDate(0, 0, -rand.Intn(30)).Add(time.Duration(rand.Intn(24)) * time.Hour)

		var totalAmount, totalDiscount, totalTax int
		// Buat item transaksi dulu sementara di memori
		items := make([]struct{pid, qty, price int}, numItems)
		for j := 0; j < numItems; j++ {
			p := allProds[rand.Intn(len(allProds))]
			qty := rand.Intn(5) + 1
			items[j] = struct{pid, qty, price int}{p.id, qty, p.price}
			sub := p.price * qty
			totalAmount += sub
		}
		totalDiscount = int(float64(totalAmount) * 0.05) // Contoh diskon 5%
		totalTax = int(float64(totalAmount) * 0.11)     // Contoh pajak 11%
		grandTotal := totalAmount - totalDiscount + totalTax

		// Insert ke tabel sales
		var saleID int
		err := saleStmt.QueryRowContext(ctx, invoice, cashierID, method, totalAmount, totalDiscount, totalTax, grandTotal, createdAt).Scan(&saleID)
		if err != nil {
			fmt.Printf("  Warning: failed to insert sale %d: %v\n", i, err)
			continue
		}

		// Insert detail item
		for _, it := range items {
			_, err = itemStmt.ExecContext(ctx, saleID, it.pid, it.qty, it.price, it.price*it.qty)
			if err != nil {
				fmt.Printf("  Warning: failed to insert item sale %d: %v\n", saleID, err)
			}
		}

		if i%1000 == 0 {
			fmt.Printf("  ...%d transactions injected\n", i)
		}
	}
	
	if err := tx2.Commit(); err != nil {
		return fmt.Errorf("failed to commit sales: %w", err)
	}
	fmt.Printf("✅ %d transactions injected successfully.\n", transactionCount)
	fmt.Println("\n🎉 Seeding process completed!")
	return nil
}

func getDSN() string {
	host := os.Getenv("DB_HOST")
	if host == "" { host = "localhost" }
	port := os.Getenv("DB_PORT")
	if port == "" { port = "5432" }
	user := os.Getenv("DB_USER")
	if user == "" { user = "pos" }
	password := os.Getenv("DB_PASSWORD")
	if password == "" { password = "admin123" }
	dbname := os.Getenv("DB_NAME")
	if dbname == "" { dbname = "retail_pos" }
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, password, host, port, dbname)
}

func randElem(s []string) string {
	return s[rand.Intn(len(s))]
}

func randElemInt(s []int) int {
	return s[rand.Intn(len(s))]
}
