package main

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	// Load environment variables
	cwd, _ := os.Getwd()
	if err := godotenv.Load(filepath.Join(cwd, ".env")); err != nil {
		log.Println("No .env file found, using defaults")
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	fmt.Println("Starting seeder...")

	// 1. Clear Data (Optional, but good for consistent results)
	fmt.Println("Cleaning existing data...")
	_, err = db.Exec(`TRUNCATE sale_items, sales, products, product_groups RESTART IDENTITY CASCADE`)
	if err != nil {
		log.Fatalf("failed to truncate: %v", err)
	}

	// 2. Insert Categories
	categories := []string{"Minuman", "Makanan Ringan", "Sembako", "Alat Tulis", "Kebutuhan Rumah"}
	categoryIDs := []int{}
	for _, name := range categories {
		var id int
		err := db.QueryRow(`INSERT INTO product_groups (name, description) VALUES ($1, $2) RETURNING id`, name, name+" deskripsi").Scan(&id)
		if err != nil {
			log.Fatal(err)
		}
		categoryIDs = append(categoryIDs, id)
	}

	// 3. Insert Products
	fmt.Println("Inserting products...")
	
	// Predefined templates to keep some data consistent
	templates := []struct {
		namePrefix string
		minPrice   int
		maxPrice   int
	}{
		{"Minuman", 3000, 20000},
		{"Snack", 2000, 15000},
		{"Kebutuhan", 10000, 100000},
		{"Elektronik", 50000, 500000},
		{"Pakaian", 35000, 250000},
	}

	productIDs := []int{}
	
	// Helper for product naming to ensure variety
	suffixes := []string{"Premium", "Hemat", "Promo", "Baru", "Original", "Plus", "Double", "Mini", "Jumbo"}

	for i := 1; i <= 100; i++ {
		tIdx := rand.Intn(len(templates))
		catIdx := rand.Intn(len(categoryIDs))
		
		name := fmt.Sprintf("%s %s %d", templates[tIdx].namePrefix, suffixes[rand.Intn(len(suffixes))], i)
		price := (rand.Intn(templates[tIdx].maxPrice - templates[tIdx].minPrice) + templates[tIdx].minPrice) / 100 * 100
		stock := rand.Intn(100) + 5
		sku := fmt.Sprintf("SKU-%04d", i)
		
		// Generate random 13-digit barcode (starting with 899 for ID or others)
		barcodeVal := fmt.Sprintf("899%010d", rand.Int63n(10000000000))
		
		var id int
		err := db.QueryRow(`INSERT INTO products (name, sku, barcode, price, stock, group_id) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
			name, sku, barcodeVal, price, stock, categoryIDs[catIdx]).Scan(&id)
		if err != nil {
			log.Fatal(err)
		}
		productIDs = append(productIDs, id)
	}

	// 4. Insert Sales History (Last 60 days)
	fmt.Println("Generating sales history...")
	now := time.Now()
	cashierID := 1 // Assuming ID 1 exists (admin)

	for i := 0; i < 60; i++ {
		day := now.AddDate(0, 0, -i)
		// Random number of transactions per day (roughly 5-15)
		numTx := rand.Intn(10) + 5
		
		for j := 0; j < numTx; j++ {
			// Random time during the day
			txTime := time.Date(day.Year(), day.Month(), day.Day(), rand.Intn(24), rand.Intn(60), rand.Intn(60), 0, day.Location())
			
			// Random number of items in sale (1-4)
			numItems := rand.Intn(3) + 1
			totalAmount := 0
			
			type Item struct {
				pID   int
				qty   int
				price int
				name  string
			}
			items := []Item{}
			
			for k := 0; k < numItems; k++ {
				pIdx := rand.Intn(len(productIDs))
				qty := rand.Intn(3) + 1
				
				// Fetch price and name from DB or use random values for simplicity since they are just dummy data
				// To keep it simple, we'll just mock it or fetch the product info
				var pName string
				var pPrice int
				db.QueryRow(`SELECT name, price FROM products WHERE id = $1`, productIDs[pIdx]).Scan(&pName, &pPrice)
				
				items = append(items, Item{productIDs[pIdx], qty, pPrice, pName})
				totalAmount += pPrice * qty
			}

			var saleID int
			err := db.QueryRow(`INSERT INTO sales (total_amount, payment_method, cashier_id, created_at) VALUES ($1, $2, $3, $4) RETURNING id`,
				totalAmount, "cash", cashierID, txTime).Scan(&saleID)
			if err != nil {
				log.Fatal(err)
			}

			for _, item := range items {
				_, err = db.Exec(`INSERT INTO sale_items (sale_id, product_id, product_name, quantity, price_at_sale) VALUES ($1, $2, $3, $4, $5)`,
					saleID, item.pID, item.name, item.qty, item.price)
				if err != nil {
					log.Fatal(err)
				}
			}
		}
	}

	fmt.Println("Seeder finished successfully!")
}
