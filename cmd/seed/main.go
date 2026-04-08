package main

import (
	"log"
	"retailPos/internal/auth"
	"retailPos/internal/model"
	"retailPos/internal/repo"

	"github.com/joho/godotenv"
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
	authService := auth.NewAuthService(userRepo)

	// Admin user
	hashedPassword, _ := authService.HashPassword("admin123")
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
	hashedPassword, _ = authService.HashPassword("cashier123")
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
