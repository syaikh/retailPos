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

	if err := db.Ping(); err != nil {
		log.Fatalf("Cannot connect to database: %v", err)
	}
	log.Println("Connected to database")

	migrations := []string{
		filepath.Join(cwd, "migrations", "0003_roles_table.sql"),
		filepath.Join(cwd, "migrations", "0004_seed_roles_permissions.sql"),
		filepath.Join(cwd, "migrations", "0005_migrate_user_roles.sql"),
	}

	for _, path := range migrations {
		content, err := os.ReadFile(path)
		if err != nil {
			log.Fatalf("Failed to read %s: %v", path, err)
		}
		log.Printf("Applying %s...", filepath.Base(path))
		if _, err := db.Exec(string(content)); err != nil {
			log.Fatalf("Migration %s failed: %v", filepath.Base(path), err)
		}
		log.Printf("✓ Applied %s", filepath.Base(path))
	}

	log.Println("All RBAC migrations applied successfully")
	fmt.Println("\nVerification queries:")
	fmt.Println("  SELECT * FROM roles;")
	fmt.Println("  SELECT * FROM permissions;")
	fmt.Println("  SELECT u.id, u.username, u.role, u.role_id, r.name AS resolved FROM users u JOIN roles r ON u.role_id = r.id;")
}
