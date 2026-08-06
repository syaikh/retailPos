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
	"brand", "category", "customer", "customergroup", "inventory",
	"platform", "pricing", "product", "purchase", "report", "sale", "shift",
	"stockopname", "storagelocation", "store", "supplier", "uom", "user",
}

var sqlKeywordRe = regexp.MustCompile(`\b(?:FROM|INTO|UPDATE|JOIN|REFERENCES|TABLE)\s+([a-z_]+)`)

var moduleTableAllowlist = map[string]map[string]bool{
	"purchase": {
		"purchase_orders":      true,
		"purchase_order_items": true,
		"goods_receipts":       true,
		"goods_receipt_items":  true,
	},
}

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

func collectSQLTables(t *testing.T, dir string) map[string]bool {
	t.Helper()
	tables := make(map[string]bool)
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
				tables[strings.ToLower(m[1])] = true
			}
			return true
		})
	}
	return tables
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
	for module, allowlist := range moduleTableAllowlist {
		tables := collectSQLTables(t, filepath.Join("..", module))
		for table := range tables {
			if !allowlist[table] {
				t.Errorf("%s references table %q which it does not own", module, table)
			}
		}
	}
}
