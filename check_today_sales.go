package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	db, err := sql.Open("postgres", "postgres://pos:admin123@localhost:5432/retail_pos?sslmode=disable&timezone=Asia/Jakarta")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Check today's transactions
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sales WHERE DATE(created_at) = CURRENT_DATE").Scan(&count)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Total transactions today (2026-05-08): %d\n", count)

	if count > 0 {
		// Show some details
		rows, err := db.Query(`
			SELECT s.id, s.invoice_number, s.total_amount, s.created_at, u.username as cashier
			FROM sales s
			JOIN users u ON s.cashier_id = u.id
			WHERE DATE(s.created_at) = CURRENT_DATE
			ORDER BY s.created_at DESC
			LIMIT 5
		`)
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()

		fmt.Println("\nRecent transactions today:")
		for rows.Next() {
			var id int
			var invoice string
			var total int
			var created_at string
			var cashier string
			rows.Scan(&id, &invoice, &total, &created_at, &cashier)
			fmt.Printf("- %s: Rp%d by %s at %s\n", invoice, total, cashier, created_at)
		}
	}
}
