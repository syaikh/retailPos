package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

// ---------- types ----------

type dailySaleProduct struct {
	ID       int
	Price    int
	Category string
	Barcode  string
	Cost     int
}

type dailySaleItem struct {
	ProductID  int
	Quantity   int
	UnitPrice  int
	Subtotal   int
}

type dailySaleRecord struct {
	Invoice       string
	CashierID     int
	StoreID       sql.NullInt64
	PaymentMethod string
	CreatedAt     time.Time
	TotalAmount   int
	Items         []dailySaleItem
}

// ---------- globals set by flags ----------

var (
	dailyDateStr   string
	dailyMin       int
	dailyMax       int
	dailyCashierID int
	dailyStoreID   int
	dailyInsertStock bool
)

func registerDailyFlags() {
	// -daily is a no-op toggle; the real mode flag scans os.Args in main().
	_ = flag.Bool("daily", false, "Run in daily-seed mode")

	flag.StringVar(&dailyDateStr, "daily.date", time.Now().Format("2006-01-02"),
		"Target date (YYYY-MM-DD) for daily transactions. Default: today")
	flag.IntVar(&dailyMin, "daily.min", 10,
		"Min transactions to generate (default: 10)")
	flag.IntVar(&dailyMax, "daily.max", 50,
		"Max transactions to generate (default: 50)")
	flag.IntVar(&dailyCashierID, "daily.cashier-id", 0,
		"Force a specific cashier user ID (0 = random)")
	flag.IntVar(&dailyStoreID, "daily.store-id", 0,
		"Force a specific store ID (0 = random)")
	flag.BoolVar(&dailyInsertStock, "daily.insert-stock", false,
		"Insert inventory_movements rows for each sale (default: false)")
}

// ---------- public entry point ----------

// RunDaily is called from main() when any daily.* flag is detected
func RunDaily(db *sql.DB) error {
	ctx := context.Background()

	targetDate, err := time.Parse("2006-01-02", dailyDateStr)
	if err != nil {
		return fmt.Errorf("invalid date %q (expected YYYY-MM-DD): %w", dailyDateStr, err)
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	dateConsidered := targetDate
	if targetDate.After(today) {
		fmt.Printf("⚠️  %s is beyond today (%s); using today instead\n",
			targetDate.Format("2006-01-02"), today.Format("2006-01-02"))
		dateConsidered = today
	}

	if dailyMin <= 0 || dailyMax <= 0 {
		return fmt.Errorf("--daily.min / --daily.max must be positive")
	}
	if dailyMax < dailyMin {
		dailyMin, dailyMax = dailyMax, dailyMin
	}

	salesCount := dailyMin + rand.Intn(dailyMax-dailyMin+1)

	fmt.Printf("📅 Daily seeder — %s — target: %d – %d transactions\n",
		dateConsidered.Format("2006-01-02"), dailyMin, dailyMax)
	fmt.Printf("   🎲 Randomised count: %d\n", salesCount)

	// 0. Read current invoice counter for this year to avoid PK collisions
	year := dateConsidered.Year()
	yearStr := fmt.Sprintf("%d", year)
	var maxSeq int
	err = db.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(
		    CAST(SUBSTRING(invoice_number FROM '\d+$') AS INTEGER)
		 ), 0)
		 FROM sales
		 WHERE invoice_number LIKE $1`,
		"INV-"+yearStr+"-%",
	).Scan(&maxSeq)
	if err != nil {
		return fmt.Errorf("read max invoice: %w", err)
	}
	fmt.Printf("   📊 Existing max sequence for %s: %d\n", yearStr, maxSeq)

	// 1. Load products
	products, err := loadProducts(ctx, db)
	if err != nil {
		return err
	}
	if len(products) == 0 {
		return fmt.Errorf("no active products — seed products first with: go run ./cmd/dummy -products 100 -days 0")
	}

	// 2. Load cashier user IDs (mirrors getIDs(ctx, db, "users") in main.go)
	cashierIDs, err := loadCashierUserIDs(ctx, db)
	if err != nil {
		return err
	}
	if len(cashierIDs) == 0 {
		return fmt.Errorf("no users found — run migrations/seeds first")
	}

	// 3. Load store IDs
	storeIDs, err := loadStoreIDs(ctx, db)
	if err != nil {
		return err
	}

	// 4. Build sales records in memory (start numbering after maxSeq)
	records := buildRecords(products, storeIDs, cashierIDs, dateConsidered, now, salesCount, maxSeq)

	// 5. Persist
	inserted := insertRecords(ctx, db, records)

	fmt.Printf("✅ Daily seeder done — %d transactions inserted for %s\n",
		inserted, dateConsidered.Format("2006-01-02"))
	return nil
}

// ---------- data loading ----------

// loadProducts mirrors getExistingProducts in main.go
func loadProducts(ctx context.Context, db *sql.DB) ([]dailySaleProduct, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT p.id, p.price, c.name as category_name
		FROM products p
		JOIN categories c ON p.category_id = c.id
		WHERE p.status = 'active'
		ORDER BY p.id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]dailySaleProduct, 0)
	for rows.Next() {
		var p dailySaleProduct
		if err := rows.Scan(&p.ID, &p.Price, &p.Category); err == nil {
			out = append(out, p)
		}
	}
	return out, nil
}

// loadCashierUserIDs mirrors getIDs(ctx, db, "users") in main.go — no role filter
func loadCashierUserIDs(ctx context.Context, db *sql.DB) ([]int, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT id FROM users WHERE is_active = true AND deleted_at IS NULL ORDER BY id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]int, 0)
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err == nil {
			out = append(out, id)
		}
	}
	return out, nil
}

// storeIDForSale returns a sql.NullInt64 — store_id is optional in the schema
func storeIDForSale(storeIDs []int, forcedID int) sql.NullInt64 {
	if forcedID > 0 {
		return sql.NullInt64{Int64: int64(forcedID), Valid: true}
	}
	if len(storeIDs) == 0 {
		return sql.NullInt64{Valid: false}
	}
	return sql.NullInt64{Int64: int64(storeIDs[rand.Intn(len(storeIDs))]), Valid: true}
}

// loadStoreIDs queries the stores table (mirrors the store lookup in main.go's processIndividualSale path)
func loadStoreIDs(ctx context.Context, db *sql.DB) ([]int, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT id FROM stores WHERE is_active = true ORDER BY id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]int, 0)
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err == nil {
			out = append(out, id)
		}
	}
	return out, nil
}

// ---------- record building ----------

func buildRecords(
	products []dailySaleProduct,
	storeIDs []int,
	cashierIDs []int,
	targetDate, now time.Time,
	salesCount, startSeq int,
) []dailySaleRecord {
	records := make([]dailySaleRecord, 0, salesCount)
	for i := 0; i < salesCount; i++ {
		createdAt := randomTimeInDate(targetDate, now)

		numItems := 1 + rand.Intn(4) // 1-4 items (most common)
		if rand.Intn(100) < 20 {
			numItems += rand.Intn(4) // 20% chance of 5-8 items
		}

		items := selectItems(products, numItems)
		if len(items) == 0 {
			continue
		}

		total := 0
		for _, it := range items {
			total += it.Subtotal
		}

		seq := startSeq + i + 1 // +1 because startSeq=0 means first invoice is 1
		cashierID := cashierIDs[rand.Intn(len(cashierIDs))]
		storeID := storeIDForSale(storeIDs, dailyStoreID)

		records = append(records, dailySaleRecord{
			Invoice:       fmt.Sprintf("INV-%d-%06d", targetDate.Year(), seq),
			CashierID:     cashierID,
			StoreID:       storeID,
			PaymentMethod: weightedPick(paymentMethods, paymentWeights),
			CreatedAt:     createdAt,
			TotalAmount:   total,
			Items:         items,
		})
	}
	return records
}

func randomTimeInDate(d, now time.Time) time.Time {
	dStart := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, d.Location())

	isToday := d.Year() == now.Year() && d.Month() == now.Month() && d.Day() == now.Day()
	if isToday {
		// Only generate times that are already in the past (up to 1 hour ago)
		oneHourAgo := now.Add(-1 * time.Hour)
		if !oneHourAgo.After(dStart) {
			oneHourAgo = dStart.Add(8 * time.Hour)
		}
		rangeSec := int64(oneHourAgo.Sub(dStart).Seconds())
		if rangeSec <= 0 {
			rangeSec = 60
		}
		offsetSec := rand.Int63n(rangeSec)
		return dStart.Add(time.Duration(offsetSec) * time.Second)
	}

	// Past day: random within business hours 07:00 - 21:00
	bizStart := 7
	bizEnd := 21
	hour := bizStart + rand.Intn(bizEnd-bizStart)
	minute := rand.Intn(60)
	second := rand.Intn(60)
	return time.Date(d.Year(), d.Month(), d.Day(), hour, minute, second, 0, d.Location())
}

func selectItems(products []dailySaleProduct, count int) []dailySaleItem {
	if len(products) == 0 {
		return nil
	}
	items := make([]dailySaleItem, 0, count)
	for i := 0; i < count; i++ {
		p := products[rand.Intn(len(products))]
		qty := pickQty(p.Category)
		items = append(items, dailySaleItem{
			ProductID:  p.ID,
			Quantity:   qty,
			UnitPrice:  p.Price,
			Subtotal:   p.Price * qty,
		})
	}
	return items
}

func pickQty(cat string) int {
	catLower := strings.ToLower(cat)
	switch {
	case strings.Contains(catLower, "grocery"),
		strings.Contains(catLower, "snack"),
		strings.Contains(catLower, "beverage"),
		strings.Contains(catLower, "food"),
		strings.Contains(catLower, "dairy"),
		strings.Contains(catLower, "frozen"):
		if rand.Intn(100) < 65 {
			return rand.Intn(3) + 1 // 1-3
		}
		return rand.Intn(5) + 4 // 4-8 bulk
	case strings.Contains(catLower, "smartphone"),
		strings.Contains(catLower, "laptop"),
		strings.Contains(catLower, "furniture"),
		strings.Contains(catLower, "appliance"),
		strings.Contains(catLower, "camera"),
		strings.Contains(catLower, "tv"):
		return 1
	default:
		if rand.Intn(100) < 80 {
			return 1
		}
		return 2
	}
}

func weightedPick(items []string, weights []int) string {
	total := 0
	for _, w := range weights {
		total += w
	}
	r := rand.Intn(total)
	cum := 0
	for i, item := range items {
		cum += weights[i]
		if r < cum {
			return item
		}
	}
	return items[0]
}

// ---------- persistence ----------

func insertRecords(ctx context.Context, db *sql.DB, records []dailySaleRecord) int {
	inserted := 0
	var mu sync.Mutex

	const workers = 4
	// Buffered large enough so that errCh never blocks workers
	errCh := make(chan error, len(records))

	var wg sync.WaitGroup
	// Feed jobs and close the channel from inside a goroutine so that
	// wg.Wait() is the only caller of close(jobs) — no data race.
	jobs := make(chan dailySaleRecord, len(records))

	go func() {
		for _, sale := range records {
			jobs <- sale
		}
		close(jobs)
	}()

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for sale := range jobs {
				if err := persistOne(ctx, db, sale); err != nil {
					errCh <- fmt.Errorf("invoice %s: %w", sale.Invoice, err)
				} else {
					mu.Lock()
					inserted++
					mu.Unlock()
				}
			}
		}()
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		fmt.Printf("   ⚠️  %v\n", err)
	}
	return inserted
}

func persistOne(ctx context.Context, db *sql.DB, sale dailySaleRecord) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var saleID int
	err = tx.QueryRowContext(ctx, `
		INSERT INTO sales
			(invoice_number, cashier_id, store_id, subtotal, total_amount, payment_method, status, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, 'completed', $7)
		 RETURNING id
	`, sale.Invoice, sale.CashierID, sale.StoreID, sale.TotalAmount, sale.TotalAmount,
		sale.PaymentMethod, sale.CreatedAt).Scan(&saleID)
	if err != nil {
		return fmt.Errorf("insert sale: %w", err)
	}

	for _, item := range sale.Items {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal)
			 VALUES ($1, $2, $3, $4, $5)`,
			saleID, item.ProductID, item.Quantity, item.UnitPrice, item.Subtotal,
		); err != nil {
			return fmt.Errorf("insert item: %w", err)
		}
	}

	if dailyInsertStock {
		for _, item := range sale.Items {
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO inventory_movements
					(product_id, quantity_change, type, reference_id, reference_table, user_id, created_at)
				 VALUES ($1, $2, 'sale', $3, 'sales', $4, $5)`,
				item.ProductID, -item.Quantity, saleID, sale.CashierID, sale.CreatedAt,
			); err != nil {
				return fmt.Errorf("insert inventory movement: %w", err)
			}
		}
	}

	return tx.Commit()
}
