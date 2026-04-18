package main

import (
	"log"
	model "retailPos/internal/model"
	"retailPos/internal/repo"

	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	db, err := repo.NewDB()
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	userRepo := repo.NewUserRepo(db)
	// Admin user
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte("admin123"), 14)
	if err != nil {
		log.Fatalf("Failed to hash admin password: %v", err)
	}
	hashedPassword := string(hashedBytes)
	admin := &model.User{
		Username:     "admin",
		PasswordHash: hashedPassword,
		Role:         "admin",
	}

	err = userRepo.Create(admin)
	if err != nil {
		log.Printf("Admin user might already exist: %v", err)
	} else {
		log.Println("Admin user created successfully")
	}

	// Cashier user
	hashedBytes, err = bcrypt.GenerateFromPassword([]byte("cashier123"), 14)
	if err != nil {
		log.Fatalf("Failed to hash cashier password: %v", err)
	}
	hashedPassword = string(hashedBytes)
	cashier := &model.User{
		Username:     "cashier",
		PasswordHash: hashedPassword,
		Role:         "cashier",
	}

	err = userRepo.Create(cashier)
	if err != nil {
		log.Printf("Cashier user might already exist: %v", err)
	} else {
		log.Println("Cashier user created successfully")
	}
}
