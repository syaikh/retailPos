// Package config loads print-agent configuration from environment variables.
package config

import (
	"errors"
	"os"
	"strings"
)

// Config holds all print-agent settings.
type Config struct {
	// Env selects production behaviour. It is the same variable name the
	// backend uses, so one secret file can describe a whole till.
	Env            string
	BindAddress    string
	Port           string
	Transport      string
	OutputDir      string
	TCPAddr        string
	SerialDevice   string
	AllowedOrigins []string
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// Load reads configuration from the environment.
//
//	ENV                 production | development    (default development)
//	PRINT_BIND_ADDRESS  listen address                (default 0.0.0.0)
//	PORT                 listen port                  (default 9123)
//	PRINT_TRANSPORT      file | tcp | serial         (default file)
//	PRINT_OUTPUT_DIR     file output directory        (default os temp dir)
//	PRINT_TCP_ADDR       host:port for tcp           (required if tcp)
//	PRINT_SERIAL_DEVICE  e.g. /dev/ttyUSB0           (required if serial)
//	ALLOWED_ORIGINS      comma-separated origins      (default "*")
//
// There is no PRINT_TOKEN. A bearer token was supported and removed: the only
// client is a browser, so the token would have to be built into the page where
// anyone can read it, which makes it useless as a secret. It had no other
// client, and enabling it made every print return 401. ALLOWED_ORIGINS plus
// network isolation is the actual access control.
func Load() Config {
	c := Config{
		Env:          getenv("ENV", "development"),
		BindAddress:  getenv("PRINT_BIND_ADDRESS", "0.0.0.0"),
		Port:         getenv("PORT", "9123"),
		Transport:    getenv("PRINT_TRANSPORT", "file"),
		OutputDir:    getenv("PRINT_OUTPUT_DIR", os.TempDir()),
		TCPAddr:      os.Getenv("PRINT_TCP_ADDR"),
		SerialDevice: os.Getenv("PRINT_SERIAL_DEVICE"),
	}
	if v := os.Getenv("ALLOWED_ORIGINS"); v != "" {
		for _, o := range strings.Split(v, ",") {
			if o = strings.TrimSpace(o); o != "" {
				c.AllowedOrigins = append(c.AllowedOrigins, o)
			}
		}
	}
	return c
}

// Validate refuses configurations that are safe in development and dangerous on
// a shop floor, so the mistake is made at startup rather than discovered by a
// customer.
//
// The one that matters is ALLOWED_ORIGINS. The browser is the only client, and
// it identifies itself with an Origin header that a page cannot forge
// cross-origin. Left unset, the handler reflects whatever origin asks, which
// means any website a cashier has open in another tab can POST a /print request
// to the till's printer: arbitrary content on a receipt roll, or a queue filled
// until it stops draining. Requiring the list in production turns that from
// silent exposure into a refused startup.
func (c Config) Validate() error {
	if c.Env != "production" {
		return nil
	}
	if len(c.AllowedOrigins) == 0 {
		return errors.New("ALLOWED_ORIGINS must list at least one exact origin when ENV=production; " +
			"an unset list lets any website the cashier visits print to this till. " +
			"Use the same value as the backend's CORS_ORIGIN")
	}
	for _, o := range c.AllowedOrigins {
		if o == "*" {
			return errors.New(`ALLOWED_ORIGINS must not be "*" when ENV=production; ` +
				"list the exact origin of the POS frontend instead")
		}
	}
	return nil
}
