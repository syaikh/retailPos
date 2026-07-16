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
	"sort"
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
	daysFlag := flag.Int("days", 0, "Number of days to generate data for (0 = interactive prompt)")
	categoriesFlag := flag.Int("categories", 0, "Number of categories to ensure exist (random if 0)")
	flag.Parse()

	if err := run(*truncateFlag, *productsFlag, *daysFlag, *categoriesFlag); err != nil {
		log.Fatalf("Dummy seeder failed: %v", err)
	}
}

func run(truncateData bool, numProducts, numDays, numCategories int) error {
	ctx := context.Background()

	// Validate parameters
	if numProducts < 0 {
		return fmt.Errorf("products count must not be negative, got %d", numProducts)
	}
	if numCategories < 0 {
		return fmt.Errorf("categories count must not be negative, got %d", numCategories)
	}

	// Interactive time range selection if not specified
	if numDays == 0 {
		numDays = promptTimeRange()
	}

	if numDays < 0 {
		return fmt.Errorf("days count must not be negative, got %d", numDays)
	}

	// Randomize counts if not specified (0 means random)
	if numProducts == 0 {
		numProducts = rand.Intn(1001) + 4500 // 4500-5500
	}
	if numCategories == 0 {
		numCategories = rand.Intn(36) + 65 // 65-100
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
		if err := truncateAllData(ctx, db); err != nil {
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

	// 3. Ensure tax classes exist
	ensureTaxClasses(ctx, db)
	fmt.Println("   ✅ Tax classes ready")

	// 3b. Ensure brands exist
	ensureBrands(ctx, db)
	fmt.Println("   ✅ Brands ready")

	// 3c. Ensure units of measure exist
	ensureUnitsOfMeasure(ctx, db)
	fmt.Println("   ✅ Units of measure ready")

	// 3d. Ensure payment methods exist
	ensurePaymentMethods(ctx, db)
	fmt.Println("   ✅ Payment methods ready")

	// 3e. Clean up test/dummy roles
	cleanupTestRoles(ctx, db)
	fmt.Println("   ✅ Test/dummy roles cleaned up")

	// 4. Inject products
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

	// 5. Get users for cashier assignment (needed for sales)
	var userIDs []int
	if len(productData) > 0 {
		userIDs = getIDs(ctx, db, "users")
		if len(userIDs) == 0 {
			return fmt.Errorf("no users found. Please run migrations/seeds first")
		}
	}

	// 6. Inject dummy customers (50-100) — must happen BEFORE sales so we can link them
	fmt.Printf("👥 Injecting dummy customers...\n")
	if err := injectCustomers(ctx, db, startDate, endDate); err != nil {
		return fmt.Errorf("failed to inject customers: %w", err)
	}
	fmt.Printf("   ✅ Customers injected\n")

	// 7. Load customer IDs for sales assignment
	customerIDs := getIDs(ctx, db, "customers")
	if len(customerIDs) == 0 {
		return fmt.Errorf("no customers found after injection")
	}

	// 8. Load walk-in customer ID
	var walkInCustomerID int
	err = db.QueryRowContext(ctx, "SELECT id FROM customers WHERE is_walk_in = true LIMIT 1").Scan(&walkInCustomerID)
	if err != nil {
		return fmt.Errorf("no walk-in customer found: %w", err)
	}

	// 9. Inject sales transactions (10-20 per day across all days)
	fmt.Printf("💰 Injecting daily sales (10-20 per day across %d days)...\n", numDays)

	if err := injectDailySales(ctx, db, userIDs, productData, customerIDs, walkInCustomerID, startDate, endDate); err != nil {
		return fmt.Errorf("failed to inject sales: %w", err)
	}

	// 10. Generate audit log entries for all created data
	fmt.Printf("📋 Generating audit log entries...\n")
	if err := generateAuditLogs(ctx, db, userIDs, categoryIDs, startDate, endDate); err != nil {
		return fmt.Errorf("failed to generate audit logs: %w", err)
	}
	fmt.Println("   ✅ Audit log entries generated")

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

// truncateAllData removes all business and master data while preserving admin data (roles, permissions, users, stores)
func truncateAllData(ctx context.Context, db *sql.DB) error {
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

	// Save system users (system role, non-test/dummy) before truncation
	type sysUser struct {
		id, roleID                    int
		username, email, passwordHash string
		isActive                      bool
	}
	var systemUsers []sysUser
	rows, err := db.QueryContext(ctx, `
		SELECT u.id, u.username, u.email, u.password_hash, u.role_id, u.is_active
		FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE r.is_system = true
		  AND u.username NOT ILIKE '%test%'
		  AND u.username NOT ILIKE '%user%'
		  AND u.username NOT ILIKE '%dummy%'
		  AND u.email NOT ILIKE '%test%'
		  AND u.email NOT ILIKE '%dummy%'`)
	if err == nil {
		for rows.Next() {
			var u sysUser
			if err := rows.Scan(&u.id, &u.username, &u.email, &u.passwordHash, &u.roleID, &u.isActive); err == nil {
				systemUsers = append(systemUsers, u)
			}
		}
		rows.Close()
	} else {
		log.Printf("Warning: could not save system users: %v", err)
	}

	// Truncate tables in correct order (children first)
	tables := []string{
		"sale_items",
		"product_stock",
		"inventory_movements",
		"sales",
		"products",
		"customers",
		"payment_methods",
		"warehouses",
		"units_of_measure",
		"tax_classes",
		"brands",
		"categories",
		"audit_logs",
		"refresh_tokens",
		"users",
	}

	for _, table := range tables {
		_, err := db.ExecContext(ctx, fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE", table))
		if err != nil {
			return fmt.Errorf("failed to truncate %s: %w", table, err)
		}
	}

	// Restore system users
	for _, u := range systemUsers {
		_, err := db.ExecContext(ctx,
			`INSERT INTO users (id, username, email, password_hash, role_id, is_active, created_at) VALUES ($1, $2, $3, $4, $5, $6, NOW()) ON CONFLICT (id) DO NOTHING`,
			u.id, u.username, u.email, u.passwordHash, u.roleID, u.isActive,
		)
		if err != nil {
			log.Printf("Warning: failed to restore system user %d: %v", u.id, err)
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
				"INSERT INTO categories (name, slug, description, is_active) VALUES ($1, $2, $3, true) RETURNING id",
				catName, generateSlug(catName), fmt.Sprintf("Auto-generated category for %s products", catName),
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

// generateSlug creates a URL-friendly slug from a name
func generateSlug(name string) string {
	slug := strings.ToLower(strings.TrimSpace(name))
	replacements := []struct{ from, to string }{
		{" ", "-"}, {"'", ""}, {`"`, ""}, {"&", "and"}, {"/", "-"},
		{"+", "plus"}, {"=", "equals"}, {"?", ""}, {"!", ""}, {"@", "at"},
		{"#", "number"}, {"%", "percent"}, {"(", ""}, {")", ""},
	}
	for _, r := range replacements {
		slug = strings.ReplaceAll(slug, r.from, r.to)
	}
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	slug = strings.Trim(slug, "-")
	if len(slug) > 120 {
		slug = slug[:120]
	}
	return slug
}

func ensureTaxClasses(ctx context.Context, db *sql.DB) {
	_, err := db.ExecContext(ctx, `
		INSERT INTO tax_classes (id, name, rate_percent, description, is_active, created_at)
		VALUES
		(1, 'PPN 11%', 11.00, 'Pajak Pertambahan Nilai standar 11%', true, NOW()),
		(2, 'PPN 0%', 0.00, 'Tidak dikenakan PPN', true, NOW()),
		(3, 'Non PPN', 0.00, 'Produk tidak kena PPN', true, NOW())
		ON CONFLICT (id) DO NOTHING`)
	if err != nil {
		fmt.Printf("Warning: failed to ensure tax classes: %v\n", err)
	}
}

func ensureBrands(ctx context.Context, db *sql.DB) {
	_, err := db.ExecContext(ctx, `
		INSERT INTO brands (id, name, description, is_active, created_at, updated_at)
		VALUES
		(1, 'Indofood', 'Produk makanan dari PT Indofood Sukses Makmur', true, NOW(), NOW()),
		(2, 'Sosro', 'Minuman teh dalam kemasan', true, NOW(), NOW()),
		(3, 'Wings', 'Snack dan makanan ringan', true, NOW(), NOW()),
		(4, 'Unilever', 'Produk konsumsi sehari-hari', true, NOW(), NOW()),
		(5, 'Lokal', 'Brand lokal/produk umum', true, NOW(), NOW())
		ON CONFLICT (id) DO NOTHING`)
	if err != nil {
		fmt.Printf("Warning: failed to ensure brands: %v\n", err)
	}
}

func ensurePaymentMethods(ctx context.Context, db *sql.DB) {
	_, err := db.ExecContext(ctx, `
		INSERT INTO payment_methods (code, name, is_active, requires_reference, sort_order, created_at)
		VALUES
		('CASH', 'Cash', true, false, 1, NOW()),
		('CARD', 'Card', true, true, 2, NOW()),
		('E_WALLET', 'E-Wallet', true, true, 3, NOW()),
		('TRANSFER', 'Transfer', true, true, 4, NOW()),
		('QRIS', 'QRIS', true, false, 5, NOW())
		ON CONFLICT (code) DO NOTHING`)
	if err != nil {
		fmt.Printf("Warning: failed to ensure payment methods: %v\n", err)
	}
}

func ensureUnitsOfMeasure(ctx context.Context, db *sql.DB) {
	_, err := db.ExecContext(ctx, `
		INSERT INTO units_of_measure (id, code, name, description, is_active, created_at)
		VALUES
		(1, 'pcs', 'Pieces', 'Satuan individual/buah', true, NOW()),
		(2, 'box', 'Box', 'Kemasan kotak', true, NOW()),
		(3, 'dus', 'Dus', 'Kemasan karton/dus', true, NOW()),
		(4, 'kg', 'Kilogram', 'Kilogram', true, NOW()),
		(5, 'gram', 'Gram', 'Gram', true, NOW()),
		(6, 'liter', 'Liter', 'Liter', true, NOW()),
		(7, 'ml', 'Mililiter', 'Mililiter', true, NOW()),
		(8, 'pack', 'Pack', 'Kemasan/pack', true, NOW())
		ON CONFLICT (id) DO NOTHING`)
	if err != nil {
		fmt.Printf("Warning: failed to ensure units of measure: %v\n", err)
	}
}

func cleanupTestRoles(ctx context.Context, db *sql.DB) {
	_, err := db.ExecContext(ctx, `DELETE FROM roles WHERE (name ILIKE '%dummy%' OR name ILIKE '%test%') AND id NOT IN (SELECT DISTINCT role_id FROM users WHERE role_id IS NOT NULL)`)
	if err != nil {
		fmt.Printf("Warning: failed to clean up test/dummy roles: %v\n", err)
	}
}

// productWorkerJob represents a job for a worker in the concurrent product injection pool
type productWorkerJob struct {
	workerID      int
	startProduct  int
	endProduct    int
	categoryIDs   []int
	categoryNames map[int]string
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
		`INSERT INTO products (sku, name, barcode, price, cost, stock, category_id, status, tax_class_id, brand_id, unit_of_measure_id, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, 'active', 1, $8, $9, $10) RETURNING id`)
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
		brandID := rand.Intn(5) + 1
		uomID := rand.Intn(8) + 1
		err := stmt.QueryRowContext(ctx, sku, name, barcode, price, cost, stock, catID, brandID, uomID, createdAt).Scan(&id)
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
	numWorkers := 1 // Single worker avoids lock contention for max throughput

	ref := time.Now().In(jakartaTZ)
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

			workerSales := processWorkerJob(ctx, db, job, userIDs, products, productMap, startDate, ref, salesCreated)
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

type SaleItemRecord struct {
	SaleID    int
	ProductID int
	Quantity  int
	UnitPrice int
	Subtotal  int
}

// processWorkerJob handles a single worker's portion of the work with optimized batch transactions
func processWorkerJob(ctx context.Context, db *sql.DB, job workerJob, userIDs []int, products []ProductInfo, productMap map[int]ProductInfo, startDate, ref time.Time, progress chan<- int) int {
	invoiceCounter := 0

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("Worker %d: begin tx: %v", job.workerID, err)
		return 0
	}

	saleStmt, err := tx.PrepareContext(ctx,
		`INSERT INTO sales (invoice_number, cashier_id, customer_id, store_id, payment_method, status, subtotal, discount, tax, total_amount, created_at)
		 VALUES ($1, $2, $3, NULL, $4, 'completed', $5, 0, $6, $7, $8) RETURNING id`)
	if err != nil {
		log.Printf("Worker %d: prepare sale stmt: %v", job.workerID, err)
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("failed to rollback: %v", rbErr)
		}
		return 0
	}
	defer saleStmt.Close()

	itemStmt, err := tx.PrepareContext(ctx,
		`INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal, dpp_amount, tax_amount) VALUES ($1, $2, $3, $4, $5, $6, $7)`)
	if err != nil {
		log.Printf("Worker %d: prepare item stmt: %v", job.workerID, err)
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("failed to rollback: %v", rbErr)
		}
		return 0
	}
	defer itemStmt.Close()

	stockStmt, err := tx.PrepareContext(ctx, `
		UPDATE product_stock
		SET quantity = GREATEST(0, quantity - $1), updated_at = NOW()
		WHERE product_id = $2 AND warehouse_id IS NULL AND store_id IS NULL`)
	if err != nil {
		log.Printf("Worker %d: prepare stock stmt: %v", job.workerID, err)
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("failed to rollback: %v", rbErr)
		}
		return 0
	}
	defer stockStmt.Close()

	stockInsertStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO product_stock (product_id, quantity, updated_at)
		VALUES ($1, GREATEST(0, 0-$2), NOW())`)
	if err != nil {
		log.Printf("Worker %d: prepare stock insert stmt: %v", job.workerID, err)
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("failed to rollback: %v", rbErr)
		}
		return 0
	}
	defer stockInsertStmt.Close()

	movementStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO inventory_movements (product_id, quantity_change, type, reference_id, reference_table, user_id, notes, created_at)
		VALUES ($1, $2, 'sale', $3, 'sales', $4, $5, $6)`)
	if err != nil {
		log.Printf("Worker %d: prepare movement stmt: %v", job.workerID, err)
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("failed to rollback: %v", rbErr)
		}
		return 0
	}
	defer movementStmt.Close()

	salesCreated := 0
	batchSize := 0

	for _, dayOffset := range job.days {
		dayDate := startDate.AddDate(0, 0, dayOffset)

		salesForDay := 10 + rand.Intn(11)

		for s := 0; s < salesForDay; s++ {
			invoiceNum := job.startInvoice + invoiceCounter
			invoice := fmt.Sprintf("INV-%d-%06d", ref.Year(), invoiceNum)
			invoiceCounter++

			cashierID := randElemInt(userIDs)
			// 30-50% of sales use the walk-in/general customer
			customerID := randElemInt(job.customerIDs)
			if rand.Intn(100) < 40 {
				customerID = job.walkInCustomerID
			}
			paymentMethod := weightedRandomChoice(paymentMethods, paymentWeights)
			createdAt := randomTime24Hour(dayDate, ref)

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
				log.Printf("Worker %d: insert sale %s: %v", job.workerID, invoice, err)
				continue
			}

			// Sort items by ProductID — consistent lock order prevents deadlocks
			sort.Slice(items, func(i, j int) bool { return items[i].ProductID < items[j].ProductID })

			// Insert sale items, update stock, record movements
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
					log.Printf("Worker %d: insert item %s: %v", job.workerID, invoice, err)
					continue
				}

				// Decrement product stock
				res, err := stockStmt.ExecContext(ctx, item.Quantity, item.ProductID)
				if err != nil {
					log.Printf("Worker %d: update stock %s (product %d): %v", job.workerID, invoice, item.ProductID, err)
					continue
				}
				if n, _ := res.RowsAffected(); n == 0 {
					if _, err := stockInsertStmt.ExecContext(ctx, item.ProductID, item.Quantity); err != nil {
						log.Printf("Worker %d: insert stock %s (product %d): %v", job.workerID, invoice, item.ProductID, err)
					}
				}

				// Record inventory movement
				if _, err := movementStmt.ExecContext(ctx, item.ProductID, -item.Quantity, saleID, cashierID, fmt.Sprintf("Sale %s", invoice), createdAt); err != nil {
					log.Printf("Worker %d: insert movement %s: %v", job.workerID, invoice, err)
				}
			}

			salesCreated++
			batchSize++

			if batchSize >= 500 {
				saleStmt.Close()
				itemStmt.Close()
				stockStmt.Close()
				stockInsertStmt.Close()
				movementStmt.Close()
				if err := tx.Commit(); err != nil {
					log.Printf("failed to commit batch: %v", err)
				}

				tx, _ = db.BeginTx(ctx, nil)
				saleStmt, _ = tx.PrepareContext(ctx,
					`INSERT INTO sales (invoice_number, cashier_id, customer_id, store_id, payment_method, status, subtotal, discount, tax, total_amount, created_at)
					 VALUES ($1, $2, $3, NULL, $4, 'completed', $5, 0, $6, $7, $8) RETURNING id`)
				itemStmt, _ = tx.PrepareContext(ctx,
					`INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal, dpp_amount, tax_amount) VALUES ($1, $2, $3, $4, $5, $6, $7)`)
				stockStmt, _ = tx.PrepareContext(ctx, `
					UPDATE product_stock
					SET quantity = GREATEST(0, quantity - $1), updated_at = NOW()
					WHERE product_id = $2 AND warehouse_id IS NULL AND store_id IS NULL`)
				stockInsertStmt, _ = tx.PrepareContext(ctx, `
					INSERT INTO product_stock (product_id, quantity, updated_at)
					VALUES ($1, GREATEST(0, 0-$2), NOW())`)
				movementStmt, _ = tx.PrepareContext(ctx, `
					INSERT INTO inventory_movements (product_id, quantity_change, type, reference_id, reference_table, user_id, notes, created_at)
					VALUES ($1, $2, 'sale', $3, 'sales', $4, $5, $6)`)

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
	stockStmt.Close()
	stockInsertStmt.Close()
	movementStmt.Close()
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
func randomTime24Hour(dayDate, ref time.Time) time.Time {
	isToday := dayDate.Year() == ref.Year() && dayDate.Month() == ref.Month() && dayDate.Day() == ref.Day()

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
		currentHour := ref.In(jakartaTZ).Hour()
		currentMinute := ref.Minute()
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
			sum += digits[i] * 3 // odd positions (1-based): i=0->pos1, i=2->pos3, etc.
		} else {
			sum += digits[i] * 1 // even positions (1-based): i=1->pos2, i=3->pos4, etc.
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
	if len(products) == 0 {
		return nil
	}
	selected := make([]int, 0, count)
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
	fmt.Println("  5. Custom")
	fmt.Println("  6. Unlimited (generate all time)")

	var choice int
	for {
		fmt.Print("\nEnter your choice (1-6): ")
		_, err := fmt.Fscan(reader, &choice)
		if err == nil && choice >= 1 && choice <= 6 {
			break
		}
		if err != nil {
			_, _ = reader.ReadString('\n') // clear buffer
		}
		fmt.Println("Invalid choice. Please enter 1-6.")
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
			fmt.Print("Enter number of months: ")
			_, err := fmt.Fscan(reader, &months)
			if err == nil && months >= 1 {
				return months * 30 // approximate days
			}
			if err != nil {
				_, _ = reader.ReadString('\n')
			}
			fmt.Println("Invalid input. Please enter 6 or more.")
		}
	case 6:
		return 99999 // effectively unlimited
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

// generateAuditLogs creates audit log entries matching the seeded data
func generateAuditLogs(ctx context.Context, db *sql.DB, userIDs, categoryIDs []int, startDate, endDate time.Time) error {
	if len(userIDs) == 0 {
		return nil
	}

	rows := make([]string, 0, 500)
	addRow := func(userID int, role, action, entityType string, entityID *int, description string, createdAt time.Time) {
		var eid any = "NULL"
		if entityID != nil {
			eid = *entityID
		}
		ip := randomPrivateIP()
		rows = append(rows, fmt.Sprintf(
			"(%d, '%s', '%s', '%s', %v, NULL, NULL, '%s', NULL, '%s', '%s')",
			userID, escapeSQL(role), escapeSQL(action), escapeSQL(entityType),
			eid, ip, escapeSQL(description),
			createdAt.Format("2006-01-02 15:04:05-07"),
		))
	}

	userID := userIDs[0]
	ref := endDate.In(jakartaTZ)

	// Login events for first few users
	for i := 0; i < len(userIDs) && i < 3; i++ {
		loginTime := ref.Add(-time.Duration(rand.Intn(48)) * time.Hour)
		addRow(userIDs[i], "superadmin", "login", "auth", nil, "User logged in", loginTime)
		logoutTime := loginTime.Add(time.Duration(8+rand.Intn(4)) * time.Hour)
		addRow(userIDs[i], "superadmin", "logout", "auth", nil, "User logged out", logoutTime)
	}

	// Category creation
	for _, catID := range categoryIDs {
		addRow(userID, "superadmin", "create", "category", &catID,
			fmt.Sprintf("Created category #%d", catID),
			ref.Add(-time.Duration(rand.Intn(72))*time.Hour))
	}

	// Brand creation
	brandID := 1
	addRow(userID, "superadmin", "create", "brand", &brandID, "Created brand: Indofood", ref.Add(-time.Duration(rand.Intn(24))*time.Hour))
	brandID = 2
	addRow(userID, "superadmin", "create", "brand", &brandID, "Created brand: Sosro", ref.Add(-time.Duration(rand.Intn(24))*time.Hour))
	brandID = 3
	addRow(userID, "superadmin", "create", "brand", &brandID, "Created brand: Wings", ref.Add(-time.Duration(rand.Intn(24))*time.Hour))
	brandID = 4
	addRow(userID, "superadmin", "create", "brand", &brandID, "Created brand: Unilever", ref.Add(-time.Duration(rand.Intn(24))*time.Hour))
	brandID = 5
	addRow(userID, "superadmin", "create", "brand", &brandID, "Created brand: Lokal", ref.Add(-time.Duration(rand.Intn(24))*time.Hour))

	// Tax class creation
	for id, name := range map[int]string{1: "PPN 11%", 2: "PPN 0%", 3: "Non PPN"} {
		v := id
		addRow(userID, "superadmin", "create", "tax_class", &v,
			fmt.Sprintf("Created tax class: %s", name),
			ref.Add(-time.Duration(rand.Intn(48))*time.Hour))
	}

	// UOM creation
	for id, name := range map[int]string{1: "Pieces", 2: "Box", 3: "Dus", 4: "Kilogram", 5: "Gram", 6: "Liter", 7: "Mililiter", 8: "Pack"} {
		v := id
		addRow(userID, "superadmin", "create", "uom", &v,
			fmt.Sprintf("Created unit of measure: %s", name),
			ref.Add(-time.Duration(rand.Intn(48))*time.Hour))
	}

	// Product creation entries (sample: one per 100 products, plus last)
	var productIDs []int
	prodRows, err := db.QueryContext(ctx, "SELECT id FROM products ORDER BY id")
	if err == nil {
		for prodRows.Next() {
			var pid int
			if err := prodRows.Scan(&pid); err == nil {
				productIDs = append(productIDs, pid)
			}
		}
		prodRows.Close()
	}
	for i := 0; i < len(productIDs); i += 100 {
		pid := productIDs[i]
		addRow(userID, "superadmin", "create", "product", &pid,
			fmt.Sprintf("Created product #%d (batch %d)", pid, i/100+1),
			ref.Add(-time.Duration(48+rand.Intn(48))*time.Hour))
	}
	if len(productIDs) > 0 {
		lastPID := productIDs[len(productIDs)-1]
		addRow(userID, "superadmin", "create", "product", &lastPID,
			fmt.Sprintf("Created product #%d", lastPID),
			ref.Add(-time.Duration(rand.Intn(24))*time.Hour))
	}

	// Customer creation entries (sample: one per 10 customers)
	var customerIDs []int
	custRows, err := db.QueryContext(ctx, "SELECT id FROM customers WHERE is_walk_in = false ORDER BY id")
	if err == nil {
		for custRows.Next() {
			var cid int
			if err := custRows.Scan(&cid); err == nil {
				customerIDs = append(customerIDs, cid)
			}
		}
		custRows.Close()
	}
	for i := 0; i < len(customerIDs); i += 10 {
		cid := customerIDs[i]
		addRow(userID, "superadmin", "create", "customer", &cid,
			fmt.Sprintf("Created customer #%d", cid),
			ref.Add(-time.Duration(72+rand.Intn(48))*time.Hour))
	}

	// Sale creation audit logs — generated from sales table
	saleRows, err := db.QueryContext(ctx,
		`SELECT id, cashier_id, customer_id, invoice_number, total_amount, payment_method, created_at
		 FROM sales ORDER BY id`)
	if err == nil {
		defer saleRows.Close()
		for saleRows.Next() {
			var sid, cid, custID, total int
			var inv, pm string
			var ct time.Time
			if err := saleRows.Scan(&sid, &cid, &custID, &inv, &total, &pm, &ct); err != nil {
				continue
			}
			nv := fmt.Sprintf(`{"invoice_number":"%s","cashier_id":%d,"customer_id":%d,"total_amount":%d,"payment_method":"%s","status":"completed"}`,
				inv, cid, custID, total, pm)
			ip := randomPrivateIP()
			rows = append(rows, fmt.Sprintf(
				"(%d, 'cashier', 'create', 'sale', %d, NULL, '%s'::jsonb, '%s', NULL, 'Created sale #%d', '%s')",
				cid, sid, escapeSQL(nv), ip, sid, ct.Format("2006-01-02 15:04:05-07"),
			))
		}
	}

	if len(rows) == 0 {
		return nil
	}

	// Batch insert audit logs
	batchSize := 100
	for i := 0; i < len(rows); i += batchSize {
		end := i + batchSize
		if end > len(rows) {
			end = len(rows)
		}
		batch := rows[i:end]
		query := `INSERT INTO audit_logs (user_id, role, action, entity_type, entity_id, old_values, new_values, ip_address, user_agent, description, created_at) VALUES `
		query += strings.Join(batch, ", ")
		if _, err := db.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("failed to insert audit logs batch %d: %w", i/batchSize, err)
		}
	}

	return nil
}

func escapeSQL(s string) string {
	s = strings.ReplaceAll(s, "'", "''")
	return s
}

func randomPrivateIP() string {
	switch rand.Intn(3) {
	case 0:
		return fmt.Sprintf("10.%d.%d.%d", rand.Intn(256), rand.Intn(256), rand.Intn(256))
	case 1:
		return fmt.Sprintf("172.%d.%d.%d", 16+rand.Intn(16), rand.Intn(256), rand.Intn(256))
	default:
		return fmt.Sprintf("192.168.%d.%d", rand.Intn(256), rand.Intn(256))
	}
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
		`INSERT INTO customers (name, phone, email, address, note, is_active, is_walk_in, created_at)
		 VALUES ($1, $2, $3, $4, $5, true, false, $6)`)
	if err != nil {
		return fmt.Errorf("prepare: %w", err)
	}
	defer stmt.Close()

	ref := time.Now().In(jakartaTZ)

	// Insert a walk-in/general customer first
	walkInID := 0
	err = tx.QueryRowContext(ctx,
		`INSERT INTO customers (name, phone, email, address, note, is_active, is_walk_in, created_at)
		 VALUES ('Walk-in / General', '', '', NULL, NULL, true, true, $1)
		 RETURNING id`,
		ref,
	).Scan(&walkInID)
	if err != nil {
		return fmt.Errorf("insert walk-in customer: %w", err)
	}

	customerStreets := []string{
		"Jl. Merdeka", "Jl. Sudirman", "Jl. Gatot Subroto", "Jl. Ahmad Yani",
		"Jl. Diponegoro", "Jl. Pahlawan", "Jl. Anggrek", "Jl. Melati",
		"Jl. Kenanga", "Jl. Mawar", "Jl. Flamboyan", "Jl. Cempaka",
		"Jl. Kartini", "Jl. Sisingamangaraja", "Jl. Veteran", "Jl. Gajah Mada",
		"Jl. Hayam Wuruk", "Jl. Juanda", "Jl. Pemuda", "Jl. Siliwangi",
	}
	customerCities := []string{
		"Jakarta Pusat", "Jakarta Selatan", "Jakarta Barat", "Jakarta Timur", "Jakarta Utara",
		"Bandung", "Surabaya", "Semarang", "Yogyakarta", "Medan",
		"Makassar", "Palembang", "Denpasar", "Malang", "Bekasi",
	}
	customerNotes := []string{
		"", "", "", "",
		"Pelanggan tetap", "Member premium", "Rekomendasi dari teman",
		"Pernah komplain", "Pembayaran tunai", "Pembayaran transfer",
		"Alergi seafood", "Request packaging khusus", "Catatan: antar ke dapur",
	}

	for i := 0; i < numCustomers; i++ {
		first := customerFirstNames[rand.Intn(len(customerFirstNames))]
		last := customerLastNames[rand.Intn(len(customerLastNames))]
		name := fmt.Sprintf("%s %s", first, last)
		phone := fmt.Sprintf("08%s", fmt.Sprintf("%010d", rand.Intn(10000000000)))
		email := fmt.Sprintf("%s.%s@email.com", strings.ToLower(first), strings.ToLower(last))
		address := fmt.Sprintf("%s No. %d, %s", customerStreets[rand.Intn(len(customerStreets))], 1+rand.Intn(200), customerCities[rand.Intn(len(customerCities))])
		note := customerNotes[rand.Intn(len(customerNotes))]

		daysAgo := rand.Intn(int(ref.Sub(startDate).Hours()/24)) + 1
		createdAt := ref.AddDate(0, 0, -daysAgo)

		if _, err := stmt.ExecContext(ctx, name, phone, email, address, note, createdAt); err != nil {
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
