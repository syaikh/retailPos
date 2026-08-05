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

var forbiddenImports = map[string][]string{
	"purchase": {
		modulePrefix + "brand",
		modulePrefix + "category",
		modulePrefix + "customer",
		modulePrefix + "customergroup",
		modulePrefix + "inventory",
		modulePrefix + "pricing",
		modulePrefix + "product",
		modulePrefix + "report",
		modulePrefix + "sale",
		modulePrefix + "shift",
		modulePrefix + "stockopname",
		modulePrefix + "storagelocation",
		modulePrefix + "store",
		modulePrefix + "supplier",
		modulePrefix + "uom",
	},
	"inventory": {
		modulePrefix + "purchase",
	},
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
	for module, forbidden := range forbiddenImports {
		imports := collectImports(t, filepath.Join("..", module))
		for _, path := range forbidden {
			if imports[path] {
				t.Errorf("%s must not import %s", module, path)
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
