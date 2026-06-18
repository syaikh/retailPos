package repository

import (
	"context"
	"fmt"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// connectBenchDB connects to the dev database (must have seeded sales data).
func connectBenchDB() (*pgxpool.Pool, error) {
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5433")
	user := getEnv("DB_USER", "pos")
	password := getEnv("DB_PASSWORD", "admin123")
	dbName := getEnv("DB_NAME", "retail_pos")

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable&timezone=Asia/Jakarta",
		user, password, host, port, dbName)

	return pgxpool.New(context.Background(), dsn)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// ---- Approach A: Go Loop Aggregation ----

// aggregateDailyGo fetches sales for a period and aggregates by day in Go.
func aggregateDailyGo(ctx context.Context, repo *postgresRepository, startDate, endDate string) []ChartDataPoint {
	sales, _, err := repo.GetAllSales(ctx, 50000, 0, "", "created_at", "ASC", startDate, endDate, nil, "", nil, nil)
	if err != nil {
		return nil
	}

	// Aggregate by date
	byDate := make(map[string]int)
	for _, s := range sales {
		t, err := time.Parse(time.RFC3339, s.CreatedAt)
		if err != nil {
			continue
		}
		date := t.In(mustLoadJakarta()).Format("2006-01-02")
		byDate[date] += s.TotalAmount
	}

	// Fill gaps
	start, _ := time.ParseInLocation("2006-01-02", startDate, mustLoadJakarta())
	end, _ := time.ParseInLocation("2006-01-02", endDate, mustLoadJakarta())

	var result []ChartDataPoint
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")
		result = append(result, ChartDataPoint{
			Date:  dateStr,
			Total: byDate[dateStr],
		})
	}
	return result
}

// goLoopDual runs Approach A: 2× GetAllSales + Go aggregation.
func goLoopDual(ctx context.Context, repo *postgresRepository, cs, ce, ps, pe string) (current, previous []ChartDataPoint) {
	current = aggregateDailyGo(ctx, repo, cs, ce)
	previous = aggregateDailyGo(ctx, repo, ps, pe)
	return
}

// ---- Approach B: SQL CTE ----

// sqlCTEDual runs Approach B: single GetDualChartData call.
func sqlCTEDual(ctx context.Context, repo *postgresRepository, cs, ce, ps, pe time.Time) (current, previous []ChartDataPoint) {
	current, previous, err := repo.GetDualChartData(ctx, cs, ce, ps, pe)
	if err != nil {
		return nil, nil
	}
	return
}

// ---- Benchmark runner ----

type benchCase struct {
	name     string
	currentS string
	currentE string
	prevS    string
	prevE    string
}

func BenchmarkChartDual(b *testing.B) {
	pool, err := connectBenchDB()
	if err != nil {
		b.Fatalf("connect to dev db: %v", err)
	}
	defer pool.Close()

	repo := NewPostgresRepository(pool)

	// Verify we have data
	var salesCount int
	err = pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM sales").Scan(&salesCount)
	if err != nil || salesCount == 0 {
		b.Fatalf("no sales data in dev db (count=%d, err=%v)", salesCount, err)
	}
	b.Logf("Sales count: %d", salesCount)

	// ---- Date ranges ----
	// Use 2026 dates which have ~2547 sales
	cases := []benchCase{
		{
			name:     "7d/7d",
			currentS: "2026-06-10",
			currentE: "2026-06-16",
			prevS:    "2026-06-03",
			prevE:    "2026-06-09",
		},
		{
			name:     "30d/30d",
			currentS: "2026-05-18",
			currentE: "2026-06-16",
			prevS:    "2026-04-18",
			prevE:    "2026-05-17",
		},
		{
			name:     "90d/90d",
			currentS: "2026-03-19",
			currentE: "2026-06-16",
			prevS:    "2025-12-19",
			prevE:    "2026-03-18",
		},
		{
			name:     "365d/365d",
			currentS: "2025-06-17",
			currentE: "2026-06-16",
			prevS:    "2024-06-17",
			prevE:    "2025-06-16",
		},
	}

	// Parse time.Time for approach B
	type parsedCase struct {
		name     string
		cs, ce   time.Time
		ps, pe   time.Time
		csStr    string
		ceStr    string
		psStr    string
		peStr    string
	}

	var parsed []parsedCase
	for _, c := range cases {
		loc := mustLoadJakarta()
		cs, _ := time.ParseInLocation("2006-01-02", c.currentS, loc)
		ce, _ := time.ParseInLocation("2006-01-02", c.currentE, loc)
		ps, _ := time.ParseInLocation("2006-01-02", c.prevS, loc)
		pe, _ := time.ParseInLocation("2006-01-02", c.prevE, loc)
		parsed = append(parsed, parsedCase{
			name:  c.name,
			cs:    cs, ce: ce, ps: ps, pe: pe,
			csStr: c.currentS, ceStr: c.currentE, psStr: c.prevS, peStr: c.prevE,
		})
	}

	// ---- Approach A: Go Loop ----
	for _, p := range parsed {
		b.Run(fmt.Sprintf("GoLoop/%s", p.name), func(b *testing.B) {
			ctx := context.Background()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				cur, prev := goLoopDual(ctx, repo, p.csStr, p.ceStr, p.psStr, p.peStr)
				if len(cur) == 0 || len(prev) == 0 {
					b.Fatal("empty result")
				}
			}
		})
	}

	// ---- Approach B: SQL CTE ----
	for _, p := range parsed {
		b.Run(fmt.Sprintf("SQL_CTE/%s", p.name), func(b *testing.B) {
			ctx := context.Background()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				cur, prev := sqlCTEDual(ctx, repo, p.cs, p.ce, p.ps, p.pe)
				if len(cur) == 0 || len(prev) == 0 {
					b.Fatal("empty result")
				}
			}
		})
	}
}

// ---- Correctness check (single run, not benchmark) ----

func TestDualApproachesMatch(t *testing.T) {
	pool, err := connectBenchDB()
	if err != nil {
		t.Skipf("dev db not available: %v", err)
	}
	defer pool.Close()

	repo := NewPostgresRepository(pool)
	ctx := context.Background()
	loc := mustLoadJakarta()

	ranges := []struct {
		name string
		cs, ce, ps, pe string
	}{
		{"7d", "2026-06-10", "2026-06-16", "2026-06-03", "2026-06-09"},
		{"30d", "2026-05-18", "2026-06-16", "2026-04-18", "2026-05-17"},
	}

	for _, r := range ranges {
		t.Run(r.name, func(t *testing.T) {
			csT, _ := time.ParseInLocation("2006-01-02", r.cs, loc)
			ceT, _ := time.ParseInLocation("2006-01-02", r.ce, loc)
			psT, _ := time.ParseInLocation("2006-01-02", r.ps, loc)
			peT, _ := time.ParseInLocation("2006-01-02", r.pe, loc)

			curA, prevA := goLoopDual(ctx, repo, r.cs, r.ce, r.ps, r.pe)
			curB, prevB := sqlCTEDual(ctx, repo, csT, ceT, psT, peT)

			if len(curA) != len(curB) {
				t.Errorf("current length mismatch: A=%d B=%d", len(curA), len(curB))
			}
			if len(prevA) != len(prevB) {
				t.Errorf("prev length mismatch: A=%d B=%d", len(prevA), len(prevB))
			}

			// Sort both by date for comparison
			sort.Slice(curA, func(i, j int) bool { return curA[i].Date < curA[j].Date })
			sort.Slice(curB, func(i, j int) bool { return curB[i].Date < curB[j].Date })
			sort.Slice(prevA, func(i, j int) bool { return prevA[i].Date < prevA[j].Date })
			sort.Slice(prevB, func(i, j int) bool { return prevB[i].Date < prevB[j].Date })

			// Compare totals (allow small variance in timezone handling)
			for i := 0; i < len(curA) && i < len(curB); i++ {
				d := curA[i].Total - curB[i].Total
				if d < 0 {
					d = -d
				}
				if d > 1000 {
					t.Errorf("current %s diff: A=%d B=%d", curA[i].Date, curA[i].Total, curB[i].Total)
				}
			}
			for i := 0; i < len(prevA) && i < len(prevB); i++ {
				d := prevA[i].Total - prevB[i].Total
				if d < 0 {
					d = -d
				}
				if d > 1000 {
					t.Errorf("prev %s diff: A=%d B=%d", prevA[i].Date, prevA[i].Total, prevB[i].Total)
				}
			}
		})
	}
}

// Verify mustLoadJakarta exists in this package
var _ = mustLoadJakarta
