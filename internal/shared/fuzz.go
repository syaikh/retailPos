package shared

import (
	"os"
	"strings"
)

// FuzzTestMode reports whether the test binary runs as a fuzz coordinator
// (-test.fuzz=...) or a fuzz worker (-test.fuzzworker).
//
// Every fuzz worker re-executes TestMain, so a package whose TestMain needs
// PostgreSQL must skip its database setup in fuzz mode: concurrent
// RunMigrations/TruncateTestData between workers contend on the same database
// and the workers die, which the coordinator reports as "fuzzing process
// terminated ... EOF". Skipping the database also lets fuzz targets run in CI
// jobs without a PostgreSQL service.
//
// Because the pool is skipped, a fuzz invocation must also pass -run matching
// the target (as .github/workflows/fuzz.yml does): the coordinator runs the
// package's regular tests unless -run filters them out, and the
// database-backed ones would otherwise panic on the nil pool.
func FuzzTestMode() bool {
	for _, arg := range os.Args {
		if strings.HasPrefix(arg, "-test.fuzz") {
			return true
		}
	}
	return false
}
