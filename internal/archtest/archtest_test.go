package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const modulePrefix = "retail-pos-system/internal/"

// domainModules are the domain modules of the modular monolith. Shared
// infrastructure (audit, config, events, eventbus, middleware, ownership,
// permissions, shared) is intentionally importable from anywhere.
var domainModules = []string{
	"brand", "category", "customer", "customergroup", "inventory",
	"platform", "pricing", "product", "purchase", "report", "sale",
	"shift", "stockopname", "storagelocation", "store", "supplier", "uom", "user",
}

// isolatedModules must not import any other domain module. Cross-module reads
// must go through ports wired in internal/wiring; cross-module effects must go
// through events in internal/events.
var isolatedModules = []string{
	"appsettings", "brand", "category", "customer", "customergroup",
	"inventory", "platform", "pricing", "product", "purchase", "report",
	"sale", "shift", "stockopname", "storagelocation", "store", "supplier",
	"uom", "user",
}

var sqlKeywordRe = regexp.MustCompile(`\b(?:FROM|INTO|UPDATE|JOIN|REFERENCES|TABLE)\s+([a-z_]+)`)

// tableContext assigns every real table to its owning bounded context, per
// ADR Modular_Monolith_Module_Boundaries §2.8.
var tableContext = map[string]string{
	// Transaksional
	"sales":               "transaksional",
	"sale_items":          "transaksional",
	"sale_payments":       "transaksional",
	"product_stock":       "transaksional",
	"inventory_movements": "transaksional",
	"shifts":              "transaksional",
	"payment_methods":     "transaksional",
	"cart_sessions":       "transaksional",
	"cart_items":          "transaksional",
	// Katalog
	"products":          "katalog",
	"categories":        "katalog",
	"brands":            "katalog",
	"units_of_measure":  "katalog",
	"tax_classes":       "katalog",
	"pricing_rules":     "katalog",
	"product_suppliers": "katalog",
	"v_products_full":   "katalog",
	// Prokuremen
	"purchase_orders":      "prokuremen",
	"purchase_order_items": "prokuremen",
	"goods_receipts":       "prokuremen",
	"goods_receipt_items":  "prokuremen",
	// Stock Opname
	"stock_opnames":                 "stockopname",
	"stock_opname_items":            "stockopname",
	"stock_opname_counts":           "stockopname",
	"stock_opname_assignments":      "stockopname",
	"stock_opname_recount_requests": "stockopname",
	"stock_opname_session_scopes":   "stockopname",
	"inventory_adjustments":         "stockopname",
	"inventory_adjustment_items":    "stockopname",
	// Referensi
	"stores":            "referensi",
	"warehouses":        "referensi",
	"storage_locations": "referensi",
	"customers":         "referensi",
	"customer_groups":   "referensi",
	"suppliers":         "referensi",
	// Analitik (read model)
	"mv_daily_sales":  "analitik",
	"mv_hourly_sales": "analitik",
	// Platform (config)
	"app_settings": "platform",
	// Platform
	"users":              "platform",
	"roles":              "platform",
	"permissions":        "platform",
	"role_permissions":   "platform",
	"refresh_tokens":     "platform",
	"audit_logs":         "platform",
	"import_jobs":        "platform",
	"import_snapshots":   "platform",
	"import_rows":        "platform",
	"import_errors":      "platform",
	"outbox":             "platform",
	"dead_letter_events": "platform",
}

// moduleContext maps each domain module to its owning bounded context.
var moduleContext = map[string]string{
	"brand":           "katalog",
	"category":        "katalog",
	"product":         "katalog",
	"uom":             "katalog",
	"pricing":         "katalog",
	"sale":            "transaksional",
	"inventory":       "transaksional",
	"shift":           "transaksional",
	"purchase":        "prokuremen",
	"stockopname":     "stockopname",
	"store":           "referensi",
	"storagelocation": "referensi",
	"customer":        "referensi",
	"customergroup":   "referensi",
	"supplier":        "referensi",
	"report":          "analitik",
	"user":            "platform",
	"platform":        "platform",
	"appsettings":     "platform",
}

// strictModuleTables overrides context ownership for modules already hardened
// to module-level ownership: they may reference ONLY these tables. Modules are
// added here as they are ported to consumer-side ports.
var strictModuleTables = map[string]map[string]bool{
	"category": {
		"categories": true,
	},
	"purchase": {
		"purchase_orders":      true,
		"purchase_order_items": true,
		"goods_receipts":       true,
		"goods_receipt_items":  true,
	},
	"shift": {
		"shifts": true,
	},
	"sale": {
		"sales":           true,
		"sale_items":      true,
		"sale_payments":   true,
		"payment_methods": true,
		"cart_sessions":   true,
		"cart_items":      true,
	},
	"supplier": {
		"suppliers": true,
	},
	"inventory": {
		"product_stock":       true,
		"inventory_movements": true,
	},
	"product": {
		"products":          true,
		"product_suppliers": true,
		"tax_classes":       true,
		"v_products_full":   true,
		"categories":        true,
	},
	"pricing": {
		"pricing_rules": true,
	},
	"customer": {
		"customers": true,
	},
	"brand": {
		"brands": true,
	},
	"uom": {
		"units_of_measure": true,
	},
	"customergroup": {
		"customer_groups": true,
	},
	"stockopname": {
		"stock_opnames":                 true,
		"stock_opname_items":            true,
		"stock_opname_counts":           true,
		"stock_opname_assignments":      true,
		"stock_opname_recount_requests": true,
		"stock_opname_session_scopes":   true,
		"inventory_adjustments":         true,
		"inventory_adjustment_items":    true,
	},
	"store": {
		"stores":     true,
		"warehouses": true,
	},
	"storagelocation": {
		"storage_locations": true,
	},
	"user": {
		"users":            true,
		"roles":            true,
		"permissions":      true,
		"role_permissions": true,
		"refresh_tokens":   true,
		"audit_logs":       true,
	},
	"platform": {
		"import_jobs":        true,
		"import_snapshots":   true,
		"import_rows":        true,
		"import_errors":      true,
		"outbox":             true,
		"dead_letter_events": true,
	},
	"appsettings": {
		"app_settings": true,
	},
}

// crossContextDebt lists acknowledged cross-context references awaiting
// porting (ADR §4 step 3: reads behind Query interfaces, step 4: writes behind
// Application Services). The ownership rule rejects these; the manifest keeps
// CI green while tracking the backlog. Remove an entry once its table access
// is ported.
//
// Empty as of the stockopname snapshot/scope porting: all stockopname reads of
// katalog tables (products, product_suppliers, units_of_measure) and of
// product_stock (snapshots, scope universes, posting locks) are routed through
// owner modules (product.ProductMetaLookup, uom.UnitNameLookup,
// inventory.StockLocker, inventory.StockSnapshotProvider, and the
// ScopeNameResolver/LocationScopeProvider/WarehouseStoreIDProvider ports).
var crossContextDebt = map[string]map[string]bool{}

func nonTestGoFiles(t *testing.T, dir string) []string {
	t.Helper()
	var files []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, "_test.go") || !strings.HasSuffix(path, ".go") {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", dir, err)
	}
	return files
}

func collectImports(t *testing.T, dir string) map[string]bool {
	t.Helper()
	imports := make(map[string]bool)
	for _, file := range nonTestGoFiles(t, dir) {
		node, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		for _, imp := range node.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			if strings.HasPrefix(path, modulePrefix) {
				imports[path] = true
			}
		}
	}
	return imports
}

// knownTables returns the set of real tables/views defined in the applied
// migrations, unioned with the ADR ownership seed. References to names outside
// this set (CTE aliases, subquery labels) are ignored by the ownership check.
func knownTables(t *testing.T) map[string]bool {
	t.Helper()
	out := make(map[string]bool, len(tableContext))
	for table := range tableContext {
		out[table] = true
	}
	createRe := regexp.MustCompile(`(?i)CREATE\s+(?:(?:OR REPLACE|UNLOGGED|MATERIALIZED)\s+)*(?:TABLE|VIEW)\s+(?:IF NOT EXISTS\s+)?(\w+)`)
	files, err := filepath.Glob(filepath.Join("..", "..", "database", "migrations", "*.sql"))
	if err != nil {
		t.Fatalf("glob migrations: %v", err)
	}
	for _, file := range files {
		b, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		for _, m := range createRe.FindAllStringSubmatch(string(b), -1) {
			out[strings.ToLower(m[1])] = true
		}
	}
	return out
}

// collectSQLTableRefs returns, per table, the SQL keywords (FROM/JOIN/INTO/...)
// with which the module's non-test files reference it. Only known tables are
// reported.
func collectSQLTableRefs(t *testing.T, dir string, known map[string]bool) map[string]map[string]bool {
	t.Helper()
	refs := make(map[string]map[string]bool)
	for _, file := range nonTestGoFiles(t, dir) {
		node, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		ast.Inspect(node, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			for _, m := range sqlKeywordRe.FindAllStringSubmatch(lit.Value, -1) {
				table := strings.ToLower(m[1])
				if !known[table] {
					continue
				}
				kw := strings.Fields(m[0])[0]
				if refs[table] == nil {
					refs[table] = make(map[string]bool)
				}
				refs[table][kw] = true
			}
			return true
		})
	}
	return refs
}

func isReadKeyword(kw string) bool {
	return kw == "FROM" || kw == "JOIN"
}

// violatedTables returns the tables a module references that the ownership
// rule rejects (strict module ownership first, then context ownership with
// read-model reads allowed).
func violatedTables(module string, refs map[string]map[string]bool) map[string]bool {
	out := make(map[string]bool)
	if strict, ok := strictModuleTables[module]; ok {
		for table := range refs {
			if !strict[table] {
				out[table] = true
			}
		}
		return out
	}
	mctx := moduleContext[module]
	for table, kws := range refs {
		tctx, owned := tableContext[table]
		if !owned {
			out[table] = true
			continue
		}
		for kw := range kws {
			if tctx == mctx {
				continue
			}
			if mctx == "analitik" && isReadKeyword(kw) {
				continue
			}
			if tctx == "analitik" && isReadKeyword(kw) {
				continue
			}
			out[table] = true
			break
		}
	}
	return out
}

func TestModuleImportBoundaries(t *testing.T) {
	for _, module := range isolatedModules {
		imports := collectImports(t, filepath.Join("..", module))
		for _, other := range domainModules {
			if other == module {
				continue
			}
			forbidden := modulePrefix + other
			if imports[forbidden] {
				t.Errorf("%s must not import %s", module, forbidden)
			}
		}
	}
}

func TestModuleSQLTableOwnership(t *testing.T) {
	known := knownTables(t)
	refsByModule := make(map[string]map[string]map[string]bool, len(domainModules))
	for _, module := range domainModules {
		refsByModule[module] = collectSQLTableRefs(t, filepath.Join("..", module), known)
	}

	violated := make(map[string]map[string]bool, len(domainModules))
	for _, module := range domainModules {
		violated[module] = violatedTables(module, refsByModule[module])
		for table := range violated[module] {
			if crossContextDebt[module][table] {
				continue
			}
			t.Errorf("%s references table %q which it does not own (see internal/archtest crossContextDebt for acknowledged backlog)", module, table)
		}
	}

	for module, tables := range crossContextDebt {
		for table := range tables {
			if !violated[module][table] {
				t.Errorf("stale crossContextDebt entry: %s no longer references %q — remove the entry", module, table)
			}
		}
	}
}
