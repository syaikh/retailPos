package main

import (
	"bufio"
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

type ProductTemplate struct {
	Category     string
	NamePatterns []string
	Brands       []string
	Models       []string
	Variants     []string
	PriceMin     int // in thousands IDR
	PriceMax     int // in thousands IDR
}

var (
	// Payment methods aligned with payment_methods master data
	paymentMethods = []string{"CASH", "QRIS", "CARD", "TRANSFER", "E_WALLET"}
	paymentWeights = []int{40, 35, 15, 5, 5} // Percentages

	// Comprehensive category list
	categories = []string{
		// Electronics (20 categories)
		"Smartphones", "Laptops", "Tablets", "Audio Equipment", "Cameras",
		"Computer Components", "Networking", "Mobile Accessories", "Gaming",
		"Smart Home Devices", "Wearable Tech", "TV & Displays", "Printers",
		"Storage Devices", "Power Banks", "Cables & Adapters", "Chargers",
		"Computer Peripherals", "Video Games", "Software",

		// Fashion & Apparel (15 categories)
		"Men's Clothing", "Women's Clothing", "Kids' Clothing", "Shoes",
		"Bags & Luggage", "Jewelry", "Watches", "Sunglasses", "Hats & Caps",
		"Belts", "Scarves", "Gloves", "Underwear", "Sportswear", "Accessories",

		// Home & Living (15 categories)
		"Furniture", "Kitchen Appliances", "Home Decor", "Bedding",
		"Bathroom", "Lighting", "Storage & Organization", "Cleaning Supplies",
		"Kitchenware", "Tableware", "Home Textiles", "Garden & Outdoor",
		"Tools & Hardware", "Paint & Supplies", "Home Improvement",

		// Food & Beverage (10 categories)
		"Groceries", "Snacks", "Beverages", "Dairy Products", "Bakery",
		"Fresh Produce", "Frozen Foods", "Canned Goods", "Condiments",
		"Health Foods",

		// Health & Beauty (8 categories)
		"Skincare", "Hair Care", "Makeup", "Personal Care", "Health Supplements",
		"Medical Supplies", "Fitness Equipment", "Wellness Products",

		// Sports & Outdoors (8 categories)
		"Sports Equipment", "Camping & Hiking", "Cycling", "Water Sports",
		"Team Sports", "Fitness Gear", "Outdoor Clothing", "Sport Accessories",

		// Books & Entertainment (6 categories)
		"Books", "Magazines", "Music", "Movies", "Games", "Art Supplies",

		// Automotive & Transportation (5 categories)
		"Car Parts", "Motorcycle Parts", "Car Accessories", "Tires",
		"Automotive Tools",

		// Other categories (8 categories)
		"Pet Supplies", "Baby Products", "Office Supplies", "Party Supplies",
		"Seasonal Items", "Collectibles", "Industrial Supplies", "Chemicals",
	}

	productTemplates = map[string]ProductTemplate{
		"Smartphones": {
			NamePatterns: []string{"%s %s %sGB"},
			Brands:       []string{"Samsung", "Apple", "Xiaomi", "Oppo", "Vivo", "Realme", "OnePlus", "Google", "Huawei", "Sony"},
			Models:       []string{"Galaxy S", "Galaxy A", "Galaxy Note", "iPhone", "Mi", "Redmi", "Find X", "Y", "Pixel", "P"},
			Variants:     []string{"128", "256", "512", "1TB"},
			PriceMin:     2000, PriceMax: 15000,
		},
		"Laptops": {
			NamePatterns: []string{"%s %s %sGB RAM"},
			Brands:       []string{"Dell", "HP", "Lenovo", "Asus", "Acer", "Apple", "MSI", "Samsung", "LG", "Sony"},
			Models:       []string{"Inspiron", "Pavilion", "ThinkPad", "ZenBook", "Predator", "MacBook", "GE", "Galaxy Book", "Gram", "Vaio"},
			Variants:     []string{"8", "16", "32"},
			PriceMin:     5000, PriceMax: 25000,
		},
		"Furniture": {
			NamePatterns: []string{"%s %s %s"},
			Brands:       []string{"IKEA", "Informa", "Olympic", "King", "Global", "Nitori", "Homes", "FurniturePlus"},
			Models:       []string{"Chair", "Table", "Sofa", "Cabinet", "Bed", "Desk", "Bookshelf", "Dresser"},
			Variants:     []string{"Wood", "Metal", "Plastic", "Fabric", "Leather", "Glass"},
			PriceMin:     500, PriceMax: 10000,
		},
		"Groceries": {
			NamePatterns: []string{"%s %s %s"},
			Brands:       []string{"ABC", "Indofood", "Unilever", "Nestle", "Mayora", "Garudafood", "Wings", "Ultra"},
			Models:       []string{"Rice", "Oil", "Sugar", "Flour", "Milk", "Tea", "Coffee", "Soap"},
			Variants:     []string{"Premium", "Regular", "Large", "Small", "Organic", "Plain"},
			PriceMin:     10, PriceMax: 200,
		},
		"Books": {
			NamePatterns: []string{"%s %s %s"},
			Brands:       []string{"Gramedia", "Mizan", "Erlangga", "Yudhistira", "Bentang", "Pustaka", "Andi", "GagasMedia"},
			Models:       []string{"Novel", "Textbook", "Biography", "Cookbook", "Dictionary", "Magazine", "Comic"},
			Variants:     []string{"Hardcover", "Paperback", "Digital", "Audiobook"},
			PriceMin:     25, PriceMax: 500,
		},
		"Clothing": {
			NamePatterns: []string{"%s %s %s"},
			Brands:       []string{"Uniqlo", "H&M", "Zara", "Nike", "Adidas", "Puma", "Levi's", "Gap", "Old Navy"},
			Models:       []string{"T-Shirt", "Jeans", "Shirt", "Pants", "Jacket", "Sweater", "Dress", "Skirt", "Shorts"},
			Variants:     []string{"S", "M", "L", "XL", "XXL", "Cotton", "Polyester", "Wool"},
			PriceMin:     50, PriceMax: 1000,
		},
		"Home Appliances": {
			NamePatterns: []string{"%s %s %s"},
			Brands:       []string{"Samsung", "LG", "Panasonic", "Sharp", "Toshiba", "Electrolux", "Whirlpool", "Bosch"},
			Models:       []string{"Refrigerator", "Washing Machine", "Air Conditioner", "Microwave", "Blender", "Toaster", "Vacuum"},
			Variants:     []string{"1 Door", "2 Door", "Side-by-Side", "Top Load", "Front Load", "Split", "Window", "Countertop"},
			PriceMin:     500, PriceMax: 15000,
		},
		"Sports Equipment": {
			NamePatterns: []string{"%s %s %s"},
			Brands:       []string{"Nike", "Adidas", "Puma", "Reebok", "Under Armour", "New Balance", "Asics", "Wilson"},
			Models:       []string{"Running Shoes", "Basketball", "Football", "Tennis Racket", "Golf Club", "Dumbbells", "Yoga Mat"},
			Variants:     []string{"Men", "Women", "Kids", "Professional", "Beginner", "Lightweight", "Durable"},
			PriceMin:     100, PriceMax: 5000,
		},
		"Electronics": {
			NamePatterns: []string{"%s %s %s"},
			Brands:       []string{"Samsung", "Sony", "LG", "Panasonic", "JBL", "Bose", "Apple", "Google", "Amazon"},
			Models:       []string{"Wireless Earbuds", "Bluetooth Speaker", "Smart Watch", "Power Bank", "HDMI Cable", "USB Drive"},
			Variants:     []string{"Black", "White", "Wireless", "Fast Charge", "Premium", "Compact"},
			PriceMin:     100, PriceMax: 3000,
		},
		"Beauty": {
			NamePatterns: []string{"%s %s %s"},
			Brands:       []string{"L'Oréal", "Maybelline", "The Body Shop", "Nivea", "Garnier", "Ponds", "Vaseline", "Cetaphil"},
			Models:       []string{"Foundation", "Lipstick", "Moisturizer", "Shampoo", "Conditioner", "Body Lotion", "Face Mask"},
			Variants:     []string{"SPF 30", "Hydrating", "Matte", "Natural", "Anti-Aging", "Sensitive Skin"},
			PriceMin:     25, PriceMax: 300,
		},
		"Food": {
			NamePatterns: []string{"%s %s %s"},
			Brands:       []string{"Indofood", "ABC", "Mayora", "Ultra", "Wings", "Garudafood", "Unilever", "Nestlé"},
			Models:       []string{"Instant Noodles", "Biscuits", "Chocolate", "Tea", "Coffee", "Snack", "Candy", "Drink"},
			Variants:     []string{"Original", "Spicy", "Sweet", "Diet", "Low Sugar", "Premium", "Family Pack"},
			PriceMin:     5, PriceMax: 50,
		},
	}

	// Fallback template for categories without specific templates
	defaultTemplate = ProductTemplate{
		NamePatterns: []string{"%s %s %s"},
		Brands:       []string{"Generic", "Standard", "Quality", "Premium", "Value", "Pro", "Elite", "Basic"},
		Models:       []string{"Model A", "Model B", "Model C", "Standard", "Deluxe", "Professional", "Classic"},
		Variants:     []string{"Basic", "Standard", "Premium", "Deluxe", "Pro", "Lite"},
		PriceMin:     50, PriceMax: 2000,
	}
)

func main() {
	// Detect whether this is a daily-seeder invocation by inspecting
	// os.Args *before* any flag.Parse() call, so that -daily.* flags
	// never collide with the bulk seeder's known flags.
	runModeDaily := func() bool {
		for _, a := range os.Args[1:] {
			if a == "-daily" || strings.HasPrefix(a, "-daily.") {
				return true
			}
		}
		return false
	}()

	if runModeDaily {
		// Register daily-seed flags in the default FlagSet, then parse
		// the full argument list once.  Any unknown main-seeder flags
		// (e.g. -truncate, -products) are simply ignored.
		registerDailyFlags()
		flag.Parse()

		db, err := sql.Open("postgres", getDSN())
		if err != nil {
			log.Fatalf("Daily seeder — db open failed: %v", err)
		}
		defer db.Close()
		if err := db.Ping(); err != nil {
			log.Fatalf("Daily seeder — db ping failed: %v", err)
		}

		if err := RunDaily(db); err != nil {
			log.Fatalf("Daily seeder failed: %v", err)
		}
		return
	}

	// --- Full bulk seeder -----------------------------------------------
	truncateFlag := flag.Bool("truncate", true, "Truncate existing data before injection")
	productsFlag := flag.Int("products", 0, "Number of products to generate (random if 0)")
	daysFlag := flag.Int("days", 0, "Number of days to generate data for (150-180, random if 0)")
	categoriesFlag := flag.Int("categories", 0, "Number of categories to ensure exist (65-80, random if 0)")
	flag.Parse()

	if err := run(*truncateFlag, *productsFlag, *daysFlag, *categoriesFlag); err != nil {
		log.Fatalf("Dummy seeder failed: %v", err)
	}
}

func run(truncateData bool, numProducts, numDays, numCategories int) error {
	ctx := context.Background()


	// Validate parameters (relaxed for continuation)
	if numProducts < 0 || numProducts > 5000 {
		return fmt.Errorf("products count must be between 0-5000, got %d", numProducts)
	}
	if numCategories < 0 || numCategories > 80 {
		return fmt.Errorf("categories count must be between 0-80, got %d", numCategories)
	}

	// Interactive time range selection if not specified
	if numDays == 0 {
		numDays = promptTimeRange()
	}

	// Convert months to days (for validation, max 3 years = 36 months)
	if numDays < 180 || numDays > 1095 {
		return fmt.Errorf("days must be between 180-1095 (6 months - 3 years), got %d", numDays)
	}

	// Randomize counts if not specified (0 means random)
	if numProducts == 0 {
		numProducts = rand.Intn(501) + 4500 // 4500-5000
	}
	if numCategories == 0 {
		numCategories = rand.Intn(16) + 65 // 65-80
	}

	// Calculate date range
	endDate := time.Now().In(jakartaTZ)
	startDate := endDate.AddDate(0, 0, -numDays)

	// Connect to database
	db, err := sql.Open("postgres", getDSN())
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	fmt.Println("Connected to database. Starting comprehensive dummy data injection...")
	fmt.Printf("Target: %d products, %d days, %d categories\n", numProducts, numDays, numCategories)

	// 1. Truncate existing data if requested
	if truncateData {
		fmt.Println("🗑️  Truncating existing transactional data...")
		if err := truncateTransactionalData(ctx, db); err != nil {
			return fmt.Errorf("failed to truncate data: %w", err)
		}
		fmt.Println("✅ Data truncated successfully")
	}

	// 2. Ensure categories exist (skip if 0)
	var categoryIDs []int
	if numCategories > 0 {
		fmt.Printf("🔧 Ensuring %d categories exist...\n", numCategories)
		categoryIDs = ensureCategories(ctx, db, numCategories)
		if len(categoryIDs) == 0 {
			return fmt.Errorf("failed to create categories")
		}
		fmt.Printf("   ✅ %d categories ready\n", len(categoryIDs))
	} else {
		// Get existing categories
		categoryIDs = getIDs(ctx, db, "categories")
		fmt.Printf("   Found %d existing categories\n", len(categoryIDs))
	}

	// 3. Inject products
	var productData []ProductInfo
	if numProducts > 0 {
		fmt.Printf("📦 Injecting %d products...\n", numProducts)
		var err error
		productData, err = injectProducts(ctx, db, categoryIDs, numProducts)
		if err != nil {
			return fmt.Errorf("failed to inject products: %w", err)
		}
		fmt.Printf("   ✅ %d products injected\n", len(productData))
	} else {
		// Get existing products
		productData = getExistingProducts(ctx, db)
		fmt.Printf("   Found %d existing products\n", len(productData))
	}

	// 4. Get users for cashier assignment (needed for sales)
	var userIDs []int
	if len(productData) > 0 {
		userIDs = getIDs(ctx, db, "users")
		if len(userIDs) == 0 {
			return fmt.Errorf("no users found. Please run migrations/seeds first")
		}
	}

	// 5. Inject dummy customers (50-100) — must happen BEFORE sales so we can link them
	fmt.Printf("👥 Injecting dummy customers...\n")
	if err := injectCustomers(ctx, db, startDate, endDate); err != nil {
		return fmt.Errorf("failed to inject customers: %w", err)
	}
	fmt.Printf("   ✅ Customers injected\n")

	// 6. Load customer IDs for sales assignment
	customerIDs := getIDs(ctx, db, "customers")
	if len(customerIDs) == 0 {
		return fmt.Errorf("no customers found after injection")
	}

	// 7. Load walk-in customer ID
	var walkInCustomerID int
	err = db.QueryRowContext(ctx, "SELECT id FROM customers WHERE is_walk_in = true LIMIT 1").Scan(&walkInCustomerID)
	if err != nil {
		return fmt.Errorf("no walk-in customer found: %w", err)
	}

	// 8. Inject sales transactions (10-20 per day across all days)
	fmt.Printf("💰 Injecting daily sales (10-20 per day across %d days)...\n", numDays)

	if err := injectDailySales(ctx, db, userIDs, productData, customerIDs, walkInCustomerID, startDate, endDate); err != nil {
		return fmt.Errorf("failed to inject sales: %w", err)
	}

	fmt.Println("🎉 Dummy data injection completed successfully!")
	return nil
}

// ProductInfo holds product data for sales generation
type ProductInfo struct {
	ID         int
	Price      int
	Category   string
	TaxClassID *int
}

// truncateTransactionalData removes all business data while preserving admin data
func truncateTransactionalData(ctx context.Context, db *sql.DB) error {
	// Disable triggers temporarily to avoid FK constraint issues
	_, err := db.ExecContext(ctx, "SET session_replication_role = 'replica'")
	if err != nil {
		return fmt.Errorf("failed to disable triggers: %w", err)
	}
	defer func() {
		if _, err := db.ExecContext(ctx, "SET session_replication_role = 'origin'"); err != nil {
			log.Printf("failed to reset replication role: %v", err)
		}
	}()

	// Truncate tables in correct order (children first)
	tables := []string{
		"sale_items",
		"inventory_movements",
		"sales",
		"products",
		"customers",
	}

	for _, table := range tables {
		_, err := db.ExecContext(ctx, fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE", table))
		if err != nil {
			return fmt.Errorf("failed to truncate %s: %w", table, err)
		}
	}

	return nil
}

// getExistingProducts retrieves all existing products with their info
func getExistingProducts(ctx context.Context, db *sql.DB) []ProductInfo {
	rows, err := db.QueryContext(ctx, `
		SELECT p.id, p.price, c.name as category_name, p.tax_class_id
		FROM products p
		JOIN categories c ON p.category_id = c.id
		WHERE p.status = 'active'
		ORDER BY p.id
	`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var products []ProductInfo
	for rows.Next() {
		var p ProductInfo
		var taxClassID sql.NullInt64
		err := rows.Scan(&p.ID, &p.Price, &p.Category, &taxClassID)
		if err != nil {
			continue
		}
		if taxClassID.Valid {
			v := int(taxClassID.Int64)
			p.TaxClassID = &v
		}
		products = append(products, p)
	}
	return products
}

// ensureCategories creates categories up to the target count
func ensureCategories(ctx context.Context, db *sql.DB, targetCount int) []int {
	// First, check existing categories
	existingIDs := getIDs(ctx, db, "categories")

	// If we already have enough categories, return them
	if len(existingIDs) >= targetCount {
		return existingIDs[:targetCount]
	}

	// Generate additional categories if needed
	needed := targetCount - len(existingIDs)
	if needed > 0 && len(categories) > len(existingIDs) {
		fmt.Printf("   Creating %d additional categories...\n", needed)

		startIdx := len(existingIDs)
		endIdx := startIdx + needed
		if endIdx > len(categories) {
			endIdx = len(categories)
		}

		for i := startIdx; i < endIdx; i++ {
			catName := categories[i]
			var id int
			err := db.QueryRowContext(ctx,
				"INSERT INTO categories (name, description, is_active) VALUES ($1, $2, true) RETURNING id",
				catName, fmt.Sprintf("Auto-generated category for %s products", catName),
			).Scan(&id)

			if err != nil {
				fmt.Printf("Warning: failed to create category %s: %v\n", catName, err)
				continue
			}
			existingIDs = append(existingIDs, id)
		}
	}

	return existingIDs
}

// productWorkerJob represents a job for a worker in the concurrent product injection pool
type productWorkerJob struct {
	workerID       int
	startProduct   int
	endProduct     int
	categoryIDs    []int
	categoryNames  map[int]string
}

// injectProducts generates products using concurrent workers for better performance
func injectProducts(ctx context.Context, db *sql.DB, categoryIDs []int, count int) ([]ProductInfo, error) {
	const numWorkers = 4
	const batchSize = 100

	// Pre-fetch category names for product generation
	categoryNames := make(map[int]string)
	for _, catID := range categoryIDs {
		var name string
		err := db.QueryRowContext(ctx, "SELECT name FROM categories WHERE id = $1", catID).Scan(&name)
		if err == nil {
			categoryNames[catID] = name
		}
	}

	fmt.Printf("🚀 Starting %d workers to inject %d products with batch size %d\n", numWorkers, count, batchSize)

	// Create worker jobs
	jobs := make([]productWorkerJob, numWorkers)
	productsPerWorker := count / numWorkers
	remainingProducts := count % numWorkers

	currentStart := 0
	for i := 0; i < numWorkers; i++ {
		workerCount := productsPerWorker
		if i < remainingProducts {
			workerCount++
		}

		jobs[i] = productWorkerJob{
			workerID:      i,
			startProduct:  currentStart,
			endProduct:    currentStart + workerCount,
			categoryIDs:   categoryIDs,
			categoryNames: categoryNames,
		}

		currentStart += workerCount
	}

	// Start workers
	var wg sync.WaitGroup
	productChan := make(chan []ProductInfo, numWorkers)
	errorChan := make(chan error, numWorkers)

	for _, job := range jobs {
		wg.Add(1)
		go func(job productWorkerJob) {
			defer wg.Done()

			products, err := processProductWorkerJob(ctx, db, job, count, batchSize)
			if err != nil {
				errorChan <- err
				return
			}
			productChan <- products
		}(job)
	}

	// Wait for all workers to complete
	go func() {
		wg.Wait()
		close(productChan)
		close(errorChan)
	}()

	// Collect results
	var allProducts []ProductInfo
	completedWorkers := 0

	for completedWorkers < numWorkers {
		select {
		case products := <-productChan:
			allProducts = append(allProducts, products...)
			completedWorkers++
			if completedWorkers%500 == 0 {
				fmt.Printf("     ...%d products injected\n", len(allProducts))
			}
		case err := <-errorChan:
			return nil, err
		}
	}

	fmt.Printf("   ✅ %d products injected concurrently\n", len(allProducts))
	return allProducts, nil
}

// processProductWorkerJob handles a single worker's portion of product injection
func processProductWorkerJob(ctx context.Context, db *sql.DB, job productWorkerJob, totalCount, batchSize int) ([]ProductInfo, error) {
	products := make([]ProductInfo, 0, job.endProduct-job.startProduct)

	// Prepare batch insert
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			log.Printf("failed to rollback: %v", err)
		}
	}()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO products (sku, name, barcode, price, cost, stock, category_id, status, tax_class_id, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, 'active', 1, $8) RETURNING id`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	// Prepare product_stock INSERT (view v_products_full reads stock from product_stock)
	stockStmt, err := tx.PrepareContext(ctx,
		`INSERT INTO product_stock (product_id, quantity) VALUES ($1, $2)`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare stock statement: %w", err)
	}
	defer stockStmt.Close()

	batch := make([]ProductInfo, 0, batchSize)

	for i := job.startProduct; i < job.endProduct; i++ {
		catID := randElemInt(job.categoryIDs)
		catName := job.categoryNames[catID]

		// Get product template for this category
		template := getProductTemplate(catName)

		// Generate product data
		name := generateProductName(template)
		sku := generateSKU(catName, i)

		// Price in IDR (multiply by 1000 since template prices are in thousands)
		price := (rand.Intn(template.PriceMax-template.PriceMin+1) + template.PriceMin) * 1000
		cost := int(float64(price) * (0.4 + rand.Float64()*0.3)) // 40-70% of price

		// Generate barcode (6-13 characters)
		barcode := generateBarcode(i)

		stock := generateStockLevel(catName)

		// Random date within last 6 months (evenly distributed across the period)
		randomDays := rand.Intn(150) + 30 // 30-180 days ago (6 month span)
		createdAt := time.Now().In(jakartaTZ).AddDate(0, 0, -randomDays)

		var id int
		err := stmt.QueryRowContext(ctx, sku, name, barcode, price, cost, stock, catID, createdAt).Scan(&id)
		if err != nil {
			fmt.Printf("Warning: worker %d failed to insert product %d: %v\n", job.workerID, i, err)
			continue
		}

		// Insert into product_stock so v_products_full view returns correct stock
		if _, err := stockStmt.ExecContext(ctx, id, stock); err != nil {
			fmt.Printf("Warning: worker %d failed to insert product_stock for product %d: %v\n", job.workerID, i, err)
		}

		taxClassID := 1
		product := ProductInfo{
			ID:         id,
			Price:      price,
			Category:   catName,
			TaxClassID: &taxClassID,
		}

		batch = append(batch, product)

		// Flush batch when it reaches batchSize
		if len(batch) >= batchSize {
			products = append(products, batch...)
			batch = batch[:0] // Reset batch
		}
	}

	// Add remaining batch items
	if len(batch) > 0 {
		products = append(products, batch...)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return products, nil
}

// workerJob represents a job for a worker in the concurrent pool
type workerJob struct {
	workerID         int
	startInvoice     int
	days             []int // Days this worker handles (relative to startDate)
	customerIDs      []int // Available customer IDs to assign to sales
	walkInCustomerID int   // Walk-in/general customer ID for 30-50% of sales
}

// injectDailySales generates transactions ensuring every day has at least 10 transactions using concurrent workers
func injectDailySales(ctx context.Context, db *sql.DB, userIDs []int, products []ProductInfo, customerIDs []int, walkInCustomerID int, startDate, endDate time.Time) error {
	numWorkers := 4 // Concurrent workers for performance

	now := time.Now().In(jakartaTZ)
	productMap := make(map[int]ProductInfo)
	for _, p := range products {
		productMap[p.ID] = p
	}

	totalDays := int(endDate.Sub(startDate).Hours() / 24)
	if totalDays < 0 {
		return fmt.Errorf("invalid date range: end date must be after start date")
	}

	// Include the end date as well (total days inclusive)
	totalDaysInclusive := totalDays + 1

	// Calculate conservative invoice ranges with generous buffer
	avgSalesPerDay := 15 // Average 10-20 transactions per day
	estimatedTotalSales := totalDaysInclusive * avgSalesPerDay
	estimatedTotalSales = int(float64(estimatedTotalSales) * 1.5) // 50% buffer for safety

	invoicesPerWorker := estimatedTotalSales / numWorkers
	if estimatedTotalSales%numWorkers != 0 {
		invoicesPerWorker++ // Round up to ensure coverage
	}

	jobs := make([]workerJob, numWorkers)

	// Distribute days evenly among workers
	if numWorkers <= 0 {
		return fmt.Errorf("invalid number of workers: %d", numWorkers)
	}
	daysPerWorker := totalDaysInclusive / numWorkers
	remainingDays := totalDaysInclusive % numWorkers

	currentDay := 0
	currentInvoice := 1
	for i := 0; i < numWorkers; i++ {
		job := workerJob{
			workerID:         i,
			startInvoice:     currentInvoice,
			days:             []int{},
			customerIDs:      customerIDs,
			walkInCustomerID: walkInCustomerID,
		}

		// Update invoice counter for next worker
		currentInvoice += invoicesPerWorker

		// Calculate how many days this worker gets
		workerDays := daysPerWorker
		if i < remainingDays {
			workerDays++ // Extra day for first 'remainingDays' workers
		}

		// Assign consecutive days to this worker
		for d := 0; d < workerDays; d++ {
			if currentDay < totalDaysInclusive {
				job.days = append(job.days, currentDay)
				currentDay++
			}
		}

		jobs[i] = job
	}

	fmt.Printf("🚀 Starting %d workers to process %d days with estimated %d total sales\n", numWorkers, totalDaysInclusive, estimatedTotalSales)
	fmt.Printf("   Each worker allocated up to %d invoice numbers\n", invoicesPerWorker)

	// Start workers
	var wg sync.WaitGroup
	salesCreated := make(chan int, numWorkers*1000) // Buffered channel for progress updates

	for _, job := range jobs {
		wg.Add(1)
		go func(job workerJob) {
			defer wg.Done()

			workerSales := processWorkerJob(ctx, db, job, userIDs, products, productMap, startDate, now, salesCreated)
			fmt.Printf("   ✅ Worker %d completed: %d sales\n", job.workerID, workerSales)
		}(job)
	}

	// Progress monitoring
	go func() {
		total := 0
		for count := range salesCreated {
			total += count
			if total%100 == 0 {
				fmt.Printf("     ...%d sales injected\n", total)
			}
		}
	}()

	wg.Wait()
	close(salesCreated)

	fmt.Printf("   ✅ All sales transactions injected across %d days (min 10 per day)\n", totalDaysInclusive)
	return nil
}

// SaleBatch represents a batch of sales data for efficient insertion
type SaleBatch struct {
	Sales     []SaleRecord
	SaleItems []SaleItemRecord
}

type SaleRecord struct {
	Invoice      string
	CashierID    int
	PaymentMethod string
	CreatedAt    time.Time
	TotalAmount  int
}

type SaleItemRecord struct {
	SaleID      int
	ProductID   int
	Quantity    int
	UnitPrice   int
	Subtotal    int
}

// processWorkerJob handles a single worker's portion of the work with optimized batch transactions
func processWorkerJob(ctx context.Context, db *sql.DB, job workerJob, userIDs []int, products []ProductInfo, productMap map[int]ProductInfo, startDate, now time.Time, progress chan<- int) int {
	invoiceCounter := 0

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0
	}

	saleStmt, err := tx.PrepareContext(ctx,
		`INSERT INTO sales (invoice_number, cashier_id, customer_id, payment_method, status, subtotal, tax, total_amount, created_at)
		 VALUES ($1, $2, $3, $4, 'completed', $5, $6, $7, $8) RETURNING id`)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("failed to rollback: %v", rbErr)
		}
		return 0
	}
	defer saleStmt.Close()

	itemStmt, err := tx.PrepareContext(ctx,
		`INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal, dpp_amount, tax_amount) VALUES ($1, $2, $3, $4, $5, $6, $7)`)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("failed to rollback: %v", rbErr)
		}
		return 0
	}
	defer itemStmt.Close()

	salesCreated := 0
	batchSize := 0

	for _, dayOffset := range job.days {
		dayDate := startDate.AddDate(0, 0, dayOffset)

		salesForDay := 10 + rand.Intn(11)

		for s := 0; s < salesForDay; s++ {
			invoiceNum := job.startInvoice + invoiceCounter
			invoice := fmt.Sprintf("INV-%d-%06d", now.Year(), invoiceNum)
			invoiceCounter++

			cashierID := randElemInt(userIDs)
			// 30-50% of sales use the walk-in/general customer
			customerID := randElemInt(job.customerIDs)
			if rand.Intn(100) < 40 {
				customerID = job.walkInCustomerID
			}
			paymentMethod := weightedRandomChoice(paymentMethods, paymentWeights)
			createdAt := randomTime24Hour(dayDate, now)

			numItems := generateItemCount()
			saleProducts := selectProductsForSale(products, numItems)

			// Calculate total and build items with DPP/tax
			const defaultRate = 11.0
			var totalAmount, totalDPP, totalTax int
			items := make([]SaleItemRecord, 0, numItems)
			for _, productID := range saleProducts {
				product := productMap[productID]
				quantity := generateQuantity(product.Category)
				unitPrice := product.Price
				subtotal := unitPrice * quantity
				totalAmount += subtotal

				var dpp, tax int
				if product.TaxClassID != nil {
					dpp = int(math.Round(float64(subtotal) * 100.0 / (100.0 + defaultRate)))
					tax = subtotal - dpp
				} else {
					dpp = subtotal
					tax = 0
				}
				totalDPP += dpp
				totalTax += tax

				items = append(items, SaleItemRecord{
					SaleID:    0, // Will be filled after sale insert
					ProductID: productID,
					Quantity:  quantity,
					UnitPrice: unitPrice,
					Subtotal:  subtotal,
				})
			}

			if totalAmount <= 0 {
				continue
			}

			// Insert sale
			var saleID int
			err := saleStmt.QueryRowContext(ctx, invoice, cashierID, customerID, paymentMethod, totalDPP, totalTax, totalAmount, createdAt).Scan(&saleID)
			if err != nil {
				continue
			}

			// Insert sale items
			for _, item := range items {
				product := productMap[item.ProductID]
				var dpp, tax int
				if product.TaxClassID != nil {
					dpp = int(math.Round(float64(item.Subtotal) * 100.0 / (100.0 + defaultRate)))
					tax = item.Subtotal - dpp
				} else {
					dpp = item.Subtotal
					tax = 0
				}
				if _, err := itemStmt.ExecContext(ctx, saleID, item.ProductID, item.Quantity, item.UnitPrice, item.Subtotal, dpp, tax); err != nil {
					log.Printf("failed to insert sale item: %v", err)
				}
			}

			salesCreated++
			batchSize++

			if batchSize >= 50 {
				saleStmt.Close()
				itemStmt.Close()
				if err := tx.Commit(); err != nil {
					log.Printf("failed to commit batch: %v", err)
				}

				tx, _ = db.BeginTx(ctx, nil)
				saleStmt, _ = tx.PrepareContext(ctx,
					`INSERT INTO sales (invoice_number, cashier_id, customer_id, payment_method, status, subtotal, tax, total_amount, created_at)
					 VALUES ($1, $2, $3, $4, 'completed', $5, $6, $7, $8) RETURNING id`)
				itemStmt, _ = tx.PrepareContext(ctx,
					`INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal, dpp_amount, tax_amount) VALUES ($1, $2, $3, $4, $5, $6, $7)`)

				batchSize = 0
			}

			if salesCreated%25 == 0 {
				select {
				case progress <- 25:
				default:
				}
			}
		}
	}

	saleStmt.Close()
	itemStmt.Close()
	if err := tx.Commit(); err != nil {
		log.Printf("failed to commit final batch: %v", err)
	}

	select {
	case progress <- (salesCreated % 25):
	default:
	}

	return salesCreated
}

// randomTime24Hour generates random time spread across 24 hours for 24/7 open store
func randomTime24Hour(dayDate, now time.Time) time.Time {
	isToday := dayDate.Year() == now.Year() && dayDate.Month() == now.Month() && dayDate.Day() == now.Day()

	// Weighted hour distribution for 24/7 store
	// Higher weight for typical business hours (09-21), but still allow all hours
	hourWeights := make([]int, 24)
	for h := 0; h < 24; h++ {
		switch {
		case h >= 9 && h < 21:
			hourWeights[h] = 15 // Peak business hours
		case h >= 7 && h < 23:
			hourWeights[h] = 12 // Near peak
		default:
			hourWeights[h] = 8 // Off hours (still open)
		}
	}

	hour := weightedPickInt(hourWeights)
	var minute int
	if isToday {
		currentHour := now.In(jakartaTZ).Hour()
		currentMinute := now.Minute()
		if hour > currentHour || (hour == currentHour && currentMinute < 5) {
			hour = currentHour
		}
		maxMinute := 59
		if hour == currentHour {
			maxMinute = currentMinute
		}
		minute = rand.Intn(maxMinute + 1)
	} else {
		minute = rand.Intn(60)
	}

	return time.Date(dayDate.Year(), dayDate.Month(), dayDate.Day(),
		hour, minute, rand.Intn(60), 0, jakartaTZ)
}

// Helper functions for realistic data generation

func getProductTemplate(category string) ProductTemplate {
	categoryLower := strings.ToLower(category)

	// Direct match first
	if template, exists := productTemplates[category]; exists {
		return template
	}

	// Exact category name matching for database categories
	switch category {
	case "Smartphones", "Tablets", "Mobile Accessories":
		return productTemplates["Smartphones"]
	case "Laptops", "Computer Components", "Computer Peripherals":
		return productTemplates["Laptops"]
	case "Books & Media", "Books":
		return productTemplates["Books"]
	case "Men's Clothing", "Women's Clothing", "Kids' Clothing", "Clothing", "Footwear":
		return productTemplates["Clothing"]
	case "Kitchen Appliances", "Home Appliances":
		return productTemplates["Home Appliances"]
	case "Sports & Outdoors", "Fitness Equipment", "Camping & Hiking":
		return productTemplates["Sports Equipment"]
	case "Groceries", "Snacks", "Beverages", "Bakery", "Dairy Products", "Fresh Produce", "Frozen Foods", "Canned Goods", "Condiments":
		return productTemplates["Food"]
	case "Furniture", "Home Decor", "Bedding", "Decor", "Lighting":
		return productTemplates["Furniture"]
	case "Skincare", "Hair Care", "Makeup", "Personal Care", "Beauty":
		return productTemplates["Beauty"]
	case "Electronics", "Audio Equipment", "Cameras & Photography", "Gaming", "Smart Home Devices", "Networking", "Wearable Tech":
		return productTemplates["Electronics"]
	}

	// Fallback pattern matching
	switch {
	case strings.Contains(categoryLower, "phone") || strings.Contains(categoryLower, "mobile"):
		return productTemplates["Smartphones"]
	case strings.Contains(categoryLower, "laptop") || strings.Contains(categoryLower, "computer"):
		return productTemplates["Laptops"]
	case strings.Contains(categoryLower, "book") || strings.Contains(categoryLower, "media"):
		return productTemplates["Books"]
	case strings.Contains(categoryLower, "cloth") || strings.Contains(categoryLower, "fashion") || strings.Contains(categoryLower, "wear"):
		return productTemplates["Clothing"]
	case strings.Contains(categoryLower, "appliance") || strings.Contains(categoryLower, "kitchen"):
		return productTemplates["Home Appliances"]
	case strings.Contains(categoryLower, "sport") || strings.Contains(categoryLower, "fitness") || strings.Contains(categoryLower, "outdoor"):
		return productTemplates["Sports Equipment"]
	case strings.Contains(categoryLower, "food") || strings.Contains(categoryLower, "grocery") || strings.Contains(categoryLower, "beverage"):
		return productTemplates["Food"]
	case strings.Contains(categoryLower, "furniture") || strings.Contains(categoryLower, "home"):
		return productTemplates["Furniture"]
	case strings.Contains(categoryLower, "beauty") || strings.Contains(categoryLower, "cosmetic") || strings.Contains(categoryLower, "skincare") || strings.Contains(categoryLower, "makeup") || strings.Contains(categoryLower, "hair"):
		return productTemplates["Beauty"]
	case strings.Contains(categoryLower, "electronic") || strings.Contains(categoryLower, "gadget") || strings.Contains(categoryLower, "device"):
		return productTemplates["Electronics"]
	default:
		return defaultTemplate
	}
}

func generateProductName(template ProductTemplate) string {
	pattern := randElem(template.NamePatterns)
	brand := randElem(template.Brands)
	model := randElem(template.Models)
	variant := randElem(template.Variants)

	// Count placeholders in pattern
	placeholderCount := strings.Count(pattern, "%s")

	switch placeholderCount {
	case 1:
		return fmt.Sprintf(pattern, brand)
	case 2:
		return fmt.Sprintf(pattern, brand, model)
	case 3:
		return fmt.Sprintf(pattern, brand, model, variant)
	case 4:
		// For patterns with 4 placeholders (like smartphones with storage)
		storage := randElem([]string{"64", "128", "256", "512", "1024"})
		return fmt.Sprintf(pattern, brand, model, storage, variant)
	default:
		// Fallback for any other pattern
		return fmt.Sprintf("%s %s %s", brand, model, variant)
	}
}

func generateSKU(category string, index int) string {
	// Format: SKU-XXXXX where XXXXX is sequential starting from 00001
	return fmt.Sprintf("SKU-%05d", index+1)
}

func generateBarcode(index int) string {
	// Generate valid EAN-13 barcode (13 digits with proper check digit)

	// Generate first 12 random digits
	digits := make([]int, 12)
	for i := 0; i < 12; i++ {
		digits[i] = rand.Intn(10)
	}

	// Calculate check digit (EAN-13 algorithm)
	sum := 0
	for i := 0; i < 12; i++ {
		if (i % 2) == 0 {
			sum += digits[i] * 3  // odd positions (1-based): i=0->pos1, i=2->pos3, etc.
		} else {
			sum += digits[i] * 1  // even positions (1-based): i=1->pos2, i=3->pos4, etc.
		}
	}
	checkDigit := (10 - (sum % 10)) % 10

	// Build the 13-digit barcode string
	barcode := ""
	for _, d := range digits {
		barcode += fmt.Sprintf("%d", d)
	}
	barcode += fmt.Sprintf("%d", checkDigit)

	return barcode
}

func generateStockLevel(category string) int {
	// Different stock levels based on category
	switch category {
	case "Smartphones", "Laptops", "Cameras":
		return rand.Intn(50) + 10 // 10-60 units (high-value items)
	case "Groceries", "Snacks", "Beverages":
		return rand.Intn(200) + 50 // 50-250 units (consumables)
	case "Books", "Magazines":
		return rand.Intn(100) + 20 // 20-120 units (information goods)
	case "Furniture", "Kitchen Appliances":
		return rand.Intn(20) + 5 // 5-25 units (large items)
	default:
		return rand.Intn(150) + 25 // 25-175 units (general)
	}
}

func weightedRandomChoice(items []string, weights []int) string {
	total := 0
	for _, w := range weights {
		total += w
	}

	r := rand.Intn(total)
	cumulative := 0

	for i, item := range items {
		cumulative += weights[i]
		if r < cumulative {
			return item
		}
	}

	return items[0] // fallback
}

func generateItemCount() int {
	// Realistic distribution: most sales have 1-3 items, some have more
	r := rand.Intn(100)
	if r < 30 {
		return 1
	} else if r < 60 {
		return 2
	} else if r < 80 {
		return 3
	} else if r < 90 {
		return 4
	} else if r < 95 {
		return 5
	} else {
		return rand.Intn(4) + 5 // 5-8 items for large transactions
	}
}

func selectProductsForSale(products []ProductInfo, count int) []int {
	selected := make([]int, 0, count)

	// Simple random selection (can be improved later)
	for i := 0; i < count; i++ {
		selectedProduct := products[rand.Intn(len(products))]
		selected = append(selected, selectedProduct.ID)
	}

	return selected
}

func generateQuantity(category string) int {
	// Quantity depends on product type
	switch category {
	case "Groceries", "Snacks", "Beverages":
		// Consumables often bought in multiples
		if rand.Intn(100) < 70 {
			return rand.Intn(3) + 1 // 1-3
		} else {
			return rand.Intn(5) + 4 // 4-8 for bulk
		}
	case "Books", "Magazines":
		return rand.Intn(3) + 1 // 1-3
	case "Smartphones", "Laptops", "Cameras", "Furniture":
		return 1 // Usually 1 of high-value items
	default:
		return rand.Intn(2) + 1 // 1-2 for most items
	}
}

// Helper functions

func getIDs(ctx context.Context, db *sql.DB, table string) []int {
	rows, err := db.QueryContext(ctx, fmt.Sprintf("SELECT id FROM %s ORDER BY id LIMIT 200", table))
	if err != nil {
		return nil
	}
	defer rows.Close()
	ids := []int{}
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			log.Printf("failed to scan row: %v", err)
			continue
		}
		ids = append(ids, id)
	}
	return ids
}

func randElem(s []string) string {
	if len(s) == 0 {
		return ""
	}
	return s[rand.Intn(len(s))]
}

func randElemInt(s []int) int {
	if len(s) == 0 {
		return 0
	}
	return s[rand.Intn(len(s))]
}

// promptTimeRange interactively asks user for time range selection
func promptTimeRange() int {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("\n📅 Transaction Time Range Selection")
	fmt.Println("---------------------------------")
	fmt.Println("How far back should the generated data span?")
	fmt.Println("  1. 6 months (180 days)")
	fmt.Println("  2. 1 year (365 days)")
	fmt.Println("  3. 2 years (730 days)")
	fmt.Println("  4. 3 years (1095 days)")
	fmt.Println("  5. Custom (specify months)")

	var choice int
	for {
		fmt.Print("\nEnter your choice (1-5): ")
		_, err := fmt.Fscan(reader, &choice)
		if err == nil && choice >= 1 && choice <= 5 {
			break
		}
		if err != nil {
			_, _ = reader.ReadString('\n') // clear buffer
		}
		fmt.Println("Invalid choice. Please enter 1-5.")
	}

	switch choice {
	case 1:
		return 180
	case 2:
		return 365
	case 3:
		return 730
	case 4:
		return 1095
	case 5:
		var months int
		for {
			fmt.Print("Enter number of months (6-36): ")
			_, err := fmt.Fscan(reader, &months)
			if err == nil && months >= 6 && months <= 36 {
				return months * 30 // approximate days
			}
			if err != nil {
				_, _ = reader.ReadString('\n')
			}
			fmt.Println("Invalid input. Please enter a number between 6 and 36.")
		}
	}
	return 180 // fallback
}

func getDSN() string {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		// Default local DB -- sesuaikan dengan docker compose jika berbeda
		dsn = "postgres://pos:admin123@localhost:5433/retail_pos?sslmode=disable&timezone=Asia/Jakarta"
	}
	return dsn
}
// ==================== CUSTOMER GENERATION ====================

var (
	customerFirstNames = []string{
		"Ahmad", "Siti", "Budi", "Dewi", "Eko", "Rina", "Hendra", "Maya", "Rizky", "Ani",
		"Joko", "Lestari", "Agus", "Wati", "Bambang", "Sri", "Dedi", "Nur", "Fajar", "Fitri",
		"Gunawan", "Hasan", "Irfan", "Joko", "Kartika", "Lukman", "Mira", "Nanda", "Oscar", "Putri",
		"Reza", "Susi", "Tri", "Umi", "Vina", "Wahyu", "Yani", "Zulfikri", "Ayu", "Bayu",
		"Citra", "Dimas", "Erna", "Farhan", "Gita", "Hanan", "Indra", "Jihan", "Kevin", "Lina",
	}
	customerLastNames = []string{
		"Santoso", "Wijaya", "Kusuma", "Pratama", "Sari", "Hidayat", "Nugroho", "Permata",
		"Saputra", "Maulana", "Hakim", "Gunawan", "Siregar", "Simanjuntak", "Nainggolan",
		"Lubis", "Pane", "Rajagukguk", "Sinaga", "Tobing", "Umar", "Utomo", "Wibowo",
		"Yulianto", "Zulkifli", "Abdullah", "Bachtiar", "Chandra", "Damanik", "Effendi",
		"Firmansyah", "Girsang", "Halim", "Ibrahim", "Junaedi", "Kurniawan", "Lestari",
		"Mansyur", "Nasution", "Oktavian", "Purba", "Quari", "Rahman", "Setiawan", "Thamrin",
	}
)

func injectCustomers(ctx context.Context, db *sql.DB, startDate, endDate time.Time) error {
	numCustomers := 50 + rand.Intn(51) // 50-100
	fmt.Printf("   🎲 Generating %d customers\n", numCustomers)

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			log.Printf("failed to rollback: %v", err)
		}
	}()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO customers (name, phone, email, is_active, is_walk_in, created_at)
		 VALUES ($1, $2, $3, true, false, $4)`)
	if err != nil {
		return fmt.Errorf("prepare: %w", err)
	}
	defer stmt.Close()

	now := time.Now().In(jakartaTZ)

	// Insert a walk-in/general customer first
	walkInID := 0
	err = tx.QueryRowContext(ctx,
		`INSERT INTO customers (name, phone, email, is_active, is_walk_in, created_at)
		 VALUES ('Walk-in / General', '', '', true, true, $1)
		 RETURNING id`,
		now,
	).Scan(&walkInID)
	if err != nil {
		return fmt.Errorf("insert walk-in customer: %w", err)
	}

	for i := 0; i < numCustomers; i++ {
		first := customerFirstNames[rand.Intn(len(customerFirstNames))]
		last := customerLastNames[rand.Intn(len(customerLastNames))]
		name := fmt.Sprintf("%s %s", first, last)
		phone := fmt.Sprintf("08%s", fmt.Sprintf("%010d", rand.Intn(10000000000)))
		email := fmt.Sprintf("%s.%s@email.com", strings.ToLower(first), strings.ToLower(last))

		daysAgo := rand.Intn(int(now.Sub(startDate).Hours()/24)) + 1
		createdAt := now.AddDate(0, 0, -daysAgo)

		if _, err := stmt.ExecContext(ctx, name, phone, email, createdAt); err != nil {
			fmt.Printf("   ⚠️  Skipped customer %s: %v\n", name, err)
			continue
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit customers: %w", err)
	}
	_ = walkInID // returned for reference
	return nil
}
