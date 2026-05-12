package main

import (
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
	// Payment methods with realistic distribution
	paymentMethods = []string{"Cash", "QRIS", "Debit", "Credit"}
	paymentWeights = []int{40, 35, 15, 10} // Percentages

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
	// Parse command line flags (values are randomized if not specified)
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
	rand.Seed(time.Now().UnixNano())

	// Validate parameters (relaxed for continuation)
	if numProducts < 0 || numProducts > 5000 {
		return fmt.Errorf("products count must be between 0-5000, got %d", numProducts)
	}
	if numDays < 0 || numDays > 180 {
		return fmt.Errorf("days count must be between 0-180, got %d", numDays)
	}
	if numCategories < 0 || numCategories > 80 {
		return fmt.Errorf("categories count must be between 0-80, got %d", numCategories)
	}

	// Randomize counts if not specified (0 means random)
	if numProducts == 0 {
		numProducts = rand.Intn(501) + 4500 // 4500-5000
	}
	if numDays == 0 {
		numDays = rand.Intn(31) + 150 // 150-180 days
	}
	if numCategories == 0 {
		numCategories = rand.Intn(16) + 65 // 65-80
	}

	// Calculate date range
	endDate := time.Now()
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

	// 5. Inject sales transactions (10-20 per day across all days)
	fmt.Printf("💰 Injecting daily sales (10-20 per day across %d days)...\n", numDays)

	if err := injectDailySales(ctx, db, userIDs, productData, startDate, endDate); err != nil {
		return fmt.Errorf("failed to inject sales: %w", err)
	}

	fmt.Println("🎉 Dummy data injection completed successfully!")
	return nil
}

// ProductInfo holds product data for sales generation
type ProductInfo struct {
	ID       int
	Price    int
	Category string
}

// truncateTransactionalData removes all business data while preserving admin data
func truncateTransactionalData(ctx context.Context, db *sql.DB) error {
	// Disable triggers temporarily to avoid FK constraint issues
	_, err := db.ExecContext(ctx, "SET session_replication_role = 'replica'")
	if err != nil {
		return fmt.Errorf("failed to disable triggers: %w", err)
	}
	defer db.ExecContext(ctx, "SET session_replication_role = 'origin'")

	// Truncate tables in correct order (children first)
	tables := []string{
		"sale_items",
		"inventory_movements",
		"sales",
		"products",
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
		SELECT p.id, p.price, c.name as category_name
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
		err := rows.Scan(&p.ID, &p.Price, &p.Category)
		if err != nil {
			continue
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
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO products (sku, name, barcode, price, cost, stock, stock_min, category_id, status, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'active', $9) RETURNING id`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

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

		// Stock levels - ensure 10-15% have low stock
		stockMin := 5 // Default stock minimum
		var stock int
		lowStockPercentage := 12 // 12% of products will have low stock
		if i < totalCount*lowStockPercentage/100 {
			// Low stock: 0 to stock_min
			stock = rand.Intn(stockMin + 1)
		} else {
			// Normal stock distribution
			stock = generateStockLevel(catName)
		}

		// Random date within last 6 months (evenly distributed across the period)
		randomDays := rand.Intn(150) + 30 // 30-180 days ago (6 month span)
		createdAt := time.Now().AddDate(0, 0, -randomDays)

		var id int
		err := stmt.QueryRowContext(ctx, sku, name, barcode, price, cost, stock, stockMin, catID, createdAt).Scan(&id)
		if err != nil {
			fmt.Printf("Warning: worker %d failed to insert product %d: %v\n", job.workerID, i, err)
			continue
		}

		product := ProductInfo{
			ID:       id,
			Price:    price,
			Category: catName,
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
	workerID     int
	startInvoice int
	days         []int // Days this worker handles (relative to startDate)
}

// injectDailySales generates transactions ensuring every day has at least 10 transactions using concurrent workers
func injectDailySales(ctx context.Context, db *sql.DB, userIDs []int, products []ProductInfo, startDate, endDate time.Time) error {
	numWorkers := 4 // Concurrent workers for performance

	now := time.Now()
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
			workerID:     i,
			startInvoice: currentInvoice,
			days:         []int{},
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

// processWorkerJob handles a single worker's portion of the work with optimized individual transactions
func processWorkerJob(ctx context.Context, db *sql.DB, job workerJob, userIDs []int, products []ProductInfo, productMap map[int]ProductInfo, startDate, now time.Time, progress chan<- int) int {
	salesCreated := 0
	invoiceCounter := 0 // Local counter for this worker

	// Process each day
	for _, dayOffset := range job.days {
		dayDate := startDate.AddDate(0, 0, dayOffset)

		// 10-20 transactions per day
		salesForDay := 10 + rand.Intn(11)

		for s := 0; s < salesForDay; s++ {
			// Generate sequential invoice number for this worker's range
			invoiceNum := job.startInvoice + invoiceCounter
			invoice := fmt.Sprintf("INV-%d-%06d", now.Year(), invoiceNum)
			invoiceCounter++

			// Random cashier
			cashierID := randElemInt(userIDs)

			// Random payment method (weighted)
			paymentMethod := weightedRandomChoice(paymentMethods, paymentWeights)

			var createdAt time.Time

			// Check if this is today
			isToday := dayDate.Year() == now.Year() && dayDate.Month() == now.Month() && dayDate.Day() == now.Day()

			if isToday {
				// Today: generate time from 6 hours ago until 5 minutes ago to ensure it's in the past
				currentNow := time.Now()
				sixHoursAgo := currentNow.Add(-6 * time.Hour)
				fiveMinutesAgo := currentNow.Add(-5 * time.Minute)

				// Generate random time between 6 hours ago and 5 minutes ago
				timeRange := fiveMinutesAgo.Sub(sixHoursAgo)
				if timeRange <= 0 {
					timeRange = time.Hour // fallback to 1 hour range
				}
				randomOffset := time.Duration(rand.Int63n(int64(timeRange)))
				createdAt = sixHoursAgo.Add(randomOffset)
			} else {
				// Other days: random hour between 8 AM and 8 PM
				randomHour := 8 + rand.Intn(12)
				randomMinute := rand.Intn(60)
				createdAt = time.Date(dayDate.Year(), dayDate.Month(), dayDate.Day(),
					randomHour, randomMinute, rand.Intn(60), 0, time.UTC)
			}

			// Process sale with individual transaction for reliability
			if err := processIndividualSale(ctx, db, invoice, cashierID, paymentMethod, createdAt, products, productMap); err == nil {
				salesCreated++

				// Send progress update every 25 sales
				if salesCreated%25 == 0 {
					select {
					case progress <- 25:
					default:
						// Channel full, skip progress update
					}
				}
			}
		}
	}

	// Send final count to progress channel
	select {
	case progress <- (salesCreated % 25): // Send remaining count
	default:
		// Channel full, skip progress update
	}

	return salesCreated
}

// processIndividualSale handles one sale with optimized transaction
func processIndividualSale(ctx context.Context, db *sql.DB, invoice string, cashierID int, paymentMethod string, createdAt time.Time, products []ProductInfo, productMap map[int]ProductInfo) error {
	// Start individual transaction for this sale
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Generate sale items first to calculate total
	numItems := generateItemCount()
	totalAmount := 0
	saleItemRecords := make([]SaleItemRecord, 0, numItems)

	// Select products for this sale
	saleProducts := selectProductsForSale(products, numItems)

	for _, productID := range saleProducts {
		product := productMap[productID]

		// Generate quantity (depends on product type)
		quantity := generateQuantity(product.Category)

		// Use current product price as snapshot (transaction-time pricing)
		unitPrice := product.Price
		subtotal := unitPrice * quantity
		totalAmount += subtotal

		saleItemRecords = append(saleItemRecords, SaleItemRecord{
			ProductID: productID,
			Quantity:  quantity,
			UnitPrice: unitPrice,
			Subtotal:  subtotal,
		})
	}

	// Skip sale if no valid items
	if totalAmount <= 0 || len(saleItemRecords) == 0 {
		return fmt.Errorf("no valid items for sale")
	}

	// Insert sale record with pre-calculated total
	var saleID int
	err = tx.QueryRowContext(ctx,
		`INSERT INTO sales (invoice_number, cashier_id, payment_method, status, subtotal, total_amount, created_at)
		 VALUES ($1, $2, $3, 'completed', $4, $4, $5) RETURNING id`,
		invoice, cashierID, paymentMethod, totalAmount, createdAt,
	).Scan(&saleID)
	if err != nil {
		return fmt.Errorf("failed to insert sale: %w", err)
	}

	// Insert sale items in batch within the same transaction
	for _, item := range saleItemRecords {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal)
			 VALUES ($1, $2, $3, $4, $5)`,
			saleID, item.ProductID, item.Quantity, item.UnitPrice, item.Subtotal,
		)
		if err != nil {
			return fmt.Errorf("failed to insert sale item: %w", err)
		}
	}

	// Commit the transaction
	return tx.Commit()
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

func generateWeightedDate(startDate time.Time, days int) time.Time {
	// Use exponential distribution to favor more recent dates
	lambda := 3.0 // Controls the weighting (higher = more recent bias)
	u := rand.Float64()
	dayOffset := int(-math.Log(1-u)/lambda*float64(days)) % days
	return startDate.AddDate(0, 0, dayOffset)
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
		rows.Scan(&id)
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func getDSN() string {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		// Default local DB -- sesuaikan dengan docker compose jika berbeda
		dsn = "postgres://pos:admin123@localhost:5432/retail_pos?sslmode=disable&timezone=Asia/Jakarta"
	}
	return dsn
}