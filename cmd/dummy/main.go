package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	_ "github.com/lib/pq"
)

var (
	adjectives = []string{"Premium", "Basic", "Advanced", "Organic", "Natural", "Turbo", "Lite", "Pro", "Smart", "Eco"}
	nouns      = []string{"Coffee", "Bottle", "Notebook", "Keyboard", "Mouse", "Monitor", "Chair", "Desk", "Lamp", "Bag"}
	models     = []string{"X1", "Pro Max", "v2", "S-Series", "Edition", "Blue", "Red", "Large", "Medium", "Small"}
	methods    = []string{"Cash", "QRIS", "Debit", "Credit"}
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Dummy seeder failed: %v", err)
	}
}

func run() error {
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

	fmt.Println("Connected to database. Starting dummy data injection...")

	// 1. Get Prerequisites
	categoryIDs := getIDs(ctx, db, "categories")
	if len(categoryIDs) == 0 {
		return fmt.Errorf("no categories found. Please run migrations/seeds first")
	}
	userIDs := getIDs(ctx, db, "users")
	if len(userIDs) == 0 {
		return fmt.Errorf("no users found. Please run migrations/seeds first")
	}

	// 2. Inject Products (2000+)
	fmt.Printf("Injecting 2000 products...\n")
	productIDs := []int{}
	for i := 0; i < 2000; i++ {
		name := fmt.Sprintf("%s %s %s #%d", randElem(adjectives), randElem(nouns), randElem(models), i)
		sku := fmt.Sprintf("SKU-%05d-%d-%d", i, rand.Intn(1000), time.Now().Unix()%10000)
		price := (rand.Intn(100) + 5) * 1000 // 5k to 100k
		cost := price / 2
		stock := rand.Intn(500)
		catID := randElemInt(categoryIDs)

		var id int
		err := db.QueryRowContext(ctx, 
			"INSERT INTO products (sku, name, price, cost, stock, category_id, is_active, created_at) VALUES ($1, $2, $3, $4, $5, $6, true, $7) RETURNING id",
			sku, name, price, cost, stock, catID, time.Now().AddDate(0, 0, -rand.Intn(30)),
		).Scan(&id)
		
		if err != nil {
			fmt.Printf("Warning: failed to insert product %d: %v\n", i, err)
			continue
		}
		productIDs = append(productIDs, id)
		if i%500 == 0 {
			fmt.Printf("  ...%d products injected\n", i)
		}
	}

	// 3. Inject Transactions (5000+)
	fmt.Printf("Injecting 5000 transactions...\n")
	for i := 0; i < 5000; i++ {
		invoice := fmt.Sprintf("INV-%d-%06d-%d", time.Now().Year(), i, time.Now().Unix()%1000)
		cashierID := randElemInt(userIDs)
		method := randElem(methods)
		createdAt := time.Now().AddDate(0, 0, -rand.Intn(30)).Add(time.Duration(rand.Intn(24)) * time.Hour)

		// Create sale
		var saleID int
		err := db.QueryRowContext(ctx,
			"INSERT INTO sales (invoice_number, cashier_id, payment_method, status, created_at) VALUES ($1, $2, $3, 'completed', $4) RETURNING id",
			invoice, cashierID, method, createdAt,
		).Scan(&saleID)

		if err != nil {
			fmt.Printf("Warning: failed to insert sale %d: %v\n", i, err)
			continue
		}

		// Add 1-5 items
		numItems := rand.Intn(5) + 1
		totalAmount := 0
		for j := 0; j < numItems; j++ {
			pID := randElemInt(productIDs)
			var price int
			db.QueryRowContext(ctx, "SELECT price FROM products WHERE id = $1", pID).Scan(&price)
			
			qty := rand.Intn(3) + 1
			subtotal := price * qty
			totalAmount += subtotal

			_, err = db.ExecContext(ctx,
				"INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal) VALUES ($1, $2, $3, $4, $5)",
				saleID, pID, qty, price, subtotal,
			)
		}

		// Update sale total
		_, err = db.ExecContext(ctx, "UPDATE sales SET total_amount = $1, subtotal = $1 WHERE id = $2", totalAmount, saleID)

		if i%1000 == 0 {
			fmt.Printf("  ...%d transactions injected\n", i)
		}
	}

	fmt.Println("Injection completed successfully!")
	return nil
}

func getIDs(ctx context.Context, db *sql.DB, table string) []int {
	rows, err := db.QueryContext(ctx, fmt.Sprintf("SELECT id FROM %s LIMIT 100", table))
	if err != nil {
		return nil
	}
	defer rows.Close()
	ids := []int{}
	for rows.Next() {
		var id int
		rows.Scan(&id)
		ids = append(ids, id)
	}
	return ids
}

func randElem(s []string) string {
	return s[rand.Intn(len(s))]
}

func randElemInt(s []int) int {
	return s[rand.Intn(len(s))]
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

	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable&timezone=Asia/Jakarta",
		user, password, host, port, dbname)
}