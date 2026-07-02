package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"retail-pos-system/internal/audit"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		host := getEnv("DB_HOST", "localhost")
		port := getEnv("DB_PORT", "5433")
		user := getEnv("DB_USER", "pos")
		password := getEnv("DB_PASSWORD", "admin123")
		dbname := getEnv("DB_NAME", "retail_pos")
		dsn = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable&timezone=Asia/Jakarta",
			user, password, host, port, dbname)
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		log.Fatalf("Unable to connect: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()

	rows, err := pool.Query(ctx, `SELECT al.id, al.action, al.entity_type, al.entity_id, al.old_values, al.new_values, COALESCE(u.username, '') FROM audit_logs al LEFT JOIN users u ON al.user_id = u.id WHERE al.description IS NULL ORDER BY al.id`)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}
	defer rows.Close()

	var updated, failed int
	for rows.Next() {
		var id int
		var action, entityType string
		var entityID *int
		var oldValues, newValues []byte
		var username string

		if err := rows.Scan(&id, &action, &entityType, &entityID, &oldValues, &newValues, &username); err != nil {
			log.Printf("Scan error at id: %v", err)
			failed++
			continue
		}

		alog := &audit.AuditLog{
			Action:     action,
			EntityType: entityType,
			EntityID:   entityID,
			Username:   username,
		}

		if oldValues != nil {
			var ov interface{}
			if err := json.Unmarshal(oldValues, &ov); err == nil {
				alog.OldValues = ov
			}
		}
		if newValues != nil {
			var nv interface{}
			if err := json.Unmarshal(newValues, &nv); err == nil {
				alog.NewValues = nv
			}
		}

		desc := audit.GenerateAuditDescription(alog)
		if desc == "" {
			continue
		}

		if _, err := pool.Exec(ctx, `UPDATE audit_logs SET description = $1 WHERE id = $2`, desc, id); err != nil {
			log.Printf("Update error id=%d: %v", id, err)
			failed++
			continue
		}
		updated++
		if updated%100 == 0 {
			fmt.Printf("Progress: %d updated\n", updated)
		}
	}

	fmt.Printf("Done. %d updated, %d failed\n", updated, failed)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func init() {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		log.Printf("Warning: failed to load Asia/Jakarta timezone: %v", err)
	}
	_ = loc
}
