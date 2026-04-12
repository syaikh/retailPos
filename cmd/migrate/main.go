package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	cwd, _ := os.Getwd()
	godotenv.Load(filepath.Join(cwd, ".env"))

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

	migration := `
	-- Add product_name column to sale_items for snapshotting
	ALTER TABLE sale_items ADD COLUMN IF NOT EXISTS product_name TEXT;

	-- Move data from products table to sale_items for existing records
	UPDATE sale_items si 
	SET product_name = p.name 
	FROM products p 
	WHERE si.product_name IS NULL AND si.product_id = p.id;
	`

	fmt.Println("Applying migration...")
	_, err = db.Exec(migration)
	if err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	fmt.Println("Migration applied successfully!")
}
