package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

func main() {
	// Connect to database
	db, err := sql.Open("postgres", getDSN())
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	fmt.Println("🔍 Revenue Calculation Verification")
	fmt.Println("====================================")

	// Test 1: Daily revenue for today
	fmt.Println("\n1. Daily Revenue (Today):")
	dailyRevenue, dailyCount := getDailyRevenue(ctx, db, time.Now().Format("2006-01-02"))
	fmt.Printf("   Revenue: Rp %d\n", dailyRevenue)
	fmt.Printf("   Transactions: %d\n", dailyCount)

	// Test 2: Weekly revenue for last 4 weeks
	fmt.Println("\n2. Weekly Revenue (Last 4 weeks):")
	weeks := getWeeklyRevenue(ctx, db, 4)
	for _, week := range weeks {
		fmt.Printf("   %s to %s: Rp %d (%d transactions)\n",
			week.Start, week.End, week.Revenue, week.Count)
	}

	// Test 3: Monthly revenue for last 3 months
	fmt.Println("\n3. Monthly Revenue (Last 3 months):")
	months := getMonthlyRevenue(ctx, db, 3)
	for _, month := range months {
		fmt.Printf("   %s: Rp %d (%d transactions)\n",
			month.Month, month.Revenue, month.Count)
	}

	// Test 4: Total products and low stock
	fmt.Println("\n4. Product Statistics:")
	totalProducts := getTotalProducts(ctx, db)
	lowStockCount := getLowStockCount(ctx, db)
	fmt.Printf("   Total Products: %d\n", totalProducts)
	fmt.Printf("   Low Stock (≤5): %d\n", lowStockCount)

	fmt.Println("\n✅ Verification complete! Compare these values with dashboard.")
}

type WeekData struct {
	Start   string
	End     string
	Revenue int
	Count   int
}

type MonthData struct {
	Month   string
	Revenue int
	Count   int
}

func getDailyRevenue(ctx context.Context, db *sql.DB, date string) (revenue, count int) {
	query := `
		SELECT COALESCE(SUM(total_amount), 0), COUNT(*)
		FROM sales
		WHERE DATE(created_at) = $1 AND status = 'completed'`

	err := db.QueryRowContext(ctx, query, date).Scan(&revenue, &count)
	if err != nil {
		log.Printf("Daily revenue query failed: %v", err)
	}
	return
}

func getWeeklyRevenue(ctx context.Context, db *sql.DB, weeks int) []WeekData {
	query := `
		SELECT
			DATE_TRUNC('week', created_at)::date as week_start,
			(DATE_TRUNC('week', created_at) + interval '6 days')::date as week_end,
			SUM(total_amount) as revenue,
			COUNT(*) as count
		FROM sales
		WHERE created_at >= $1
			AND status = 'completed'
		GROUP BY DATE_TRUNC('week', created_at)
		ORDER BY week_start DESC
		LIMIT $2`

	// Calculate the start date in Go
	startDate := time.Now().AddDate(0, 0, -weeks*7)

	rows, err := db.QueryContext(ctx, query, startDate, weeks)
	if err != nil {
		log.Printf("Weekly revenue query failed: %v", err)
		return nil
	}
	defer rows.Close()

	var results []WeekData
	for rows.Next() {
		var w WeekData
		err := rows.Scan(&w.Start, &w.End, &w.Revenue, &w.Count)
		if err != nil {
			log.Printf("Scan error: %v", err)
			continue
		}
		results = append(results, w)
	}
	return results
}

func getMonthlyRevenue(ctx context.Context, db *sql.DB, months int) []MonthData {
	query := `
		SELECT
			TO_CHAR(created_at, 'YYYY-MM') as month,
			SUM(total_amount) as revenue,
			COUNT(*) as count
		FROM sales
		WHERE created_at >= $1
			AND status = 'completed'
		GROUP BY TO_CHAR(created_at, 'YYYY-MM')
		ORDER BY month DESC
		LIMIT $2`

	// Calculate the start date in Go (months-1 because we want last N months)
	startDate := time.Now().AddDate(0, -(months-1), 0)

	rows, err := db.QueryContext(ctx, query, startDate, months)
	if err != nil {
		log.Printf("Monthly revenue query failed: %v", err)
		return nil
	}
	defer rows.Close()

	var results []MonthData
	for rows.Next() {
		var m MonthData
		err := rows.Scan(&m.Month, &m.Revenue, &m.Count)
		if err != nil {
			log.Printf("Scan error: %v", err)
			continue
		}
		results = append(results, m)
	}
	return results
}

func getTotalProducts(ctx context.Context, db *sql.DB) int {
	query := `SELECT COUNT(*) FROM products WHERE deleted_at IS NULL`
	var count int
	err := db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		log.Printf("Total products query failed: %v", err)
	}
	return count
}

func getLowStockCount(ctx context.Context, db *sql.DB) int {
	query := `SELECT COUNT(*) FROM products WHERE deleted_at IS NULL AND stock <= 5`
	var count int
	err := db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		log.Printf("Low stock query failed: %v", err)
	}
	return count
}

func getDSN() string {
	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "5432"
	}
	user := os.Getenv("DB_USER")
	if user == "" {
		user = "pos"
	}
	password := os.Getenv("DB_PASSWORD")
	if password == "" {
		password = "admin123"
	}
	dbname := os.Getenv("DB_NAME")
	if dbname == "" {
		dbname = "retail_pos"
	}

	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		user, password, host, port, dbname)
}