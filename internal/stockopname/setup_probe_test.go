package stockopname

import (
	"fmt"
	"testing"

	"retail-pos-system/internal/shared"
)

func TestSetupProbe(t *testing.T) {
	pool, err := shared.NewTestDB()
	if err != nil {
		t.Fatalf("NewTestDB err: %v", err)
	}
	defer pool.Close()
	if err := shared.RunMigrations(pool, "../../database/migrations"); err != nil {
		t.Fatalf("RunMigrations err: %v", err)
	}
	if err := shared.TruncateTestData(pool); err != nil {
		t.Fatalf("TruncateTestData err: %v", err)
	}
	fmt.Println("setup ok")
}
