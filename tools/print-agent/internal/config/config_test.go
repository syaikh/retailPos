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

func TestLoadReadsCashDrawerSettings(t *testing.T) {
	// Pin and width are environment-agnostic, so the environment is pinned to
	// development here: an ENV=production inherited from the shell would make
	// Validate fail on ALLOWED_ORIGINS and mask what is being tested.
	t.Setenv("ENV", "")
	t.Setenv("PRINT_DRAWER_PIN", "2")
	t.Setenv("PRINT_DRAWER_PULSE_MS", "40")

	c := Load()
	if c.DrawerPin != 2 {
		t.Errorf("DrawerPin = %d, want 2", c.DrawerPin)
	}
	if c.DrawerPulseMs != 40 {
		t.Errorf("DrawerPulseMs = %d, want 40", c.DrawerPulseMs)
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("Validate() = %v, want nil", err)
	}
}

func TestLoadDefaultsCashDrawerSettings(t *testing.T) {
	// Pin 0 drives drawer connector pin 2, which is what every RJ11 drawer
	// sold for these printers is wired to, so it has to be what an operator
	// who configures nothing gets.
	t.Setenv("ENV", "")
	t.Setenv("PRINT_DRAWER_PIN", "")
	t.Setenv("PRINT_DRAWER_PULSE_MS", "")

	c := Load()
	if c.DrawerPin != defaultDrawerPin {
		t.Errorf("DrawerPin = %d, want %d", c.DrawerPin, defaultDrawerPin)
	}
	if c.DrawerPulseMs != 0 {
		t.Errorf("DrawerPulseMs = %d, want 0 (the encoder supplies the width)", c.DrawerPulseMs)
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("Validate() = %v, want nil", err)
	}
}

// A drawer setting that could never open the drawer is refused at startup in
// every environment. The failure it prevents is not a crash: it is a till that
// runs, accepts sales, and quietly never opens.
func TestValidateRefusesUnusableCashDrawerSettings(t *testing.T) {
	cases := []struct {
		name    string
		env     map[string]string
		wantErr string
	}{
		{
			name:    "a pin that is not a number is refused",
			env:     map[string]string{"PRINT_DRAWER_PIN": "two"},
			wantErr: "PRINT_DRAWER_PIN must be a whole number",
		},
		{
			name:    "a pin outside the documented set is refused",
			env:     map[string]string{"PRINT_DRAWER_PIN": "3"},
			wantErr: "PRINT_DRAWER_PIN must be 0 or 2",
		},
		{
			name:    "a pulse length above a byte is refused",
			env:     map[string]string{"PRINT_DRAWER_PULSE_MS": "999"},
			wantErr: "PRINT_DRAWER_PULSE_MS must be a whole number",
		},
		{
			name:    "a negative pulse length is refused",
			env:     map[string]string{"PRINT_DRAWER_PULSE_MS": "-5"},
			wantErr: "PRINT_DRAWER_PULSE_MS must be a whole number",
		},
		{
			name: "every documented pin is accepted",
			env:  map[string]string{"PRINT_DRAWER_PIN": "5"},
		},
		{
			name: "a zero pulse length is accepted as unset",
			env:  map[string]string{"PRINT_DRAWER_PULSE_MS": "0"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Clear both so a value left in the developer's environment by an
			// earlier case cannot make this one pass or fail on its own.
			t.Setenv("ENV", "")
			t.Setenv("PRINT_DRAWER_PIN", "")
			t.Setenv("PRINT_DRAWER_PULSE_MS", "")
			for k, v := range tc.env {
				t.Setenv(k, v)
			}

			err := Load().Validate()
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
