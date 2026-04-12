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
	productItems := []struct {
		name    string
		catIdx  int
		price   int
		stock   int
	}{
		{"Aqua 600ml", 0, 3500, 50},
		{"Coca Cola 330ml", 0, 6000, 24},
		{"Teh Pucuk", 0, 4000, 100},
		{"Indemie Goreng", 1, 3000, 8},
		{"Chitato L", 1, 12000, 15},
		{"Beras 5kg", 2, 65000, 5},
		{"Minyak Goreng 1L", 2, 18000, 20},
		{"Buku Tulis Sidu", 3, 5000, 40},
		{"Pulpen Snowman", 3, 2500, 2},
		{"Sabun Cuci Piring", 4, 15000, 30},
	}

	productIDs := []int{}
	for i, p := range productItems {
		var id int
		sku := fmt.Sprintf("SKU-%04d", i+1)
		err := db.QueryRow(`INSERT INTO products (name, sku, price, stock, group_id) VALUES ($1, $2, $3, $4, $5) RETURNING id`,
			p.name, sku, p.price, p.stock, categoryIDs[p.catIdx]).Scan(&id)
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
				pIdx := rand.Intn(len(productItems))
				qty := rand.Intn(3) + 1
				price := productItems[pIdx].price
				name := productItems[pIdx].name
				items = append(items, Item{productIDs[pIdx], qty, price, name})
				totalAmount += price * qty
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
