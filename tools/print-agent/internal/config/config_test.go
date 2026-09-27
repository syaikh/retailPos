package config

import (
	"strings"
	"testing"
)

func TestValidateRequiresExplicitOriginsInProduction(t *testing.T) {
	cases := []struct {
		name    string
		cfg     Config
		wantErr string
	}{
		{
			name: "production without origins is refused",
			cfg:  Config{Env: "production"},
			// The message has to name the variable and say why, because this is
			// the failure an operator will meet first and it is not obvious.
			wantErr: "ALLOWED_ORIGINS must list at least one exact origin",
		},
		{
			name:    "production with a wildcard is refused",
			cfg:     Config{Env: "production", AllowedOrigins: []string{"*"}},
			wantErr: `ALLOWED_ORIGINS must not be "*"`,
		},
		{
			name:    "production with an exact origin is accepted",
			cfg:     Config{Env: "production", AllowedOrigins: []string{"https://pos.example.com"}},
			wantErr: "",
		},
		{
			name:    "production with several origins is accepted",
			cfg:     Config{Env: "production", AllowedOrigins: []string{"https://pos.example.com", "http://localhost:5173"}},
			wantErr: "",
		},
		{
			// Development keeps the permissive default on purpose: the file
			// transport writes to a temp directory, and refusing to start would
			// only get in the way of the e2e suite and local testing.
			name:    "development without origins is allowed",
			cfg:     Config{Env: "development"},
			wantErr: "",
		},
		{
			name:    "unset env is treated as development",
			cfg:     Config{},
			wantErr: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.Validate()
			switch {
			case tc.wantErr == "" && err != nil:
				t.Fatalf("Validate() = %v, want nil", err)
			case tc.wantErr != "" && err == nil:
				t.Fatalf("Validate() = nil, want error containing %q", tc.wantErr)
			case tc.wantErr != "" && !strings.Contains(err.Error(), tc.wantErr):
				t.Fatalf("Validate() = %q, want it to contain %q", err, tc.wantErr)
			}
		})
	}
}

func TestLoadReadsBindAddressAndEnv(t *testing.T) {
	t.Setenv("ENV", "production")
	t.Setenv("PRINT_BIND_ADDRESS", "127.0.0.1")

	c := Load()
	if c.Env != "production" {
		t.Errorf("Env = %q, want production", c.Env)
	}
	if c.BindAddress != "127.0.0.1" {
		t.Errorf("BindAddress = %q, want 127.0.0.1", c.BindAddress)
	}
}

func TestLoadDefaultsBindAddressToAllInterfaces(t *testing.T) {
	// The browser is a different machine on the till, so loopback-only would
	// stop printing entirely. The exposure this creates is handled by the
	// origin allowlist in Validate, not by narrowing the listener.
	t.Setenv("PRINT_BIND_ADDRESS", "")

	if got := Load().BindAddress; got != "0.0.0.0" {
		t.Errorf("BindAddress = %q, want 0.0.0.0", got)
	}
}
