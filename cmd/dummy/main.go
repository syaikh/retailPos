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
	"runtime"
	"sort"
	"strconv"
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

	// Central company name for store generation (single name for all branches)
	centralCompanyName = "Jadi Baru"

	// Store location names (Indonesian cities/regions) - no duplicates
	storeLocationNames = []string{
		"Kebumen", "Sumpiuh", "Kroya", "Purwokerto", "Cilacap", "Banyumas",
		"Wangon", "Ajibarang", "Gombong", "Karanganyar", "Kebasen", "Kutowinangun",
		"Purwojati", "Sempor", "Ambal", "Kuwarasan", "Bonorowo", "Adimulyo",
		"Alian", "Jogonalan", "Kaligesing", "Kutorejo", "Musuk", "Cepogo",
		"Selo", "Boyolali", "Klego", "Ngemplak", "Sawit", "Andong",
		"Kemusu", "Giriroto", "Wonosari", "Teras", "Juwangi", "Ketawang",
		"Plandi", "Sumberlawang", "Mojolaban", "Sukoharjo", "Ngadirejo", "Ngargoyoso",
		"Kerjo", "Jaten", "Jumantono", "Gondangrejo", "Karas",
		"Tasikmadu", "Karangpandan", "Colomadu", "Ngaglik", "Sleman", "Depok",
		"Gamping", "Godean", "Moyudan", "Sedayu", "Lendah", "Sentolo",
		"Panjatan", "Wates", "Temon", "Tambak", "Banguntapan", "Pleret",
		"Prambanan", "Kalasan", "Berbah", "Ngablak", "Bringin",
		"Getasan", "Sumber", "Sumowono", "Bawen", "Bandungan",
		"Umbulomah", "Jambu", "Kandangan", "Kedungaret", "Secang",
		"Mertoyudan", "Sawangan", "Mungkid", "Mendut", "Borobudur", "Salaman",
		"Kajoran", "Kaliangkrik", "Bandongan", "Candirejo",
	}

	// Warehouse names (Indonesian)
	warehouseNamePool = []string{
		"Gudang Pusat", "Gudang Utama", "Gudang Sentral", "Gudang Induk",
		"Gudang Cabang", "Gudang Distribusi", "Gudang Logistik",
		"Gudang Konsinyasi", "Gudang Penyimpanan", "Gudang Operasional",
		"Gudang Regional", "Gudang Area", "Gudang Lokal", "Gudang Cabang Utama",
		"Gudang Pengiriman", "Gudang Penerimaan", "Gudang Sortir", "Gudang Transit",
	}

	// Supplier company names (Indonesian)
	supplierCompanyNames = []string{
		"PT Sumber Makmur", "CV Berkah Jaya", "PT Maju Bersama", "CV Sentosa Trading",
		"PT Dewa Elektronik", "CV Lestari Supplies", "PT Nusantara Distribution",
		"CV Prima Kencana", "PT Gemilang Perkasa", "CV Sinar Terang",
		"PT Abadi Makmur", "CV Cahaya Baru", "PT Sejahtera Abadi", "CV Mitra Usaha",
		"PT Global Supply", "CV Bintang Jaya", "PT Multi Sukses", "CV Harapan Jaya",
		"PT Cahaya Gemilang", "CV Mutiara Supplies", "PT Prima Nusantara", "CV Sentosa Makmur",
		"PT Berkah Abadi", "CV Jaya Bersama", "PT Sukses Mandiri", "CV Cemerlang Trading",
		"PT Mas Jaya", "CV Sumber Rejeki", "PT Prima Sejahtera", "CV Maju Jaya",
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
		defer func() { _ = db.Close() }()
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
	stockOpnamesFlag := flag.Int("stock-opnames", 0, "Number of stock opname sessions to inject (auto ~1/month if 0)")
	storesFlag := flag.Int("stores", 0, "Number of stores to generate (random 20-40 if 0)")
	warehousesFlag := flag.Int("warehouses", 1, "Number of warehouses per store")
	storageZonesFlag := flag.Int("storage-zones", 4, "Number of storage zones per warehouse")
	storageRacksFlag := flag.Int("storage-racks", 5, "Number of racks per storage zone")
	suppliersFlag := flag.Int("suppliers", 0, "Number of suppliers to generate (random 10-15 if 0)")
	consignmentFlag := flag.Int("consignment", 0, "Number of consignment suppliers (10-20% of suppliers if 0)")
	flag.Parse()

	if err := run(*truncateFlag, *productsFlag, *daysFlag, *categoriesFlag, *stockOpnamesFlag,
		*storesFlag, *warehousesFlag, *storageZonesFlag, *storageRacksFlag, *suppliersFlag, *consignmentFlag); err != nil {
		log.Fatalf("Dummy seeder failed: %v", err)
	}
}

func run(truncateData bool, numProducts, numDays, numCategories, numStockOpnames,
	numStores, numWarehousesPerStore, storageZones, storageRacks, numSuppliers, numConsignment int) error {
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

	// Randomize categories if not specified (0 means random)
	if numCategories == 0 {
		numCategories = rand.Intn(36) + 65 // 65-100
	}

	// Validate and randomize new parameters
	if numStores < 0 {
		return fmt.Errorf("stores count must not be negative, got %d", numStores)
	}
	if numStores == 0 {
		numStores = rand.Intn(21) + 20 // 20-40
	}
	if numWarehousesPerStore < 0 {
		return fmt.Errorf("warehouses count must not be negative, got %d", numWarehousesPerStore)
	}
	if numWarehousesPerStore == 0 {
		numWarehousesPerStore = 1
	}
	if storageZones < 0 {
		return fmt.Errorf("storage zones count must not be negative, got %d", storageZones)
	}
	if storageZones == 0 {
		storageZones = 4
	}
	if storageZones > 26 {
		fmt.Printf("   ⚠️  Capping storage zones from %d to 26 (max supported)\n", storageZones)
		storageZones = 26
	}
	if storageRacks < 0 {
		return fmt.Errorf("storage racks count must not be negative, got %d", storageRacks)
	}
	if storageRacks == 0 {
		storageRacks = 5
	}
	if numSuppliers < 0 {
		return fmt.Errorf("suppliers count must not be negative, got %d", numSuppliers)
	}
	if numSuppliers == 0 {
		numSuppliers = rand.Intn(6) + 10 // 10-15
	}
	if numConsignment < 0 {
		return fmt.Errorf("consignment count must not be negative, got %d", numConsignment)
	}
	if numConsignment == 0 {
		// Default: 10-20% of suppliers
		numConsignment = numSuppliers * (rand.Intn(11) + 10) / 100
		if numConsignment < 1 && numSuppliers > 0 {
			numConsignment = 1
		}
	}
	if numConsignment > numSuppliers {
		numConsignment = numSuppliers
	}

	// Calculate date range
	endDate := time.Now().In(jakartaTZ)
	startDate := endDate.AddDate(0, 0, -numDays)

	// Connect to database
	db, err := sql.Open("postgres", getDSN())
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer func() { _ = db.Close() }()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Randomize the product count if not specified (0 means random). When
	// re-seeding (-truncate=false) a dataset that already has products, reuse
	// them instead of injecting a fresh 4500+ set whose index-based SKUs would
	// collide with the unique products.sku key and abort the whole run.
	if numProducts == 0 {
		var existingProducts int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM products`).Scan(&existingProducts); err == nil && existingProducts > 0 {
			fmt.Printf("   Found %d existing products, reusing them for this re-seed\n", existingProducts)
		} else {
			numProducts = rand.Intn(1001) + 4500 // 4500-5500
		}
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

	// Re-check product count after truncation — the pre-truncation count may
	// have been non-zero but the table was wiped. Generate new products so the
	// seeder doesn't silently produce an empty dataset.
	if numProducts == 0 {
		var postTruncCount int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM products`).Scan(&postTruncCount); err == nil && postTruncCount == 0 {
			numProducts = rand.Intn(1001) + 4500 // 4500-5500
		}
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

	// 3e. Ensure stores exist
	ensureStores(ctx, db, numStores)
	fmt.Println("   ✅ Stores ready")

	// 3e1. Ensure warehouses exist
	fmt.Printf("🏭 Ensuring warehouses...\n")
	ensureWarehouses(ctx, db, numWarehousesPerStore)
	fmt.Println("   ✅ Warehouses ready")

	// 3e2. Ensure storage locations (racks) exist
	fmt.Printf("📍 Ensuring storage locations...\n")
	ensureStorageLocations(ctx, db, storageZones, storageRacks)
	fmt.Println("   ✅ Storage locations ready")

	// 3f. Ensure customer groups exist
	fmt.Printf("👥 Ensuring customer groups...\n")
	ensureCustomerGroups(ctx, db)
	fmt.Println("   ✅ Customer groups ready")

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

	// 4a. Backfill rack-level stock for storage locations
	fmt.Printf("📍 Backfilling rack-level stock...\n")
	backfillRackStock(ctx, db)
	fmt.Println("   ✅ Rack stock backfilled")

	// 4b. Ensure suppliers exist and link to products
	fmt.Printf("🏭 Injecting suppliers and product links...\n")
	if err := ensureSuppliers(ctx, db, productData, numSuppliers); err != nil {
		return fmt.Errorf("failed to inject suppliers: %w", err)
	}
	fmt.Println("   ✅ Suppliers and product links ready")

	// 4c. Ensure pricing rules exist
	fmt.Printf("💰 Injecting pricing rules...\n")
	if err := ensurePricingRules(ctx, db, productData); err != nil {
		return fmt.Errorf("failed to inject pricing rules: %w", err)
	}
	fmt.Println("   ✅ Pricing rules ready")

	// 4d. Inject purchase orders and goods receipts
	fmt.Printf("📋 Injecting purchase orders and goods receipts...\n")
	if err := injectPurchaseOrdersAndGRs(ctx, db, startDate, endDate); err != nil {
		return fmt.Errorf("failed to inject purchase orders: %w", err)
	}
	fmt.Println("   ✅ Purchase orders and goods receipts injected")

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

	fmt.Printf("🔄 Injecting shifts for %d days...\n", numDays)
	if err := injectShifts(ctx, db, startDate, endDate); err != nil {
		return fmt.Errorf("failed to inject shifts: %w", err)
	}
	fmt.Println("   ✅ Shifts injected")

	if err := syncInvoiceSequence(ctx, db); err != nil {
		return fmt.Errorf("failed to sync invoice sequence: %w", err)
	}

	// 9c. Inject consignment (Konsinyasi Supplier) data — after sales/shifts so
	// consignment sale items can be backfilled, before stock opnames so
	// consignment-owned SKUs are excluded from the opname snapshot.
	fmt.Printf("🤝 Injecting consignment (Konsinyasi) data...\n")
	if err := injectConsignment(ctx, db, startDate, endDate, numConsignment); err != nil {
		return fmt.Errorf("failed to inject consignment: %w", err)
	}
	fmt.Println("   ✅ Consignment data injected")

	// 9b. Inject stock opname sessions (realistic, mostly approved history + one active session)
	fmt.Printf("📦 Injecting stock opname sessions...\n")
	if err := injectStockOpnames(ctx, db, startDate, endDate, numStockOpnames); err != nil {
		return fmt.Errorf("failed to inject stock opnames: %w", err)
	}

	// 10. Generate audit log entries for all created data
	fmt.Printf("📋 Generating audit log entries...\n")
	if err := generateAuditLogs(ctx, db, userIDs, categoryIDs, startDate, endDate); err != nil {
		return fmt.Errorf("failed to generate audit logs: %w", err)
	}
	fmt.Println("   ✅ Audit log entries generated")

	// 11. Refresh materialized views so report queries return data immediately
	fmt.Printf("🔄 Refreshing materialized views...\n")
	if _, err := db.ExecContext(ctx, `SELECT refresh_sales_mv()`); err != nil {
		return fmt.Errorf("failed to refresh materialized views: %w", err)
	}
	fmt.Println("   ✅ Materialized views refreshed")

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
		reportsTo                     *int
		isActive                      bool
	}
	var systemUsers []sysUser
	rows, err := db.QueryContext(ctx, `
		SELECT u.id, u.username, u.email, u.password_hash, u.role_id, u.reports_to, u.is_active
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
			var reportsTo sql.NullInt64
			if err := rows.Scan(&u.id, &u.username, &u.email, &u.passwordHash, &u.roleID, &reportsTo, &u.isActive); err == nil {
				if reportsTo.Valid {
					v := int(reportsTo.Int64)
					u.reportsTo = &v
				}
				systemUsers = append(systemUsers, u)
			}
		}
		_ = rows.Close()
	} else {
		log.Printf("Warning: could not save system users: %v", err)
	}

	// Truncate tables in correct order (children first)
	tables := []string{
		// Consignment child tables first
		"consignment_settlement_items",
		"consignment_payouts",
		"consignment_return_items",
		"consignment_receipt_items",
		"consignment_returns",
		"consignment_settlements",
		"consignment_receipts",
		"consignment_pending_returns",
		"consignment_terms",
		"consignment_stock",
		"consignment_sale_items",
		"consignment_arrangements",
		// Core transaction tables
		"goods_receipt_items",
		"goods_receipts",
		"purchase_order_items",
		"purchase_orders",
		"stock_opname_assignments",
		"stock_opname_counts",
		"stock_opname_items",
		"stock_opnames",
		"sale_items",
		"product_stock",
		"storage_locations",
		"inventory_movements",
		"sales",
		"shifts",
		"pricing_rules",
		"product_suppliers",
		"suppliers",
		"products",
		"customers",
		"payment_methods",
		"warehouses",
		"stores",
		"units_of_measure",
		"tax_classes",
		"brands",
		"categories",
		"customer_groups",
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
		var reportsTo interface{}
		if u.reportsTo != nil {
			reportsTo = *u.reportsTo
		}
		_, err := db.ExecContext(ctx,
			`INSERT INTO users (id, username, email, password_hash, role_id, reports_to, is_active, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW()) ON CONFLICT (id) DO NOTHING`,
			u.id, u.username, u.email, u.passwordHash, u.roleID, reportsTo, u.isActive,
		)
		if err != nil {
			log.Printf("Warning: failed to restore system user %d: %v", u.id, err)
		}
	}

	// Resync sequences after explicit ID inserts to prevent duplicate key errors
	if _, err := db.ExecContext(ctx, `SELECT setval('users_id_seq', (SELECT COALESCE(MAX(id), 1) FROM users))`); err != nil {
		log.Printf("Warning: failed to resync users_id_seq: %v", err)
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
	defer func() { _ = rows.Close() }()

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

func ensureStores(ctx context.Context, db *sql.DB, numStores int) {
	// Check if stores already exist
	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM stores").Scan(&count); err == nil && count > 0 {
		fmt.Printf("   Found %d existing stores, skipping creation\n", count)
		return
	}

	// Shuffle location names for variety
	locations := make([]string, len(storeLocationNames))
	copy(locations, storeLocationNames)
	rand.Shuffle(len(locations), func(i, j int) {
		locations[i], locations[j] = locations[j], locations[i]
	})

	// Warn if requesting more stores than available locations
	if numStores > len(locations) {
		fmt.Printf("   ⚠️  Capping stores from %d to %d (max available locations)\n", numStores, len(locations))
		numStores = len(locations)
	}

	// Generate store names: {CentralCompanyName} {Location}
	stmt, err := db.PrepareContext(ctx, `
		INSERT INTO stores (id, name, address, phone, is_active, created_at)
		VALUES ($1, $2, $3, $4, true, NOW())
		ON CONFLICT (id) DO NOTHING`)
	if err != nil {
		fmt.Printf("Warning: failed to prepare store stmt: %v\n", err)
		return
	}
	defer func() { _ = stmt.Close() }()

	created := 0
	for i := 0; i < numStores && i < len(locations); i++ {
		storeName := fmt.Sprintf("%s %s", centralCompanyName, locations[i])
		address := fmt.Sprintf("Jl. Raya %s No. %d, %s", locations[i], 1+rand.Intn(100), locations[i])
		phone := fmt.Sprintf("081%d%08d", i%10, rand.Intn(100000000))
		if _, err := stmt.ExecContext(ctx, i+1, storeName, address, phone); err != nil {
			fmt.Printf("Warning: failed to insert store %s: %v\n", storeName, err)
			continue
		}
		created++
	}
	fmt.Printf("   🎲 Created %d stores\n", created)
}

func ensureWarehouses(ctx context.Context, db *sql.DB, numWarehousesPerStore int) {
	// Check if warehouses already exist
	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM warehouses").Scan(&count); err == nil && count > 0 {
		fmt.Printf("   Found %d existing warehouses, skipping creation\n", count)
		return
	}

	// Get store IDs so warehouses are linked to existing stores
	rows, err := db.QueryContext(ctx, `SELECT id FROM stores ORDER BY id`)
	if err != nil {
		fmt.Printf("Warning: failed to query stores for warehouses: %v\n", err)
		return
	}
	defer func() { _ = rows.Close() }()

	var storeIDs []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err == nil {
			storeIDs = append(storeIDs, id)
		}
	}
	if len(storeIDs) == 0 {
		fmt.Println("Warning: no stores found, skipping warehouse creation")
		return
	}

	// Shuffle warehouse names for variety
	names := make([]string, len(warehouseNamePool))
	copy(names, warehouseNamePool)
	rand.Shuffle(len(names), func(i, j int) {
		names[i], names[j] = names[j], names[i]
	})

	stmt, err := db.PrepareContext(ctx, `
		INSERT INTO warehouses (name, code, address, store_id, is_active, created_at)
		VALUES ($1, $2, $3, $4, true, NOW())
		ON CONFLICT (code) DO NOTHING`)
	if err != nil {
		fmt.Printf("Warning: failed to prepare warehouse stmt: %v\n", err)
		return
	}
	defer func() { _ = stmt.Close() }()

	created := 0
	whNum := 0
	for _, storeID := range storeIDs {
		for j := 0; j < numWarehousesPerStore && j < len(names); j++ {
			whNum++
			whName := names[j%len(names)]
			code := fmt.Sprintf("WH-%03d", whNum)
			address := fmt.Sprintf("Jl. Gudang No. %d, Gudang %s", 1+rand.Intn(50), whName)
			if _, err := stmt.ExecContext(ctx, whName, code, address, storeID); err != nil {
				fmt.Printf("Warning: failed to insert warehouse %s: %v\n", whName, err)
				continue
			}
			created++
		}
	}
	fmt.Printf("   🎲 Created %d warehouses (%d per store)\n", created, numWarehousesPerStore)
}

func ensureStorageLocations(ctx context.Context, db *sql.DB, storageZones, storageRacks int) {
	// Check if storage locations already exist
	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM storage_locations").Scan(&count); err == nil && count > 0 {
		fmt.Printf("   Found %d existing storage locations, skipping creation\n", count)
		return
	}

	// Get warehouse IDs
	warehouseIDs := getIDs(ctx, db, "warehouses")
	if len(warehouseIDs) == 0 {
		fmt.Println("   ⚠️  No warehouses found, skipping storage location creation")
		return
	}

	// Generate zone labels: A, B, C, ... up to storageZones
	zones := make([]string, storageZones)
	for i := 0; i < storageZones; i++ {
		zones[i] = string(rune('A' + i))
	}

	stmt, err := db.PrepareContext(ctx, `
		INSERT INTO storage_locations (code, name, warehouse_id, notes, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, true, NOW(), NOW())
		ON CONFLICT (code) DO NOTHING`)
	if err != nil {
		fmt.Printf("Warning: failed to prepare storage location stmt: %v\n", err)
		return
	}
	defer func() { _ = stmt.Close() }()

	totalCreated := 0
	for _, whID := range warehouseIDs {
		created := 0
		for _, zone := range zones {
			for rack := 1; rack <= storageRacks; rack++ {
				code := fmt.Sprintf("WH%d-%s%02d", whID, zone, rack)
				name := fmt.Sprintf("Rak %s-%02d", zone, rack)
				notes := fmt.Sprintf("Zone %s, Rack %d", zone, rack)
				if _, err := stmt.ExecContext(ctx, code, name, whID, notes); err != nil {
					fmt.Printf("Warning: failed to insert storage location %s: %v\n", code, err)
					continue
				}
				created++
			}
		}
		totalCreated += created
	}
	fmt.Printf("   🎲 Created %d storage locations across %d warehouses (%d zones × %d racks)\n",
		totalCreated, len(warehouseIDs), storageZones, storageRacks)
}

func backfillRackStock(ctx context.Context, db *sql.DB) {
	// Get warehouse IDs
	warehouseIDs := getIDs(ctx, db, "warehouses")
	if len(warehouseIDs) == 0 {
		fmt.Println("   ⚠️  No warehouses found, skipping rack stock backfill")
		return
	}

	// Get location IDs per warehouse
	type whLocations struct {
		id       int
		locationIDs []int
	}
	var whList []whLocations
	for _, whID := range warehouseIDs {
		rows, err := db.QueryContext(ctx, `
			SELECT id FROM storage_locations WHERE warehouse_id = $1 AND is_active = true ORDER BY id`, whID)
		if err != nil {
			continue
		}
		var ids []int
		for rows.Next() {
			var id int
			if err := rows.Scan(&id); err == nil {
				ids = append(ids, id)
			}
		}
		_ = rows.Close()
		if len(ids) > 0 {
			whList = append(whList, whLocations{id: whID, locationIDs: ids})
		}
	}
	if len(whList) == 0 {
		fmt.Println("   ⚠️  No warehouses with locations found, skipping rack stock backfill")
		return
	}

	// Get active product IDs (sample a subset for rack stock)
	prodRows, err := db.QueryContext(ctx, `
		SELECT id FROM products WHERE status = 'active' ORDER BY id`)
	if err != nil {
		fmt.Printf("Warning: failed to query products for rack stock: %v\n", err)
		return
	}
	defer func() { _ = prodRows.Close() }()

	var productIDs []int
	for prodRows.Next() {
		var id int
		if err := prodRows.Scan(&id); err == nil {
			productIDs = append(productIDs, id)
		}
	}
	if len(productIDs) == 0 {
		fmt.Println("   ⚠️  No active products found, skipping rack stock backfill")
		return
	}

	// Shuffle products and assign ~30% to rack stock
	rand.Shuffle(len(productIDs), func(i, j int) { productIDs[i], productIDs[j] = productIDs[j], productIDs[i] })
	rackProductCount := len(productIDs) * 30 / 100
	if rackProductCount == 0 {
		rackProductCount = 1
	}

	stmt, err := db.PrepareContext(ctx, `
		INSERT INTO product_stock (product_id, warehouse_id, store_id, location_id, quantity, reorder_point, reorder_quantity, created_at, updated_at)
		VALUES ($1, $2, NULL, $3, $4, $5, $6, NOW(), NOW())
		ON CONFLICT ON CONSTRAINT uq_product_stock DO UPDATE SET
			quantity = product_stock.quantity + EXCLUDED.quantity,
			updated_at = NOW()`)
	if err != nil {
		fmt.Printf("Warning: failed to prepare rack stock stmt: %v\n", err)
		return
	}
	defer func() { _ = stmt.Close() }()

	totalRows := 0
	for i := 0; i < rackProductCount; i++ {
		prodID := productIDs[i]
		// Pick a random warehouse and random location within it
		wh := whList[rand.Intn(len(whList))]
		if len(wh.locationIDs) == 0 {
			continue
		}
		locID := wh.locationIDs[rand.Intn(len(wh.locationIDs))]
		qty := 5 + rand.Intn(46) // 5-50 units per rack
		reorderPoint := 5 + rand.Intn(10)
		reorderQty := 10 + rand.Intn(41) // 10-50

		if _, err := stmt.ExecContext(ctx, prodID, wh.id, locID, qty, reorderPoint, reorderQty); err != nil {
			// Skip silently (constraint violations, etc.)
			continue
		}
		totalRows++
	}
	fmt.Printf("   🎲 Backfilled %d rack-level stock rows across %d warehouses\n", totalRows, len(whList))
}

func ensureCustomerGroups(ctx context.Context, db *sql.DB) {
	_, err := db.ExecContext(ctx, `
		INSERT INTO customer_groups (name, description, is_active, color, created_at, updated_at)
		VALUES
		('Walk-in', 'Pelanggan umum tanpa kartu member', true, '#636E72', NOW(), NOW()),
		('Member', 'Pelanggan terdaftar dengan kartu member', true, '#00B894', NOW(), NOW()),
		('VIP', 'Pelanggan prioritas dengan harga khusus', true, '#6C5CE7', NOW(), NOW()),
		('Reseller', 'Reseller/pengusaha kecil dengan harga grosir', true, '#0984E3', NOW(), NOW()),
		('Corporate', 'Pelanggan korporat dengan volume besar', true, '#E17055', NOW(), NOW()),
		('Student', 'Pelajar/mahasiswa dengan diskon khusus', true, '#FFD93D', NOW(), NOW()),
		('Wholesale', 'Pembeli grosir/toko lain', true, '#00CEC9', NOW(), NOW()),
		('Online', 'Pelanggan dari channel online/marketplace', true, '#E84393', NOW(), NOW())
		ON CONFLICT (name) DO UPDATE SET
			description = EXCLUDED.description,
			color = EXCLUDED.color,
			updated_at = NOW()`)
	if err != nil {
		fmt.Printf("Warning: failed to ensure customer groups: %v\n", err)
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

var (
	supplierStreets = []string{
		"Jl. Industri", "Jl. Raya Bogor", "Jl. Pasar Minggu", "Jl. Kemang Raya",
		"Jl. Tebet Raya", "Jl. Senayan", "Jl. Kuningan", "Jl. Rasuna Said",
	}
	supplierCities = []string{
		"Jakarta Selatan", "Jakarta Timur", "Tangerang", "Bekasi", "Depok",
	}
)

func ensureSuppliers(ctx context.Context, db *sql.DB, products []ProductInfo, numSuppliers int) error {
	// Check if suppliers already exist
	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM suppliers").Scan(&count); err == nil && count > 0 {
		fmt.Printf("   Found %d existing suppliers, skipping creation\n", count)
		return nil
	}

	// Cap to available names
	if numSuppliers > len(supplierCompanyNames) {
		numSuppliers = len(supplierCompanyNames)
	}

	// Shuffle names for variety
	names := make([]string, len(supplierCompanyNames))
	copy(names, supplierCompanyNames)
	rand.Shuffle(len(names), func(i, j int) {
		names[i], names[j] = names[j], names[i]
	})

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			log.Printf("failed to rollback: %v", err)
		}
	}()

	// Insert suppliers
	supplierStmt, err := tx.PrepareContext(ctx,
		`INSERT INTO suppliers (name, code, contact_name, phone, email, address, is_active, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, true, $7) RETURNING id`)
	if err != nil {
		return fmt.Errorf("prepare supplier stmt: %w", err)
	}
	defer func() { _ = supplierStmt.Close() }()

	ref := time.Now().In(jakartaTZ)
	supplierIDs := make([]int, 0, numSuppliers)

	for i := 0; i < numSuppliers; i++ {
		supplierName := names[i]
		contactFirst := customerFirstNames[rand.Intn(len(customerFirstNames))]
		contactLast := customerLastNames[rand.Intn(len(customerLastNames))]
		contactName := fmt.Sprintf("%s %s", contactFirst, contactLast)
		phone := fmt.Sprintf("021-%08d", rand.Intn(100000000))
		email := fmt.Sprintf("info@%s.co.id", strings.ToLower(strings.ReplaceAll(supplierName, " ", "")))
		address := fmt.Sprintf("%s No. %d, %s", supplierStreets[rand.Intn(len(supplierStreets))], 1+rand.Intn(100), supplierCities[rand.Intn(len(supplierCities))])
		createdAt := ref.AddDate(0, 0, -rand.Intn(90)-30)

		var id int
		if err := supplierStmt.QueryRowContext(ctx, supplierName, fmt.Sprintf("SUP-%03d", i+1), contactName, phone, email, address, createdAt).Scan(&id); err != nil {
			fmt.Printf("   Warning: failed to insert supplier %s: %v\n", supplierName, err)
			continue
		}
		supplierIDs = append(supplierIDs, id)
	}

	linkCount := 0
	if len(products) > 0 {
		// Link suppliers to products (each product gets 1-3 suppliers)
		linkStmt, err := tx.PrepareContext(ctx,
			`INSERT INTO product_suppliers (product_id, supplier_id, supplier_sku, unit_cost, lead_time_days, is_preferred, created_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)
			 ON CONFLICT (product_id, supplier_id) DO NOTHING`)
		if err != nil {
			return fmt.Errorf("prepare link stmt: %w", err)
		}
		defer func() { _ = linkStmt.Close() }()

		for _, p := range products {
			numLinks := 1 + rand.Intn(3) // 1-3 suppliers per product
			if numLinks > len(supplierIDs) {
				numLinks = len(supplierIDs)
			}

			// Shuffle supplier IDs for this product
			shuffled := make([]int, len(supplierIDs))
			copy(shuffled, supplierIDs)
			rand.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })

			for j := 0; j < numLinks; j++ {
				supID := shuffled[j]
				sku := fmt.Sprintf("SKU-P%d-S%d", p.ID, supID)
				unitCost := int(float64(p.Price) * (0.5 + rand.Float64()*0.3)) // 50-80% of product price
				leadTime := 1 + rand.Intn(14)                                  // 1-14 days
				isPreferred := j == 0                                          // first supplier is preferred
				createdAt := ref.AddDate(0, 0, -rand.Intn(60))

				if _, err := linkStmt.ExecContext(ctx, p.ID, supID, sku, unitCost, leadTime, isPreferred, createdAt); err != nil {
					// Silently skip constraint violations (e.g. preferred index)
					continue
				}
				linkCount++
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit suppliers: %w", err)
	}

	fmt.Printf("   🎲 Created %d suppliers with %d product links\n", len(supplierIDs), linkCount)
	return nil
}

func ensurePricingRules(ctx context.Context, db *sql.DB, products []ProductInfo) error {
	if len(products) == 0 {
		return nil
	}

	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM pricing_rules").Scan(&count); err == nil && count > 0 {
		fmt.Printf("   Found %d existing pricing rules, skipping creation\n", count)
		return nil
	}

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
		`INSERT INTO pricing_rules (product_id, category_id, brand_id, pricing_type, pricing_method, pricing_value, name, minimum_quantity, maximum_quantity, priority, is_active, status, allow_combine, customer_group_id, store_id, recurrence_days, time_from, time_to, effective_from, effective_until, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, true, 'approved', $11, $12, $13, $14, $15, $16, $17, $18, $19)`)
	if err != nil {
		return fmt.Errorf("prepare pricing rule stmt: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	ref := time.Now().In(jakartaTZ)
	ruleCount := 0

	// Pick 3 products for detailed rule demos — spread across product list
	p1 := products[0]
	p2 := products[len(products)/3]
	p3 := products[(len(products)*2)/3]
	effectiveFrom := ref.AddDate(0, 0, -30)
	effectiveUntil := ref.AddDate(0, 6, 0)

	exec := func(pType, method, name string, val float64, pid, catID, brandID, minQty, maxQty, priority int, combine bool, custGroup, store int, days, tFrom, tTo string) {
		if _, err := stmt.ExecContext(ctx,
			nullableInt(pid), nullableInt(catID), nullableInt(brandID),
			pType, method, val, name, minQty, nullableInt(maxQty), priority,
			combine, nullableInt(custGroup), nullableInt(store),
			nullableTextArray(days), nullableStr(tFrom), nullableStr(tTo),
			effectiveFrom, effectiveUntil, ref); err != nil {
			fmt.Printf("   ⚠️  Pricing rule '%s' failed: %v\n", name, err)
		} else {
			ruleCount++
		}
	}

	// === SPECIAL_PRICE rules (6) ===

	// 1. fixed_price wholesale
	exec("special_price", "fixed_price", "Harga Grosir Min 5", float64(p1.Price)*0.85, p1.ID, 0, 0, 5, 0, 0, false, 0, 0, "", "", "")

	// 2. discount_percent bulk
	exec("special_price", "discount_percent", "Diskon 10% Min 3", 10, p2.ID, 0, 0, 3, 0, 0, false, 0, 0, "", "", "")

	// 3. discount_amount member
	exec("special_price", "discount_amount", "Potongan Rp 5.000", 5000, p3.ID, 0, 0, 1, 0, 0, false, 2, 0, "", "", "")

	// 4. markup_percent reseller
	exec("special_price", "markup_percent", "Harga Reseller +5%", 5, p1.ID, 0, 0, 2, 0, 0, false, 3, 0, "", "", "")

	// 5. category-wide (category_id = 1, Smartphones)
	exec("special_price", "fixed_price", "Harga Khusus Elektronik", 1500000, 0, 1, 0, 1, 0, 0, false, 0, 0, "", "", "")

	// 6. brand-wide (brand_id = 1, Indofood)
	exec("special_price", "discount_percent", "Diskon Brand Indofood", 8, 0, 0, 1, 1, 0, 0, false, 0, 0, "", "", "")

	// === PROMOTION rules (6) ===

	// 7. fixed_price flash sale
	exec("promotion", "fixed_price", "Flash Sale Rp 99.000", 99000, p1.ID, 0, 0, 1, 10, 1, false, 0, 0, "", "", "")

	// 8. discount_percent weekend
	exec("promotion", "discount_percent", "Diskon Weekend 15%", 15, p2.ID, 0, 0, 1, 0, 1, true, 0, 0, "fri,sat,sun", "", "")

	// 9. discount_amount happy hour
	exec("promotion", "discount_amount", "Happy Hour Rp 10.000 Off", 10000, p3.ID, 0, 0, 1, 0, 1, true, 0, 0, "", "12:00", "14:00")

	// 10. markup_percent bundle
	exec("promotion", "markup_percent", "Bundle Premium +10%", 10, p1.ID, 0, 0, 2, 0, 1, false, 0, 0, "", "", "")

	// 11. promotion category-wide
	exec("promotion", "discount_percent", "Promo Kategori Laptops 12%", 12, 0, 2, 0, 1, 0, 1, true, 0, 0, "mon,tue,wed,thu,fri", "", "")

	// 12. promotion brand-wide
	exec("promotion", "discount_amount", "Cashback Brand Sosro Rp 2.000", 2000, 0, 0, 2, 1, 0, 1, false, 0, 0, "", "", "")

	// === STACKING rules (2) — promotion chain on same product ===

	// 13. stacking: extra 5% off on top of special_price
	exec("promotion", "discount_percent", "Extra 5% Off (Stack)", 5, p1.ID, 0, 0, 1, 0, 2, true, 0, 0, "", "", "")

	// 14. stacking: Rp 3.000 cashback (stacks with above)
	exec("promotion", "discount_amount", "Cashback Rp 3.000 (Stack)", 3000, p1.ID, 0, 0, 1, 0, 3, true, 0, 0, "", "", "")

	// === EDGE CASE rules (4) ===

	// 15. store-specific (store_id = 1, Main Store)
	exec("special_price", "fixed_price", "Harga Khusus Main Store", 85000, p2.ID, 0, 0, 1, 0, 0, false, 0, 1, "", "", "")

	// 16. customer_group VIP only
	exec("special_price", "discount_percent", "VIP Exclusive 20%", 20, p3.ID, 0, 0, 1, 0, 0, false, 3, 0, "", "", "")

	// 17. max quantity limit
	exec("special_price", "discount_amount", "Diskon Rp 2.000 (Max 10)", 2000, p1.ID, 0, 0, 1, 10, 0, false, 0, 0, "", "", "")

	// 18. weekday-only promotion
	exec("promotion", "discount_percent", "Promo Weekday 8%", 8, p2.ID, 0, 0, 1, 0, 1, false, 0, 0, "mon,tue,wed,thu,fri", "09:00", "17:00")

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit pricing rules: %w", err)
	}

	fmt.Printf("   🎲 Created %d pricing rules (special_price:6, promotion:6+2 stack, edge:4)\n", ruleCount)
	return nil
}

func nullableInt(v int) interface{} {
	if v == 0 {
		return nil
	}
	return v
}

func nullableStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func nullableTextArray(s string) interface{} {
	if s == "" {
		return nil
	}
	return "{" + s + "}"
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
		`INSERT INTO products (sku, name, barcode, price, cost, category_id, status, tax_class_id, brand_id, unit_of_measure_id, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, 'active', 1, $7, $8, $9)
		 ON CONFLICT (sku) DO NOTHING RETURNING id`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	// Prepare product_stock INSERT (view v_products_full reads stock from product_stock)
	stockStmt, err := tx.PrepareContext(ctx,
		`INSERT INTO product_stock (product_id, quantity, updated_at) VALUES ($1, $2, NOW())
		 ON CONFLICT ON CONSTRAINT uq_product_stock DO UPDATE SET
			quantity = EXCLUDED.quantity, updated_at = NOW()`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare stock statement: %w", err)
	}
	defer func() { _ = stockStmt.Close() }()

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
		err := stmt.QueryRowContext(ctx, sku, name, barcode, price, cost, catID, brandID, uomID, createdAt).Scan(&id)
		if err != nil {
			if err == sql.ErrNoRows {
				continue
			}
			// Detect transaction poisoning: a prior failure (e.g. product_stock FK) aborts
			// the PostgreSQL transaction, causing every subsequent statement to fail with
			// "current transaction is aborted". The only fix is to roll back and start over.
			errStr := err.Error()
			if strings.Contains(errStr, "current transaction is aborted") || strings.Contains(errStr, "abort the current transaction") {
				_ = tx.Rollback()
				return nil, fmt.Errorf("transaction poisoned at product %d: %w", i, err)
			}
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
	stockUpdateCh    chan<- stockUpdateMsg
}

// stockUpdateMsg is a request to update product stock, processed sequentially
type stockUpdateMsg struct {
	productID int
	quantity  int
}

// runStockUpdater processes stock updates sequentially on a dedicated goroutine
// to prevent deadlocks when multiple workers update the same product concurrently.
// Updates are batched to reduce database round-trips.
func runStockUpdater(ctx context.Context, db *sql.DB) chan<- stockUpdateMsg {
	ch := make(chan stockUpdateMsg, 10000)
	go func() {
		stmt, err := db.PrepareContext(ctx, `
			INSERT INTO product_stock (product_id, quantity, updated_at)
			VALUES ($1, GREATEST(0, COALESCE((
				SELECT quantity FROM product_stock WHERE product_id = $1 AND warehouse_id IS NULL AND store_id IS NULL
			), 0) - $2), NOW())
			ON CONFLICT ON CONSTRAINT uq_product_stock DO UPDATE SET
				quantity = GREATEST(0, product_stock.quantity - EXCLUDED.quantity),
				updated_at = NOW()`)
		if err != nil {
			log.Printf("stock updater: prepare stmt failed: %v", err)
			return
		}
		defer func() { _ = stmt.Close() }()

		var batch []stockUpdateMsg
		flush := func() {
			if len(batch) == 0 {
				return
			}
			// Deduplicate by summing quantities per product within the batch
			totals := make(map[int]int)
			for _, msg := range batch {
				totals[msg.productID] += msg.quantity
			}

			if len(totals) == 0 {
				batch = batch[:0]
				return
			}

			// Build multi-row INSERT ... ON CONFLICT to reduce round-trips
			var sb strings.Builder
			sb.WriteString(`INSERT INTO product_stock (product_id, quantity, updated_at) VALUES `)
			args := make([]interface{}, 0, len(totals)*2)
			i := 1
			for productID, qty := range totals {
				if i > 1 {
					sb.WriteString(", ")
				}
				sb.WriteString(fmt.Sprintf("($%d, $%d, NOW())", i, i+1))
				args = append(args, productID, qty)
				i += 2
			}
			sb.WriteString(` ON CONFLICT ON CONSTRAINT uq_product_stock DO UPDATE SET
				quantity = GREATEST(0, product_stock.quantity - EXCLUDED.quantity),
				updated_at = NOW()`)

			if _, err := db.ExecContext(ctx, sb.String(), args...); err != nil {
				log.Printf("stock updater: batch exec failed: %v", err)
				// Fall back to individual queries
				for productID, qty := range totals {
					if _, err := stmt.ExecContext(ctx, productID, qty); err != nil {
						log.Printf("stock updater: product %d qty %d: %v", productID, qty, err)
					}
				}
			}
			batch = batch[:0]
		}

		for msg := range ch {
			batch = append(batch, msg)
			if len(batch) >= 1000 {
				flush()
			}
		}
		flush()
	}()
	return ch
}

// syncInvoiceSequence updates the invoice_seq sequence to match the highest
// existing invoice number so nextval() never returns a duplicate.
func syncInvoiceSequence(ctx context.Context, db *sql.DB) error {
	var maxSeq int
	err := db.QueryRowContext(ctx, `
		SELECT COALESCE(MAX(
			CAST(REGEXP_REPLACE(invoice_number, '^INV-\d+-0*', '') AS bigint)
		), 0)
		FROM sales
		WHERE invoice_number ~ '^INV-\d+-\d+$'
	`).Scan(&maxSeq)
	if err != nil {
		return fmt.Errorf("read max invoice sequence: %w", err)
	}

	_, err = db.ExecContext(ctx, `SELECT setval('invoice_seq', $1)`, maxSeq)
	if err != nil {
		return fmt.Errorf("sync invoice sequence: %w", err)
	}

	fmt.Printf("   🔄 Synced invoice_seq to %d\n", maxSeq+1)
	return nil
}

// syncPOSequences updates po_seq and gr_seq to match the highest existing
// PO/GR numbers (app format "PO-<year>-<seq>" / "GR-<year>-<seq>") so
// nextval() never returns a duplicate, matching GetNextPONumber/GetNextGRNumber.
func syncPOSequences(ctx context.Context, db *sql.DB) error {
	type seqSync struct {
		seq    string
		table  string
		column string
		prefix string
	}
	for _, s := range []seqSync{
		{"po_seq", "purchase_orders", "po_number", "PO-"},
		{"gr_seq", "goods_receipts", "gr_number", "GR-"},
	} {
		var maxSeq int
		err := db.QueryRowContext(ctx, fmt.Sprintf(`
			SELECT COALESCE(MAX(CAST(REGEXP_REPLACE(%s, '^%s\d+-0*', '') AS bigint)), 0)
			FROM %s
			WHERE %s ~ '^%s\d+-\d+$'`, s.column, s.prefix, s.table, s.column, s.prefix)).Scan(&maxSeq)
		if err != nil {
			return fmt.Errorf("read max %s: %w", s.seq, err)
		}
		if maxSeq == 0 {
			// setval(..., false) marks the value as unused, so the next
			// nextval() returns 1 (matching a truncated DB).
			if _, err := db.ExecContext(ctx, fmt.Sprintf(`SELECT setval('%s', 1, false)`, s.seq)); err != nil {
				return fmt.Errorf("sync %s: %w", s.seq, err)
			}
		} else if _, err := db.ExecContext(ctx, fmt.Sprintf(`SELECT setval('%s', $1)`, s.seq), maxSeq); err != nil {
			return fmt.Errorf("sync %s: %w", s.seq, err)
		}
	}
	fmt.Println("   🔄 Synced po_seq and gr_seq")
	return nil
}

// injectDailySales generates transactions ensuring every day has at least 10 transactions using concurrent workers
func injectDailySales(ctx context.Context, db *sql.DB, userIDs []int, products []ProductInfo, customerIDs []int, walkInCustomerID int, startDate, endDate time.Time) error {
	numWorkers := runtime.NumCPU()
	if v := os.Getenv("SALES_WORKERS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 {
			numWorkers = n
		}
	}

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

	// Calculate invoice range per worker based on max possible sales
	maxSalesPerDay := 20 // Upper bound from salesForDay logic
	daysPerWorker := totalDaysInclusive / numWorkers
	remainingDays := totalDaysInclusive % numWorkers
	maxDaysPerWorker := daysPerWorker
	if remainingDays > 0 {
		maxDaysPerWorker++ // First workers get one extra day
	}
	invoicesPerWorker := maxSalesPerDay * maxDaysPerWorker

	jobs := make([]workerJob, numWorkers)

	// Distribute days evenly among workers
	if numWorkers <= 0 {
		return fmt.Errorf("invalid number of workers: %d", numWorkers)
	}

	currentDay := 0
	currentInvoice := 1
	// Continue the invoice counter from existing data so a re-seed
	// (-truncate=false) never collides with invoices from a previous run.
	if err := db.QueryRowContext(ctx, `
		SELECT COALESCE(MAX(CAST(REGEXP_REPLACE(invoice_number, '^INV-\d+-0*', '') AS bigint)), 0) + 1
		FROM sales
		WHERE invoice_number ~ '^INV-\d+-\d+$'
	`).Scan(&currentInvoice); err != nil {
		return fmt.Errorf("compute starting invoice number: %w", err)
	}
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

	fmt.Printf("🚀 Starting %d workers to process %d days\n", numWorkers, totalDaysInclusive)
	fmt.Printf("   Each worker allocated up to %d invoice numbers\n", invoicesPerWorker)

	// Start stock updater (single goroutine prevents deadlock)
	stockUpdateCh := runStockUpdater(ctx, db)
	defer close(stockUpdateCh)

	// Assign stock updater channel to all workers
	for i := range jobs {
		jobs[i].stockUpdateCh = stockUpdateCh
	}

	// Start workers
	var wg sync.WaitGroup
	salesCreated := make(chan int, numWorkers*1000) // Buffered channel for progress updates
	workerErrors := make(chan error, numWorkers)

	for _, job := range jobs {
		wg.Add(1)
		go func(job workerJob) {
			defer wg.Done()

			workerSales, err := processWorkerJob(ctx, db, job, userIDs, products, productMap, startDate, ref, salesCreated)
			if err != nil {
				workerErrors <- fmt.Errorf("worker %d failed: %w", job.workerID, err)
			}
			fmt.Printf("   ✅ Worker %d completed: %d sales\n", job.workerID, workerSales)
		}(job)
	}

	wg.Wait()
	close(salesCreated)
	close(workerErrors)

	totalSales := 0
	for count := range salesCreated {
		totalSales += count
		if totalSales%100 == 0 {
			fmt.Printf("     ...%d sales injected\n", totalSales)
		}
	}

	for err := range workerErrors {
		if err != nil {
			return err
		}
	}

	if totalSales == 0 {
		return fmt.Errorf("no sales were injected across %d days (all workers failed or produced 0 sales)", totalDaysInclusive)
	}

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
func processWorkerJob(ctx context.Context, db *sql.DB, job workerJob, userIDs []int, products []ProductInfo, productMap map[int]ProductInfo, startDate, ref time.Time, progress chan<- int) (int, error) {
	invoiceCounter := 0

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("Worker %d: begin tx: %v", job.workerID, err)
		return 0, fmt.Errorf("worker %d: begin tx: %w", job.workerID, err)
	}

	saleStmt, err := tx.PrepareContext(ctx,
		`INSERT INTO sales (invoice_number, cashier_id, customer_id, store_id, payment_method, status, subtotal, discount, tax, total_amount, created_at)
		 VALUES ($1, $2, $3, NULL, $4, 'completed', $5, 0, $6, $7, $8) RETURNING id`)
	if err != nil {
		log.Printf("Worker %d: prepare sale stmt: %v", job.workerID, err)
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("failed to rollback: %v", rbErr)
		}
		return 0, fmt.Errorf("worker %d: prepare sale stmt: %w", job.workerID, err)
	}
	defer func() { _ = saleStmt.Close() }()

	itemStmt, err := tx.PrepareContext(ctx,
		`INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal, dpp_amount, tax_amount) VALUES ($1, $2, $3, $4, $5, $6, $7)`)
	if err != nil {
		log.Printf("Worker %d: prepare item stmt: %v", job.workerID, err)
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("failed to rollback: %v", rbErr)
		}
		return 0, fmt.Errorf("worker %d: prepare item stmt: %w", job.workerID, err)
	}
	defer func() { _ = itemStmt.Close() }()

	movementStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO inventory_movements (product_id, quantity_change, type, reference_id, reference_table, user_id, notes, created_at)
		VALUES ($1, $2, 'sale', $3, 'sales', $4, $5, $6)`)
	if err != nil {
		log.Printf("Worker %d: prepare movement stmt: %v", job.workerID, err)
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("failed to rollback: %v", rbErr)
		}
		return 0, fmt.Errorf("worker %d: prepare movement stmt: %w", job.workerID, err)
	}
	defer func() { _ = movementStmt.Close() }()

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

				// Queue stock update to single updater goroutine to prevent deadlock
				job.stockUpdateCh <- stockUpdateMsg{productID: item.ProductID, quantity: item.Quantity}

				// Record inventory movement
				if _, err := movementStmt.ExecContext(ctx, item.ProductID, -item.Quantity, saleID, cashierID, fmt.Sprintf("Sale %s", invoice), createdAt); err != nil {
					log.Printf("Worker %d: insert movement %s product %d: %v", job.workerID, invoice, item.ProductID, err)
				}
			}

			salesCreated++
			batchSize++

			if batchSize >= 500 {
				_ = saleStmt.Close()
				_ = itemStmt.Close()
				_ = movementStmt.Close()
				if err := tx.Commit(); err != nil {
					log.Printf("failed to commit batch: %v", err)
				}

				tx, _ = db.BeginTx(ctx, nil)
				saleStmt, _ = tx.PrepareContext(ctx,
					`INSERT INTO sales (invoice_number, cashier_id, customer_id, store_id, payment_method, status, subtotal, discount, tax, total_amount, created_at)
				 VALUES ($1, $2, $3, NULL, $4, 'completed', $5, 0, $6, $7, $8) RETURNING id`)
				itemStmt, _ = tx.PrepareContext(ctx,
					`INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal, dpp_amount, tax_amount) VALUES ($1, $2, $3, $4, $5, $6, $7)`)
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

	_ = saleStmt.Close()
	_ = itemStmt.Close()
	_ = movementStmt.Close()
	if err := tx.Commit(); err != nil {
		log.Printf("failed to commit final batch: %v", err)
	}

	select {
	case progress <- (salesCreated % 25):
	default:
	}

	return salesCreated, nil
}

// injectShifts opens and closes shifts for each day/cashier, linking sales to shifts.
// Runs after all sales are injected so it can query completed sales grouped by date + cashier.
// Optimized: 3 set-based SQL statements instead of one query per day + 4 queries per shift.
func injectShifts(ctx context.Context, db *sql.DB, startDate, endDate time.Time) error {
	// Normalize to midnight Jakarta time so day range [start, end)
	// covers the full calendar day, matching how injectDailySales generates
	// timestamps via randomTime24Hour (which uses date components only).
	start := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, jakartaTZ)
	end := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 0, 0, 0, 0, jakartaTZ).AddDate(0, 0, 1)

	// Pick a default store (first active store, or NULL)
	var storeIDArg any
	var sid int
	if err := db.QueryRowContext(ctx, `SELECT id FROM stores WHERE is_active = true ORDER BY id LIMIT 1`).Scan(&sid); err == nil {
		storeIDArg = sid
	}

	// 1) Create one shift per (cashier, Jakarta date) with completed sales.
	//    Opening time 07:00–08:59 WIB, opening balance 500k–2M.
	//    uq_open_shift_per_user only allows a single 'open' shift per user, so
	//    only the latest sale date per cashier is inserted as 'open'; older
	//    dates are inserted as 'closed' and finalized below with the others.
	rows, err := db.QueryContext(ctx, `
		INSERT INTO shifts (user_id, store_id, status, opening_balance, opened_at, created_at, updated_at)
		SELECT s.cashier_id, $1,
		       CASE WHEN s.sale_date = s.last_date THEN 'open' ELSE 'closed' END,
		       ((500 + floor(random() * 1501)) * 1000)::int,
		       (s.sale_date + time '07:00:00'
		           + (floor(random() * 2)) * interval '1 hour'
		           + floor(random() * 60) * interval '1 minute') AT TIME ZONE 'Asia/Jakarta',
		       NOW(), NOW()
		FROM (
		    SELECT cashier_id,
		           (created_at AT TIME ZONE 'Asia/Jakarta')::date AS sale_date,
		           MAX((created_at AT TIME ZONE 'Asia/Jakarta')::date) OVER (PARTITION BY cashier_id) AS last_date
		    FROM sales
		    WHERE status = 'completed' AND cashier_id IS NOT NULL
		      AND created_at >= $2 AND created_at < $3
		    GROUP BY cashier_id, sale_date
		) s
		WHERE NOT EXISTS (
		    SELECT 1 FROM shifts existing
		    WHERE existing.user_id = s.cashier_id
		      AND (existing.opened_at AT TIME ZONE 'Asia/Jakarta')::date = s.sale_date
		)
		RETURNING id
	`, storeIDArg, start, end)
	if err != nil {
		return fmt.Errorf("open shifts: %w", err)
	}
	defer func() { _ = rows.Close() }()

	shiftCount := 0
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return fmt.Errorf("scan shift id: %w", err)
		}
		shiftCount++
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate shift ids: %w", err)
	}
	_ = rows.Close()

	// NOTE: when re-seeding, step 1 may create 0 new shifts (every (cashier,
	// date) already has one). The sales-linking step below must still run so
	// newly injected overlapping sales attach to the existing shifts.

	// 2) Link completed sales to the matching shift for the same cashier + Jakarta date.
	//    The join picks the earliest shift per (cashier, date) so a re-seed
	//    (-truncate=false) links new overlapping sales to the existing shifts
	//    from the previous run instead of leaving them unlinked.
	//    (status is not checked here: only the latest date per cashier is 'open'
	//    and older dates are already 'closed', but all of them still need linking.)
	if _, err := db.ExecContext(ctx, `
		UPDATE sales s
		SET shift_id = sh.id
		FROM shifts sh
		WHERE s.status = 'completed' AND s.shift_id IS NULL
		  AND s.cashier_id = sh.user_id
		  AND (s.created_at AT TIME ZONE 'Asia/Jakarta')::date = (sh.opened_at AT TIME ZONE 'Asia/Jakarta')::date
		  AND NOT EXISTS (
		      SELECT 1 FROM shifts sh2
		      WHERE sh2.user_id = sh.user_id
		        AND (sh2.opened_at AT TIME ZONE 'Asia/Jakarta')::date = (sh.opened_at AT TIME ZONE 'Asia/Jakarta')::date
		        AND sh2.id < sh.id
		  )
	`); err != nil {
		return fmt.Errorf("link sales to shifts: %w", err)
	}

	// 3) Aggregate totals per shift and close each one. Random variance simulates
	//    real-world cash counting error; closing time 20:00–22:59 WIB.
	if _, err := db.ExecContext(ctx, `
		WITH totals AS (
		    SELECT shift_id,
		           COALESCE(SUM(CASE WHEN LOWER(payment_method) = 'cash' THEN total_amount ELSE 0 END), 0) AS cash_sales,
		           COALESCE(SUM(CASE WHEN LOWER(payment_method) != 'cash' THEN total_amount ELSE 0 END), 0) AS non_cash_sales,
		           COALESCE(SUM(total_amount), 0) AS total_sales,
		           COUNT(*) AS transaction_count
		    FROM sales
		    WHERE shift_id IS NOT NULL AND status = 'completed'
		    GROUP BY shift_id
		),
		close_vals AS (
		    SELECT sh.id,
		           t.cash_sales, t.non_cash_sales, t.total_sales, t.transaction_count,
		           (floor(random() * 20001) - 10000)::int AS variance,
		           ((sh.opened_at AT TIME ZONE 'Asia/Jakarta')::date + time '20:00:00'
		               + (floor(random() * 3)) * interval '1 hour'
		               + floor(random() * 60) * interval '1 minute') AT TIME ZONE 'Asia/Jakarta' AS closed_at
		    FROM shifts sh
		    JOIN totals t ON t.shift_id = sh.id
		    WHERE sh.updated_at = sh.created_at
		)
		UPDATE shifts sh
		SET status = 'closed',
		    cash_sales = cv.cash_sales,
		    non_cash_sales = cv.non_cash_sales,
		    total_sales = cv.total_sales,
		    transaction_count = cv.transaction_count,
		    closing_balance = sh.opening_balance + cv.cash_sales + cv.variance,
		    discrepancy = cv.variance,
		    needs_review = (cv.variance < -50000 OR cv.variance > 50000),
		    closed_at = cv.closed_at,
		    updated_at = cv.closed_at
		FROM close_vals cv
		WHERE sh.id = cv.id
	`); err != nil {
		return fmt.Errorf("close shifts: %w", err)
	}

	fmt.Printf("   🎲 Created %d shifts across %d days\n", shiftCount, int(endDate.Sub(startDate).Hours()/24)+1)
	return nil
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
	}
	return rand.Intn(4) + 5 // 5-8 items for large transactions
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
		}
		return rand.Intn(5) + 4 // 4-8 for bulk
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
	defer func() { _ = rows.Close() }()
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
		_ = prodRows.Close()
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
		_ = custRows.Close()
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
		defer func() { _ = saleRows.Close() }()
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

// ==================== PURCHASE ORDER & GOODS RECEIPT GENERATION ====================

type poItemInput struct {
	ProductID int
	Qty       int
	UnitCost  int
}

func injectPurchaseOrdersAndGRs(ctx context.Context, db *sql.DB, startDate, endDate time.Time) error {
	// Sync document sequences so re-seeds (-truncate=false) never collide with
	// existing PO/GR numbers (matching the app's GetNextPONumber/GetNextGRNumber).
	if err := syncPOSequences(ctx, db); err != nil {
		return err
	}

	supplierIDs := getIDs(ctx, db, "suppliers")
	if len(supplierIDs) == 0 {
		return fmt.Errorf("no suppliers found")
	}
	products := getExistingProducts(ctx, db)
	if len(products) == 0 {
		return fmt.Errorf("no products found")
	}
	userIDs := getIDs(ctx, db, "users")
	if len(userIDs) == 0 {
		return fmt.Errorf("no users found")
	}
	storeIDs := getIDs(ctx, db, "stores")
	if len(storeIDs) == 0 {
		return fmt.Errorf("no stores found")
	}

	type prodCost struct {
		ProductID int
		UnitCost  int
	}
	costRows, err := db.QueryContext(ctx, `
		SELECT ps.product_id, MIN(ps.unit_cost) as unit_cost
		FROM product_suppliers ps
		WHERE ps.unit_cost IS NOT NULL
		GROUP BY ps.product_id
	`)
	if err != nil {
		return fmt.Errorf("query product costs: %w", err)
	}
	costMap := make(map[int]int)
	for costRows.Next() {
		var pc prodCost
		if err := costRows.Scan(&pc.ProductID, &pc.UnitCost); err == nil {
			costMap[pc.ProductID] = pc.UnitCost
		}
	}
	_ = costRows.Close()

	// Fallback cost: 65% of product price if no supplier link
	for _, p := range products {
		if _, ok := costMap[p.ID]; !ok {
			costMap[p.ID] = int(float64(p.Price) * 0.65)
		}
	}

	// Number of POs: roughly 1 per 25 products, capped
	numPOs := len(products) / 25
	if numPOs < 5 {
		numPOs = 5
	}
	if numPOs > 200 {
		numPOs = 200
	}

	totalDays := int(endDate.Sub(startDate).Hours()/24) + 1
	if totalDays < 1 {
		totalDays = 1
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	poInserted := 0
	grInserted := 0

	type poResult struct {
		poID       int
		status     string
		createdAt  time.Time
		confirmedAt *time.Time
		items      []struct {
			poItemID   int
			qtyOrdered int
			unitCost   int
			productID  int
		}
	}

	// Workers channel
	type poJob struct {
		seq int
	}
	jobs := make(chan poJob, numPOs)
	results := make(chan poResult, numPOs)
	errCh := make(chan error, 4)

	var wg sync.WaitGroup
	for w := 0; w < 4; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				poRes, err := func() (poResult, error) {
					// Pick random values for this PO
					supID := supplierIDs[rand.Intn(len(supplierIDs))]
					storeID := storeIDs[rand.Intn(len(storeIDs))]
					createdBy := userIDs[rand.Intn(len(userIDs))]
					updatedBy := userIDs[rand.Intn(len(userIDs))]

					dayOffset := rand.Intn(totalDays)
					createdAt := startDate.AddDate(0, 0, dayOffset)
					if createdAt.After(endDate) {
						createdAt = endDate
					}

					// 1-5 line items
					numItems := 1 + rand.Intn(5)
					shuffled := make([]int, len(products))
					for i, p := range products {
						shuffled[i] = p.ID
					}
					rand.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
					if numItems > len(shuffled) {
						numItems = len(shuffled)
					}

					items := make([]poItemInput, 0, numItems)
					var subtotal int
					for i := 0; i < numItems; i++ {
						pid := shuffled[i]
						uc := costMap[pid]
						if uc <= 0 {
							uc = 1000
						}
						qty := 2 + rand.Intn(19) // 2-20
						lineSubtotal := uc * qty
						subtotal += lineSubtotal
						items = append(items, poItemInput{
							ProductID: pid,
							Qty:       qty,
							UnitCost:  uc,
						})
					}

					// Status distribution: 60% confirmed, 20% draft, 20% cancelled
					var status string
					var confirmedAt, cancelledAt *time.Time
					var confirmedBy, cancelledBy *int
					r := rand.Intn(100)
					switch {
					case r < 60:
						status = "confirmed"
						ca := createdAt.Add(time.Duration(rand.Intn(72)) * time.Hour)
						confirmedAt = &ca
						cb := userIDs[rand.Intn(len(userIDs))]
						confirmedBy = &cb
					case r < 80:
						status = "draft"
					default:
						status = "cancelled"
						ca := createdAt.Add(time.Duration(rand.Intn(48)) * time.Hour)
						cancelledAt = &ca
						cb := userIDs[rand.Intn(len(userIDs))]
						cancelledBy = &cb
					}

					expectedDate := createdAt.AddDate(0, 0, 3+rand.Intn(12))

					// Payment term always filled; other optional fields ~50% chance each
					terms := []string{"Cash on Delivery", "Net 7", "Net 14", "Net 30", "DP 50%"}
					paymentTerm := terms[rand.Intn(len(terms))]
					var deliveryAddress, supplierRef, notes *string
					if rand.Intn(2) == 0 {
						v := fmt.Sprintf("Jl. Contoh No.%d, Jakarta", 1+rand.Intn(100))
						deliveryAddress = &v
					}
					if rand.Intn(2) == 0 {
						v := fmt.Sprintf("REF-%d", 1000+rand.Intn(9000))
						supplierRef = &v
					}
					if rand.Intn(2) == 0 {
						v := fmt.Sprintf("PO notes batch %d", job.seq)
						notes = &v
					}

					// ~20% chance of discount_amount and tax_amount
					var discAmt, taxAmt int
					if rand.Intn(5) == 0 {
						discAmt = int(float64(subtotal) * (0.05 + rand.Float64()*0.10))
					}
					if rand.Intn(5) == 0 {
						taxAmt = int(float64(subtotal-discAmt) * 0.11)
					}
					grandTotal := subtotal - discAmt + taxAmt
					if grandTotal < 0 {
						grandTotal = subtotal
					}

					tx, err := db.BeginTx(ctx, nil)
					if err != nil {
						return poResult{}, fmt.Errorf("begin tx: %w", err)
					}
					defer func() {
						if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
							log.Printf("po worker rollback: %v", err)
						}
					}()

					var poSeq int
					if err := tx.QueryRowContext(ctx, `SELECT nextval('po_seq')`).Scan(&poSeq); err != nil {
						return poResult{}, fmt.Errorf("nextval po_seq: %w", err)
					}
					poNum := fmt.Sprintf("PO-%d-%06d", createdAt.In(jakartaTZ).Year(), poSeq)
					var poID int
					err = tx.QueryRowContext(ctx, `
						INSERT INTO purchase_orders
							(po_number, supplier_id, store_id, status, expected_date,
							 payment_term, delivery_address, supplier_reference_number, notes,
							 discount_amount, tax_amount,
							 subtotal, grand_total, created_by, updated_by, created_at, updated_at)
						VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$16)
						RETURNING id
					`, poNum, supID, storeID, status, expectedDate,
						paymentTerm, deliveryAddress, supplierRef, notes,
						discAmt, taxAmt,
						subtotal, grandTotal, createdBy, updatedBy, createdAt,
					).Scan(&poID)
					if err != nil {
						return poResult{}, fmt.Errorf("insert PO: %w", err)
					}

					// Update confirmed/cancelled fields
					if confirmedAt != nil {
						_, err = tx.ExecContext(ctx, `
							UPDATE purchase_orders SET confirmed_at=$1, confirmed_by=$2 WHERE id=$3
						`, *confirmedAt, *confirmedBy, poID)
						if err != nil {
							return poResult{}, fmt.Errorf("confirm PO: %w", err)
						}
					}
					if cancelledAt != nil {
						_, err = tx.ExecContext(ctx, `
							UPDATE purchase_orders SET cancelled_at=$1, cancelled_by=$2 WHERE id=$3
						`, *cancelledAt, *cancelledBy, poID)
						if err != nil {
							return poResult{}, fmt.Errorf("cancel PO: %w", err)
						}
					}

					itemStmt, err := tx.PrepareContext(ctx, `
					INSERT INTO purchase_order_items
						(purchase_order_id, product_id, qty_ordered, unit_cost, subtotal, product_name, sku, barcode)
					VALUES ($1,$2,$3,$4,$5,
						COALESCE((SELECT name FROM products WHERE id=$2), ''),
						COALESCE((SELECT sku FROM products WHERE id=$2), ''),
						COALESCE((SELECT barcode FROM products WHERE id=$2), ''))
					RETURNING id
					`)
					if err != nil {
						return poResult{}, fmt.Errorf("prepare item stmt: %w", err)
					}
					defer func() { _ = itemStmt.Close() }()

					var poRes poResult
					poRes.poID = poID
					poRes.status = status
					poRes.createdAt = createdAt
					poRes.confirmedAt = confirmedAt
					for _, item := range items {
						var poItemID int
						err := itemStmt.QueryRowContext(ctx, poID, item.ProductID,
							item.Qty, item.UnitCost, item.UnitCost*item.Qty,
						).Scan(&poItemID)
						if err != nil {
							return poResult{}, fmt.Errorf("insert PO item: %w", err)
						}
						poRes.items = append(poRes.items, struct {
							poItemID   int
							qtyOrdered int
							unitCost   int
							productID  int
						}{poItemID, item.Qty, item.UnitCost, item.ProductID})
					}

					if err := tx.Commit(); err != nil {
						return poResult{}, fmt.Errorf("commit PO: %w", err)
					}
					return poRes, nil
				}()
				if err != nil {
					select {
					case errCh <- err:
					default:
					}
					return
				}
				results <- poRes
			}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
		close(errCh)
	}()

	// Feed jobs
	for seq := 1; seq <= numPOs; seq++ {
		jobs <- poJob{seq: seq}
	}
	close(jobs)

	// Collect results
	type poItemInfo struct {
		poItemID   int
		qtyOrdered int
		unitCost   int
		productID  int
	}
	type poInfo struct {
		poID       int
		status     string
		items      []poItemInfo
		createdAt  time.Time
		confirmedAt *time.Time
	}
	var confirmedPOs []poInfo

	for poRes := range results {
		poInserted++
		info := poInfo{
			poID:       poRes.poID,
			status:     poRes.status,
			createdAt:  poRes.createdAt,
			confirmedAt: poRes.confirmedAt,
			items:      make([]poItemInfo, len(poRes.items)),
		}
		for j, item := range poRes.items {
			info.items[j] = poItemInfo{
				poItemID:   item.poItemID,
				qtyOrdered: item.qtyOrdered,
				unitCost:   item.unitCost,
				productID:  item.productID,
			}
		}
		if info.status == "confirmed" {
			confirmedPOs = append(confirmedPOs, info)
		}
	}

	// Check errors
	select {
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("PO worker error: %w", err)
		}
	default:
	}

	fmt.Printf("   🎲 Created %d purchase orders (%d confirmed, %d draft/cancelled)\n",
		poInserted, len(confirmedPOs), poInserted-len(confirmedPOs))

	// ---------- Goods Receipts ----------
	// ~80% of confirmed POs get GRs; ~70% full receipt, ~30% partial
	for _, po := range confirmedPOs {
		if rand.Intn(100) >= 80 {
			continue
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin gr tx: %w", err)
		}
		defer func() {
			if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
				log.Printf("gr worker rollback: %v", err)
			}
		}()

		storeID := storeIDs[rand.Intn(len(storeIDs))]
		receivedBy := userIDs[rand.Intn(len(userIDs))]
		// receivedAt must be after confirmedAt to maintain temporal consistency
		afterConfirm := po.createdAt.Add(time.Duration(1+rand.Intn(72)) * time.Hour)
		if po.confirmedAt != nil && po.confirmedAt.After(afterConfirm) {
			afterConfirm = po.confirmedAt.Add(time.Duration(1+rand.Intn(24)) * time.Hour)
		}
		receivedAt := afterConfirm

		grInserted++
		var grSeq int
		if err := tx.QueryRowContext(ctx, `SELECT nextval('gr_seq')`).Scan(&grSeq); err != nil {
			return fmt.Errorf("nextval gr_seq: %w", err)
		}
		grNum := fmt.Sprintf("GR-%d-%06d", receivedAt.In(jakartaTZ).Year(), grSeq)

		// Optional GR fields — ~50% chance
		var doNumber, grNotes *string
		if rand.Intn(2) == 0 {
			v := fmt.Sprintf("DO-%s-%04d", receivedAt.Format("200601"), rand.Intn(9999))
			doNumber = &v
		}
		if rand.Intn(2) == 0 {
			v := fmt.Sprintf("GR notes for PO %d", po.poID)
			grNotes = &v
		}

		var grID int
		err = tx.QueryRowContext(ctx, `
			INSERT INTO goods_receipts
				(gr_number, purchase_order_id, store_id, received_by, received_at,
				 delivery_order_number, notes, created_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$5)
			RETURNING id
		`, grNum, po.poID, storeID, receivedBy, receivedAt,
			doNumber, grNotes,
		).Scan(&grID)
		if err != nil {
			return fmt.Errorf("insert GR: %w", err)
		}

		itemStmt, err := tx.PrepareContext(ctx, `
			INSERT INTO goods_receipt_items
				(goods_receipt_id, purchase_order_item_id, product_id, qty_good, qty_damaged, unit_cost, product_name)
			VALUES ($1,$2,$3,$4,$5,$6,
				COALESCE((SELECT name FROM products WHERE id=$3), ''))
		`)
		if err != nil {
			return fmt.Errorf("prepare gr item stmt: %w", err)
		}
		defer func() { _ = itemStmt.Close() }()

		for _, item := range po.items {
			fullReceipt := rand.Intn(100) < 70
			qtyGood := item.qtyOrdered
			qtyDamaged := 0
			if !fullReceipt {
				// Partial receipt: 30-95% of ordered qty
				factor := 0.30 + rand.Float64()*0.65
				qtyGood = int(float64(item.qtyOrdered) * factor)
				if qtyGood < 1 {
					qtyGood = 1
				}
			}
			if rand.Intn(100) < 5 && qtyGood >= 2 {
				// 5% chance of damaged goods (only when qtyGood >= 2 to avoid Intn(0))
				qtyDamaged = 1 + rand.Intn(qtyGood/2)
				if qtyDamaged > qtyGood {
					qtyDamaged = qtyGood
				}
				qtyGood -= qtyDamaged
			}
			if qtyGood < 0 {
				qtyGood = 0
			}

			_, err := itemStmt.ExecContext(ctx, grID, item.poItemID, item.productID,
				qtyGood, qtyDamaged, item.unitCost,
			)
			if err != nil {
				return fmt.Errorf("insert GR item: %w", err)
			}

			// Update PO item qty_received
			qtyReceived := qtyGood + qtyDamaged
			_, err = tx.ExecContext(ctx, `
				UPDATE purchase_order_items SET qty_received = qty_received + $1 WHERE id = $2
			`, qtyReceived, item.poItemID)
			if err != nil {
				return fmt.Errorf("update PO item qty_received: %w", err)
			}

			// Update product stock
			_, err = tx.ExecContext(ctx, `
				INSERT INTO product_stock (product_id, quantity, updated_at)
				VALUES ($1, $2, NOW())
				ON CONFLICT ON CONSTRAINT uq_product_stock DO UPDATE SET
					quantity = product_stock.quantity + $2,
					updated_at = NOW()
			`, item.productID, qtyGood)
			if err != nil {
				return fmt.Errorf("update product stock: %w", err)
			}

			// Record inventory movement
			_, err = tx.ExecContext(ctx, `
				INSERT INTO inventory_movements
					(product_id, quantity_change, type, reference_id, reference_table, user_id, notes, created_at)
				VALUES ($1, $2, 'purchase_receive', $3, 'goods_receipts', $4, $5, $6)
			`, item.productID, qtyGood, grID, receivedBy,
				fmt.Sprintf("GR %s", grNum), receivedAt)
			if err != nil {
				return fmt.Errorf("insert inventory movement: %w", err)
			}
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit GR: %w", err)
		}
	}

	// Recalculate PO status for all POs that received goods.
	// Without this, POs remain "confirmed" even when items are partially/fully received.
	if _, err := db.ExecContext(ctx, `
		UPDATE purchase_orders po
		SET status = CASE
			WHEN sub.total_received = 0 THEN 'confirmed'
			WHEN sub.total_received >= sub.total_ordered THEN 'fully_received'
			ELSE 'partial_received'
		END,
		updated_at = NOW()
		FROM (
			SELECT purchase_order_id,
			       SUM(qty_ordered) AS total_ordered,
			       SUM(qty_received) AS total_received
			FROM purchase_order_items
			GROUP BY purchase_order_id
		) sub
		WHERE po.id = sub.purchase_order_id
		  AND po.status = 'confirmed'
		  AND sub.total_received > 0
	`); err != nil {
		return fmt.Errorf("recalculate PO statuses: %w", err)
	}

	fmt.Printf("   🎲 Created %d goods receipts for confirmed POs\n", grInserted)
	return nil
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

	// Look up customer group IDs from the database (all groups seeded by ensureCustomerGroups)
	groupIDs := make(map[string]int)
	rows, err := db.QueryContext(ctx, `SELECT id, LOWER(name) FROM customer_groups`)
	if err == nil {
		for rows.Next() {
			var id int
			var gname string
			if err := rows.Scan(&id, &gname); err == nil {
				groupIDs[gname] = id
			}
		}
		_ = rows.Close()
	}

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
		`INSERT INTO customers (name, phone, email, address, note, is_active, is_walk_in, customer_group_id, created_at)
		 VALUES ($1, $2, $3, $4, $5, true, false, $6, $7)`)
	if err != nil {
		return fmt.Errorf("prepare: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	ref := time.Now().In(jakartaTZ)

	// Insert a walk-in/general customer first
	walkInID := 0
	err = tx.QueryRowContext(ctx,
		`INSERT INTO customers (name, phone, email, address, note, is_active, is_walk_in, customer_group_id, created_at)
		 VALUES ('Walk-in / General', '', '', NULL, NULL, true, true, $1, $2)
		 ON CONFLICT (phone) DO NOTHING
		 RETURNING id`,
		nullableInt(groupIDs["walk-in"]),
		ref,
	).Scan(&walkInID)
	if err == sql.ErrNoRows {
		// Already present from a previous run (re-seed); reuse the existing row.
		if err := tx.QueryRowContext(ctx, `SELECT id FROM customers WHERE is_walk_in = true ORDER BY id LIMIT 1`).Scan(&walkInID); err != nil {
			return fmt.Errorf("find existing walk-in customer: %w", err)
		}
	} else if err != nil {
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

		// Assign customer group with realistic distribution
		groupID := 0
		switch r := rand.Intn(100); {
		case r < 35: // 35% Walk-in
			groupID = groupIDs["walk-in"]
		case r < 55: // 20% Member
			groupID = groupIDs["member"]
		case r < 65: // 10% VIP
			groupID = groupIDs["vip"]
		case r < 75: // 10% Reseller
			groupID = groupIDs["reseller"]
		case r < 83: // 8% Corporate
			groupID = groupIDs["corporate"]
		case r < 90: // 7% Student
			groupID = groupIDs["student"]
		case r < 95: // 5% Wholesale
			groupID = groupIDs["wholesale"]
		default: // 5% Online
			groupID = groupIDs["online"]
		}

		if _, err := stmt.ExecContext(ctx, name, phone, email, address, note, nullableInt(groupID), createdAt); err != nil {
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
