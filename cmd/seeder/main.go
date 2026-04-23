package main

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

// ─────────────────────────────── helpers ────────────────────────────────────

func randInt(min, max int) int {
	if max <= min {
		return min
	}
	return rand.Intn(max-min) + min
}

// generateBarcode produces a valid-looking 13-digit EAN barcode.
// Prefix 899 = Indonesia, 8996 = common Indonesian FMCG prefix.
func generateBarcode(seq int) string {
	// Mix Indonesian-style (899...) and international-style prefixes for variety
	prefixes := []string{"8996", "7622", "0737", "5000", "4902", "8991", "3017"}
	pfx := prefixes[rand.Intn(len(prefixes))]
	// Pad the rest so total digits = 13
	rest := 13 - len(pfx)
	num := fmt.Sprintf("%0*d", rest, (seq*7919+rand.Intn(999))%int(pow10(rest)))
	return pfx + num
}

func pow10(n int) int64 {
	v := int64(1)
	for range n {
		v *= 10
	}
	return v
}

// ─────────────────────────────── data pools ─────────────────────────────────

var groupDefs = []struct{ name, desc string }{
	{"Minuman Kemasan", "Minuman dalam kemasan botol, kaleng, atau karton"},
	{"Minuman Segar", "Jus, minuman buah segar, dan infused water"},
	{"Minuman Berenergi", "Minuman energi dan suplemen cair"},
	{"Kopi & Teh", "Kopi, teh, dan minuman berbasis kafein"},
	{"Susu & Dairy", "Susu segar, UHT, yoghurt, dan produk dairy"},
	{"Makanan Ringan", "Snack, keripik, dan camilan"},
	{"Biskuit & Kue", "Biskuit, crackers, cookies, dan kue kering"},
	{"Coklat & Permen", "Coklat, permen, dan kembang gula"},
	{"Mie Instan", "Mie instan berbagai merek dan rasa"},
	{"Makanan Kaleng", "Sardine, kornet, dan makanan kaleng lainnya"},
	{"Bumbu Dapur", "Kecap, saos, bumbu masak, dan rempah"},
	{"Minyak & Lemak", "Minyak goreng, mentega, dan margarin"},
	{"Tepung & Bahan Kue", "Tepung terigu, gula, garam, dan bahan kue"},
	{"Beras & Serealia", "Beras, oatmeal, sereal sarapan"},
	{"Sembako Pokok", "Kebutuhan sembilan bahan pokok"},
	{"Perawatan Kulit", "Sabun, pelembab, dan lotion"},
	{"Perawatan Rambut", "Shampoo, kondisioner, dan hair treatment"},
	{"Perawatan Gigi", "Pasta gigi, sikat gigi, dan obat kumur"},
	{"Kosmetik", "Lipstik, bedak, blush on, dan makeup"},
	{"Parfum & Deodorant", "Parfum, body spray, dan deodorant"},
	{"Obat-obatan OTC", "Obat bebas: parasetamol, antasida, vitamin"},
	{"Suplemen & Vitamin", "Vitamin C, multivitamin, dan suplemen kesehatan"},
	{"Perawatan Luka", "Plester, antiseptik, dan perban"},
	{"Alat Tulis Kantor", "Bolpoin, pensil, buku tulis, dan kertas"},
	{"Perlengkapan Sekolah", "Penggaris, penghapus, tas, dan tempat pensil"},
	{"Printer & Tinta", "Cartridge tinta dan kertas printer"},
	{"Elektronik Kecil", "Baterai, adaptor, kabel, dan aksesori elektronik"},
	{"Lampu & Listrik", "Lampu LED, sekring, dan perlengkapan listrik"},
	{"Peralatan Dapur", "Spatula, wajan, panci, dan peralatan masak"},
	{"Peralatan Makan", "Piring, gelas, sendok, garpu, dan mangkok"},
	{"Perlengkapan Mandi", "Handuk, sabun mandi, dan aksesoris kamar mandi"},
	{"Perlengkapan Tidur", "Bantal, guling, dan sprei"},
	{"Produk Bayi", "Susu formula, popok, dan perlengkapan bayi"},
	{"Mainan Anak", "Mainan edukatif dan hiburan anak"},
	{"Tas & Dompet", "Tas belanja, dompet, dan aksesoris"},
	{"Pakaian Dalam", "Kaos dalam, celana dalam, dan kaus kaki"},
	{"Aksesoris Fashion", "Ikat pinggang, jam tangan, dan perhiasan"},
	{"Sepatu & Sandal", "Sandal jepit, sepatu olahraga, dan sandal rumah"},
	{"Perlengkapan Olahraga", "Bola, raket, peluit, dan aksesoris olahraga"},
	{"Pembersih Lantai", "Pel, sapu, dan cairan pembersih lantai"},
	{"Deterjen & Laundry", "Deterjen cuci, softener, dan pewangi pakaian"},
	{"Pembersih Dapur", "Sabun cuci piring, spons, dan pembersih kompor"},
	{"Insektisida & Pestisida", "Obat nyamuk, baygon, dan anti serangga"},
	{"Pot & Tanaman", "Pot bunga, tanah, dan aksesoris tanaman"},
	{"Hewan Peliharaan", "Makanan kucing, anjing, dan aksesoris hewan"},
	{"Bahan Bangunan Kecil", "Lem, cat tembok kecil, dan paku"},
	{"Peralatan Tangan", "Obeng, palu, tang, dan kunci pas"},
	{"Plastik & Kemasan", "Kantong plastik, toples, dan wadah makanan"},
	{"Kartu & Pulsa", "Pulsa listrik, voucher game, dan e-money"},
	{"Lain-lain", "Produk yang tidak termasuk kategori lain"},
}

var brandsByGroup = map[int][]string{
	0:  {"Aqua", "Club", "Pristine", "Le Minerale", "Ades", "Vit"},
	1:  {"Buavita", "Pulpy", "Minute Maid", "Ceres", "Nu Green Tea"},
	2:  {"Extra Joss", "Kratingdaeng", "M-150", "Hemaviton Jreng", "Cobra"},
	3:  {"Kapal Api", "Good Day", "Nescafe", "Tora Bika", "Sariwangi", "Teh Botol"},
	4:  {"Indomilk", "Frisian Flag", "Ultra", "Anlene", "Milo", "Yakult"},
	5:  {"Chitato", "Lays", "Pringles", "Qtela", "Taro", "Cheetos"},
	6:  {"Roma", "Oreo", "Monde", "Khong Guan", "Julie's", "Ritz"},
	7:  {"Kit Kat", "Silverqueen", "Cadbury", "Toblerone", "Kopiko", "Mentos"},
	8:  {"Indomie", "Mie Sedaap", "Supermi", "Sarimi", "Gaga"},
	9:  {"Botan", "ABC", "Maya", "Ayam Brand", "Bernardi"},
	10: {"Indofood", "Heinz", "ABC Kecap", "Masako", "Royco", "Maggi"},
	11: {"Bimoli", "Sania", "Sunco", "Rose Brand", "Filma"},
	12: {"Segitiga Biru", "Cakra Kembar", "Gulaku", "Refina", "Cap Gula"},
	13: {"Kepala Dua", "Pandan Wangi", "Rojolele", "Piala", "Quaker"},
	14: {"Bogasari", "Indofood", "ABC", "Nescafe", "Masako"},
}

var adjectives = []string{
	"Premium", "Ekonomis", "Spesial", "Original", "Classic",
	"Fresh", "New", "Jumbo", "Mini", "Hemat",
	"Super", "Gold", "Lite", "Plus", "Pro",
	"Deluxe", "Ultra", "Vario", "Turbo", "Mega",
}

var productSuffixes = []string{
	"200ml", "500ml", "1L", "1.5L", "2L",
	"100g", "200g", "250g", "500g", "1kg",
	"3in1", "Sachet", "Refill", "Pack", "Box",
	"Reguler", "Large", "Small", "Family", "Single",
}

var paymentMethods = []string{"cash", "card"}

// ─────────────────────────────── main ───────────────────────────────────────

func main() {
	rand.Seed(time.Now().UnixNano())

	// Load .env
	cwd, _ := os.Getwd()
	if err := godotenv.Load(filepath.Join(cwd, ".env")); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"), os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("open db:", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("ping db:", err)
	}
	fmt.Println("✔ Connected to database")

	// ── 1. Truncate ─────────────────────────────────────────────────────────
	fmt.Println("→ Cleaning existing data...")
	_, err = db.Exec(`TRUNCATE sale_items, sales, products, product_groups RESTART IDENTITY CASCADE`)
	if err != nil {
		log.Fatal("truncate:", err)
	}
	fmt.Println("  ✔ Tables cleared")

	// ── 2. Product Groups / Categories (50+) ────────────────────────────────
	fmt.Printf("→ Inserting %d categories...\n", len(groupDefs))
	groupIDs := make([]int, 0, len(groupDefs))
	for _, g := range groupDefs {
		var id int
		err := db.QueryRow(
			`INSERT INTO product_groups (name, description) VALUES ($1, $2) RETURNING id`,
			g.name, g.desc,
		).Scan(&id)
		if err != nil {
			log.Fatal("insert group:", err)
		}
		groupIDs = append(groupIDs, id)
	}
	fmt.Printf("  ✔ %d categories inserted\n", len(groupIDs))

	// ── 3. Products (1000+) ─────────────────────────────────────────────────
	const totalProducts = 1050
	const lowStockTarget = 60 // products that will have stock ≤ 5
	fmt.Printf("→ Inserting %d products...\n", totalProducts)

	type Product struct {
		id    int
		price int
		name  string
	}
	products := make([]Product, 0, totalProducts)
	barcodeSet := make(map[string]bool) // uniqueness guard
	skuSet := make(map[string]bool)

	for i := 1; i <= totalProducts; i++ {
		gIdx := rand.Intn(len(groupIDs))
		groupID := groupIDs[gIdx]

		// Pick a brand from the group's pool (fallback to generic)
		var brand string
		if pool, ok := brandsByGroup[gIdx]; ok {
			brand = pool[rand.Intn(len(pool))]
		} else {
			brand = adjectives[rand.Intn(len(adjectives))]
		}

		adj := adjectives[rand.Intn(len(adjectives))]
		sfx := productSuffixes[rand.Intn(len(productSuffixes))]
		name := fmt.Sprintf("%s %s %s", brand, adj, sfx)

		// Prices in tiers to reflect realistic retail
		priceOptions := []int{
			randInt(500, 3000),
			randInt(3000, 15000),
			randInt(15000, 50000),
			randInt(50000, 250000),
		}
		priceRaw := priceOptions[rand.Intn(len(priceOptions))]
		price := (priceRaw / 100) * 100
		if price < 500 {
			price = 500
		}

		// Stock: first lowStockTarget products get very low stock, rest normal
		var stock int
		if i <= lowStockTarget {
			stock = randInt(0, 6) // 0–5 = low stock
		} else {
			stock = randInt(10, 300)
		}

		// SKU - guaranteed unique
		sku := fmt.Sprintf("SKU-%05d", i)
		for skuSet[sku] {
			sku = fmt.Sprintf("SKU-%05d-X%d", i, rand.Intn(9999))
		}
		skuSet[sku] = true

		// Barcode: 90% of products get one
		var barcodeArg interface{}
		if rand.Float64() < 0.90 {
			var bc string
			for {
				bc = generateBarcode(i*1000 + rand.Intn(9999))
				if !barcodeSet[bc] {
					break
				}
			}
			barcodeSet[bc] = true
			barcodeArg = bc
		} else {
			barcodeArg = nil
		}

		var id int
		err := db.QueryRow(
			`INSERT INTO products (name, sku, barcode, price, stock, group_id, store_id)
			 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`,
			name, sku, barcodeArg, price, stock, groupID, 1,
		).Scan(&id)
		if err != nil {
			log.Fatalf("insert product %d: %v", i, err)
		}
		products = append(products, Product{id, price, name})

		if i%200 == 0 {
			fmt.Printf("  ... %d products inserted\n", i)
		}
	}
	fmt.Printf("  ✔ %d products inserted (first %d have low stock)\n", totalProducts, lowStockTarget)

	// ── 4. Cashier IDs ──────────────────────────────────────────────────────
	// Collect existing user IDs from the users table
	rows, err := db.Query(`SELECT id FROM users ORDER BY id`)
	if err != nil {
		log.Fatal("query users:", err)
	}
	var cashierIDs []int
	for rows.Next() {
		var uid int
		rows.Scan(&uid)
		cashierIDs = append(cashierIDs, uid)
	}
	rows.Close()

	if len(cashierIDs) == 0 {
		log.Fatal("No users found – run 'go run ./cmd/seed' first to create admin/cashier users")
	}

	// ── 5. Sales + Sale Items (500+ transactions) ───────────────────────────
	const totalSales = 550
	fmt.Printf("→ Generating %d sales transactions (last 180 days)...\n", totalSales)

	now := time.Now()
	daySpread := 180

	for saleIdx := 0; saleIdx < totalSales; saleIdx++ {
		// Random date in past daySpread days
		daysBack := randInt(0, daySpread)
		txDate := now.AddDate(0, 0, -daysBack)
		txTime := time.Date(
			txDate.Year(), txDate.Month(), txDate.Day(),
			randInt(7, 22), randInt(0, 60), randInt(0, 60), 0,
			txDate.Location(),
		)

		cashierID := cashierIDs[rand.Intn(len(cashierIDs))]
		payMethod := paymentMethods[rand.Intn(len(paymentMethods))]

		// Items per transaction: 1–6
		numItems := randInt(1, 7)

		// Pick unique products for this transaction
		chosen := make(map[int]bool)
		type SaleItem struct {
			productID int
			name      string
			price     int
			qty       int
		}
		var saleItems []SaleItem

		attempts := 0
		for len(saleItems) < numItems && attempts < numItems*5 {
			attempts++
			p := products[rand.Intn(len(products))]
			if chosen[p.id] {
				continue
			}
			chosen[p.id] = true
			qty := randInt(1, 5)
			// Apply occasional discount rounding to simulate real retail
			salePrice := p.price
			if rand.Float64() < 0.1 {
				// 10% chance of slight discount
				salePrice = (salePrice * randInt(80, 99) / 100 / 100) * 100
				if salePrice < 100 {
					salePrice = 100
				}
			}
			saleItems = append(saleItems, SaleItem{p.id, p.name, salePrice, qty})
		}

		if len(saleItems) == 0 {
			continue
		}

		totalAmount := 0
		for _, si := range saleItems {
			totalAmount += si.price * si.qty
		}

		var saleID int
		err := db.QueryRow(
			`INSERT INTO sales (total_amount, payment_method, cashier_id, created_at)
			 VALUES ($1, $2, $3, $4) RETURNING id`,
			totalAmount, payMethod, cashierID, txTime,
		).Scan(&saleID)
		if err != nil {
			log.Fatalf("insert sale %d: %v", saleIdx, err)
		}

		for _, si := range saleItems {
			_, err = db.Exec(
				`INSERT INTO sale_items (sale_id, product_id, product_name, quantity, price_at_sale)
				 VALUES ($1, $2, $3, $4, $5)`,
				saleID, si.productID, si.name, si.qty, si.price,
			)
			if err != nil {
				log.Fatalf("insert sale_item for sale %d: %v", saleID, err)
			}
		}

		if (saleIdx+1)%100 == 0 {
			fmt.Printf("  ... %d sales inserted\n", saleIdx+1)
		}
	}

	// ── 6. Summary ──────────────────────────────────────────────────────────
	var (
		catCount      int
		productCount  int
		barcodeCount  int
		lowStockCount int
		saleCount     int
		saleItemCount int
	)
	db.QueryRow(`SELECT COUNT(*) FROM product_groups`).Scan(&catCount)
	db.QueryRow(`SELECT COUNT(*) FROM products`).Scan(&productCount)
	db.QueryRow(`SELECT COUNT(*) FROM products WHERE barcode IS NOT NULL`).Scan(&barcodeCount)
	db.QueryRow(`SELECT COUNT(*) FROM products WHERE stock <= 5`).Scan(&lowStockCount)
	db.QueryRow(`SELECT COUNT(*) FROM sales`).Scan(&saleCount)
	db.QueryRow(`SELECT COUNT(*) FROM sale_items`).Scan(&saleItemCount)

	fmt.Println()
	fmt.Println("═══════════════════════════════════")
	fmt.Println("        SEEDER SUMMARY             ")
	fmt.Println("═══════════════════════════════════")
	fmt.Printf("  Categories     : %d\n", catCount)
	fmt.Printf("  Products       : %d\n", productCount)
	fmt.Printf("  With barcode   : %d (%.1f%%)\n", barcodeCount, float64(barcodeCount)/float64(productCount)*100)
	fmt.Printf("  Low stock (≤5) : %d\n", lowStockCount)
	fmt.Printf("  Sales          : %d\n", saleCount)
	fmt.Printf("  Sale items     : %d\n", saleItemCount)
	fmt.Println("═══════════════════════════════════")

	// Validate requirements
	ok := true
	check := func(label string, got, want int) {
		if got < want {
			fmt.Printf("  ✘ %s: got %d, want >= %d\n", label, got, want)
			ok = false
		} else {
			fmt.Printf("  ✔ %s: %d\n", label, got)
		}
	}
	fmt.Println("\n  Requirement Check:")
	check("Categories >= 50", catCount, 50)
	check("Products >= 1000", productCount, 1000)
	check("Low stock >= 50", lowStockCount, 50)
	check("Sales >= 500", saleCount, 500)

	if ok {
		fmt.Println("\n  ✔ All requirements met!")
	} else {
		fmt.Println("\n  ✘ Some requirements not met – please check seeder.")
	}
	_ = strings.TrimSpace("") // keep import
}
