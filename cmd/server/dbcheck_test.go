//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"os"
	"retail-pos-system/internal/shared"
)

func main() {
	dsn := shared.GetTestDSN()
	fmt.Println("DSN:", dsn)
	pool, err := shared.NewTestDB()
	if err != nil {
		fmt.Println("ERROR:", err)
		os.Exit(1)
	}
	defer pool.Close()
	fmt.Println("OK: connected")
}
