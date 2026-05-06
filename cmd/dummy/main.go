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
	// Parse command line flags
	truncateFlag := flag.Bool("truncate", true, "Truncate existing data before injection")
	productsFlag := flag.Int("products", 4500, "Number of products to generate (4000-5000)")
	salesFlag := flag.Int("sales", 4500, "Number of sales transactions to generate (4000-5000)")
	categoriesFlag := flag.Int("categories", 65, "Number of categories to ensure exist (50-80)")
	flag.Parse()

	if err := run(*truncateFlag, *productsFlag, *salesFlag, *categoriesFlag); err != nil {
		log.Fatalf("Dummy seeder failed: %v", err)
	}
}

func run(truncateData bool, numProducts, numSales, numCategories int) error {
	ctx := context.Background()
	rand.Seed(time.Now().UnixNano())

	// Validate parameters (relaxed for continuation)
	if numProducts < 0 || numProducts > 5000 {
		return fmt.Errorf("products count must be between 0-5000, got %d", numProducts)
	}
	if numSales < 0 || numSales > 5000 {
		return fmt.Errorf("sales count must be between 0-5000, got %d", numSales)
	}
	if numCategories < 0 || numCategories > 80 {
		return fmt.Errorf("categories count must be between 0-80, got %d", numCategories)
	}

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
	fmt.Printf("Target: %d products, %d sales, %d categories\n", numProducts, numSales, numCategories)

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

	// 3. Get users for cashier assignment (only needed for sales)
	var userIDs []int
	if numSales > 0 {
		userIDs = getIDs(ctx, db, "users")
		if len(userIDs) == 0 {
			return fmt.Errorf("no users found. Please run migrations/seeds first")
		}
	}

	// 4. Inject products (4000-5000, spanning 5-6 months)
	var productData []ProductInfo
	if numProducts > 0 {
		fmt.Printf("📦 Injecting %d products (5-6 months span)...\n", numProducts)
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

	// 5. Inject sales transactions (4000-5000, spanning 5-6 months)
	if numSales > 0 {
		fmt.Printf("💰 Injecting %d sales transactions (5-6 months span)...\n", numSales)
		if err := injectSales(ctx, db, userIDs, productData, numSales); err != nil {
			return fmt.Errorf("failed to inject sales: %w", err)
		}
		fmt.Printf("   ✅ %d sales transactions injected\n", numSales)
	}

	fmt.Println("🎉 Dummy data injection completed successfully!")
	fmt.Printf("   📊 Summary: %d products, %d sales, %d categories\n", len(productData), numSales, len(categoryIDs))
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
		WHERE p.is_active = true
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

// injectProducts generates 4000-5000 realistic products across 5-6 months
func injectProducts(ctx context.Context, db *sql.DB, categoryIDs []int, count int) ([]ProductInfo, error) {
	products := make([]ProductInfo, 0, count)

	// Pre-fetch category names for product generation
	categoryNames := make(map[int]string)
	for _, catID := range categoryIDs {
		var name string
		err := db.QueryRowContext(ctx, "SELECT name FROM categories WHERE id = $1", catID).Scan(&name)
		if err == nil {
			categoryNames[catID] = name
		}
	}

	for i := 0; i < count; i++ {
		catID := randElemInt(categoryIDs)
		catName := categoryNames[catID]

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
		if i < count*lowStockPercentage/100 {
			// Low stock: 0 to stock_min
			stock = rand.Intn(stockMin + 1)
		} else {
			// Normal stock distribution
			stock = generateStockLevel(catName)
		}

	// Random date within last month (evenly distributed across the period)
	randomDays := rand.Intn(30) + 1 // 1-30 days ago (1 month span)
	createdAt := time.Now().AddDate(0, 0, -randomDays)

		var id int
		err := db.QueryRowContext(ctx,
			`INSERT INTO products (sku, name, barcode, price, cost, stock, stock_min, stock_max, category_id, is_active, created_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, true, $10) RETURNING id`,
			sku, name, barcode, price, cost, stock, stockMin, stock*2, catID, createdAt,
		).Scan(&id)

		if err != nil {
			fmt.Printf("Warning: failed to insert product %d: %v\n", i, err)
			continue
		}

		products = append(products, ProductInfo{
			ID:       id,
			Price:    price,
			Category: catName,
		})

		if i%500 == 0 && i > 0 {
			fmt.Printf("     ...%d products injected\n", i)
		}
	}

	return products, nil
}

// injectSales generates transactions ensuring every day has at least 10 transactions across 5-6 months
func injectSales(ctx context.Context, db *sql.DB, userIDs []int, products []ProductInfo, count int) error {
	now := time.Now()
	productMap := make(map[int]ProductInfo)
	for _, p := range products {
		productMap[p.ID] = p
	}

	// Calculate date range for last month (always span full period)
	startDate := time.Now().AddDate(0, 0, -30) // 30 days ago
	endDate := time.Now()                       // now
	totalDays := int(endDate.Sub(startDate).Hours() / 24) // Actual days in period (~30 days)
	minSalesPerDay := 10

	// Distribute sales across the full date range
	// Allow flexible sales per day to ensure full period coverage
	baseSalesPerDay := count / totalDays
	extraSales := count % totalDays

	// Ensure minimum distribution
	if baseSalesPerDay == 0 {
		baseSalesPerDay = 1
		totalDays = count // Each day gets 1 sale
	}

	salesCreated := 0

	for day := 0; day < totalDays && salesCreated < count; day++ {
		// Calculate remaining sales needed
		remainingNeeded := count - salesCreated

		// Base sales for this day
		salesForDay := baseSalesPerDay

		// Distribute extra sales across days
		if day < extraSales {
			salesForDay += 1
		}

		// Add some randomness to sales distribution
		if salesForDay == 0 && rand.Intn(10) < 3 { // 30% chance of 1 sale on empty days
			salesForDay = 1
		}

		// Ensure we don't exceed remaining sales needed
		if salesForDay > remainingNeeded {
			salesForDay = remainingNeeded
		}

		// Skip days with no sales
		if salesForDay <= 0 {
			continue
		}

		// Ensure minimum of 10 transactions per day
		if salesForDay < minSalesPerDay {
			salesForDay = minSalesPerDay
		}

		// Ensure we don't exceed remaining sales needed
		if salesForDay > remainingNeeded {
			salesForDay = remainingNeeded
		}

		// Skip days with no sales (shouldn't happen with our logic)
		if salesForDay <= 0 {
			continue
		}

		for s := 0; s < salesForDay && salesCreated < count; s++ {
			// Generate invoice number
			invoice := fmt.Sprintf("INV-%d-%06d", now.Year(), salesCreated+1)

			// Random cashier
			cashierID := randElemInt(userIDs)

			// Random payment method (weighted)
			paymentMethod := weightedRandomChoice(paymentMethods, paymentWeights)

			// Calculate transaction time for this day
			dayDate := startDate.AddDate(0, 0, day)

			var createdAt time.Time
			if dayDate.Year() == now.Year() && dayDate.Month() == now.Month() && dayDate.Day() == now.Day() {
				// Today: ensure transaction time is before current time
				maxHour := now.Hour()
				maxMinute := now.Minute()
				if maxHour < 8 {
					maxHour = 8 // Minimum 8 AM
				}
				randomHour := 8 + rand.Intn(maxHour-7) // 8 AM to current hour
				randomMinute := rand.Intn(60)
				if randomHour == maxHour && randomMinute >= maxMinute {
					randomMinute = maxMinute - 1
					if randomMinute < 0 {
						randomMinute = 0
						randomHour--
						if randomHour < 8 {
							randomHour = 8
						}
					}
				}
				createdAt = time.Date(now.Year(), now.Month(), now.Day(),
					randomHour, randomMinute, rand.Intn(60), 0, time.UTC)
			} else {
				// Other days: random hour between 8 AM and 8 PM
				randomHour := 8 + rand.Intn(12)
				randomMinute := rand.Intn(60)
				createdAt = time.Date(dayDate.Year(), dayDate.Month(), dayDate.Day(),
					randomHour, randomMinute, rand.Intn(60), 0, time.UTC)
			}

			// Create sale record
			var saleID int
			err := db.QueryRowContext(ctx,
				`INSERT INTO sales (invoice_number, cashier_id, payment_method, status, created_at)
				 VALUES ($1, $2, $3, 'completed', $4) RETURNING id`,
				invoice, cashierID, paymentMethod, createdAt,
			).Scan(&saleID)

			if err != nil {
				fmt.Printf("Warning: failed to insert sale %d: %v\n", salesCreated, err)
				continue
			}

			// Generate sale items (1-8 items, realistic distribution)
			numItems := generateItemCount()
			totalAmount := 0

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

				_, err = db.ExecContext(ctx,
					`INSERT INTO sale_items (sale_id, product_id, quantity, unit_price, subtotal)
					 VALUES ($1, $2, $3, $4, $5)`,
					saleID, productID, quantity, unitPrice, subtotal,
				)

				if err != nil {
					fmt.Printf("Warning: failed to insert sale item for sale %d: %v\n", saleID, err)
				}
			}

			// Ensure transaction has positive value (skip if no items or zero total)
			if totalAmount > 0 {
				// Update sale totals
				_, err = db.ExecContext(ctx,
					"UPDATE sales SET subtotal = $1, total_amount = $1 WHERE id = $2",
					totalAmount, saleID)

				salesCreated++

				if salesCreated%500 == 0 && salesCreated > 0 {
					fmt.Printf("     ...%d sales injected\n", salesCreated)
				}
			} else {
				// Remove the sale if it has no valid items
				_, err = db.ExecContext(ctx, "DELETE FROM sales WHERE id = $1", saleID)
				if err != nil {
					fmt.Printf("Warning: failed to remove invalid sale %d: %v\n", saleID, err)
				}
			}
		}
	}

	fmt.Printf("   ✅ %d sales transactions injected across %d days (min 10 per day)\n", salesCreated, totalDays)
	return nil
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
	// Generate EAN-13 or custom barcode (6-13 characters)
	// For simplicity, we'll generate custom barcodes of varying lengths

	// 70% chance of EAN-13 (13 digits), 30% chance of custom (6-12 digits)
	if rand.Intn(100) < 70 {
		// EAN-13: 13 digits
		return fmt.Sprintf("%013d", rand.Int63n(9999999999999))
	} else {
		// Custom barcode: 6-12 characters (mix of letters and numbers)
		length := 6 + rand.Intn(7) // 6-12 characters
		chars := "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
		barcode := make([]byte, length)
		for i := range barcode {
			barcode[i] = chars[rand.Intn(len(chars))]
		}
		return string(barcode)
	}
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