package main

import (
	"context"
	"log"
	"retail-pos-system/internal/domain"
	"retail-pos-system/internal/repository"

	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// DB connection pool with retry
	dbPool, err := repository.NewDBConnection()
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer dbPool.Close()

	// Debug: Check current database
	var dbName string
	err = dbPool.QueryRow(context.Background(), "SELECT current_database()").Scan(&dbName)
	if err != nil {
		log.Printf("Failed to get database name: %v", err)
	} else {
		log.Printf("Connected to database: %s", dbName)
	}

	// Create repositories
	userRepo := repository.NewPostgresRepository(dbPool)

	// Get admin role ID
	var adminRoleID int
	err = dbPool.QueryRow(context.Background(), "SELECT id FROM roles WHERE name = 'admin'").Scan(&adminRoleID)
	if err != nil {
		log.Fatalf("Failed to get admin role ID: %v", err)
	}

	// Get cashier role ID
	var cashierRoleID int
	err = dbPool.QueryRow(context.Background(), "SELECT id FROM roles WHERE name = 'cashier'").Scan(&cashierRoleID)
	if err != nil {
		log.Fatalf("Failed to get cashier role ID: %v", err)
	}

	// Admin user (superadmin role, store_id = NULL for admin - can see all)
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to hash admin password: %v", err)
	}
	hashedPassword := string(hashedBytes)

	// Check if superadmin already exists
	var existingID int
	err = dbPool.QueryRow(context.Background(), "SELECT id FROM users WHERE username = 'superadmin'").Scan(&existingID)
	if err == nil {
		log.Println("Superadmin user already exists, skipping creation")
	} else {
		admin := &domain.User{
			Username: "superadmin",
			Email:    "superadmin@retailpos.local",
			Password: hashedPassword,
			RoleID:   1, // superadmin role
			IsActive: true,
		}

		err = userRepo.CreateUser(context.Background(), admin)
		if err != nil {
			log.Printf("Failed to create superadmin user: %v", err)
		} else {
			log.Println("Superadmin user created successfully")
		}
	}

	// Cashier user (assigned to role_id = 4 for cashier)
	storeID := 1
	hashedBytes, err = bcrypt.GenerateFromPassword([]byte("cashier123"), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to hash cashier password: %v", err)
	}
	hashedPassword = string(hashedBytes)

	// Check if cashier already exists
	err = dbPool.QueryRow(context.Background(), "SELECT id FROM users WHERE username = 'cashier'").Scan(&existingID)
	if err == nil {
		log.Println("Cashier user already exists, skipping creation")
	} else {
		cashier := &domain.User{
			Username: "cashier",
			Email:    "cashier@retailpos.local",
			Password: hashedPassword,
			RoleID:   cashierRoleID,
			StoreID:  &storeID,
			IsActive: true,
		}

		err = userRepo.CreateUser(context.Background(), cashier)
		if err != nil {
			log.Printf("Failed to create cashier user: %v", err)
		} else {
			log.Println("Cashier user created successfully")
		}
	}

	log.Println("Database seeding completed successfully!")
}
