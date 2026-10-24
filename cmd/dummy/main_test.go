package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestGuardNotProduction re-executes this test binary as a child process: the
// guard refuses by calling log.Fatalf, and an os.Exit can only be observed
// through the subprocess's exit status and stderr. Inside the child,
// DUMMY_GUARD_HELPER turns the test into a bare invocation of the guard.
//
// It pins the same three checks as scripts/reset-dev-db.sh and the Makefile
// `clean` target (cmd/dummy/main.go guardNotProduction).
func TestGuardNotProduction(t *testing.T) {
	if os.Getenv("DUMMY_GUARD_HELPER") == "1" {
		guardNotProduction()
		os.Exit(0) // reached only when the guard lets the seeder proceed
	}

	// run invokes the guard in a subprocess. The appended values win because
	// os/exec keeps the last occurrence of a duplicate environment key, so a
	// developer shell exporting ENV/DB_* cannot change the outcome: every case
	// starts from a neutralized environment and sets only its own variables.
	run := func(extraEnv ...string) (string, error) {
		cmd := exec.Command(os.Args[0], "-test.run=^TestGuardNotProduction$") //nolint:gosec // G204: re-execs this test binary itself; os.Args[0] is not untrusted input
		cmd.Env = append(os.Environ(), "DUMMY_GUARD_HELPER=1",
			"ENV=", "DATABASE_URL=", "DB_HOST=", "DB_NAME=")
		cmd.Env = append(cmd.Env, extraEnv...)
		out, err := cmd.CombinedOutput()
		return string(out), err
	}

	tests := []struct {
		name    string
		env     []string
		wantRef bool
		wantMsg string
	}{
		{
			name:    "ENV=production refuses",
			env:     []string{"ENV=production"},
			wantRef: true,
			wantMsg: "refuses to run with ENV=production",
		},
		{
			name:    "non-localhost DB_HOST refuses",
			env:     []string{"DB_HOST=db.prod.example.com"},
			wantRef: true,
			wantMsg: "non-localhost DB_HOST",
		},
		{
			name:    "DB_NAME other than retail_pos refuses",
			env:     []string{"DB_NAME=staging_db"},
			wantRef: true,
			wantMsg: "refuses to run against DB_NAME",
		},
		{
			name: "DATABASE_URL bypasses the DB_* checks",
			env: []string{
				"DATABASE_URL=postgres://user@db.prod.example.com:5432/x",
				"DB_HOST=db.prod.example.com",
				"DB_NAME=staging_db",
			},
			wantRef: false,
		},
		{
			name:    "dev defaults proceed",
			env:     nil,
			wantRef: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := run(tt.env...)
			if !tt.wantRef {
				if err != nil {
					t.Fatalf("guard should let the dev seeder proceed, got: %v\n%s", err, out)
				}
				return
			}
			if err == nil {
				t.Fatalf("guard should refuse, but the subprocess exited 0:\n%s", out)
			}
			if !strings.Contains(out, tt.wantMsg) {
				t.Fatalf("want refusal message %q in subprocess output:\n%s", tt.wantMsg, out)
			}
		})
	}
}
