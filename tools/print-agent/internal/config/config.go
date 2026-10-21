// Package config loads print-agent configuration from environment variables.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
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

	// DrawerPin is the drawer-connector pin pulsed by ESC p: 0 drives pin 2 and
	// 1 drives pin 5 per the EPSON reference, with 2 and 5 accepted as the
	// synonyms ESC/POS compatible printers use.
	DrawerPin uint8
	// DrawerPulseMs is the ESC p pulse length in milliseconds. Zero means "not
	// configured" and escpos.DrawerPulse supplies its default, so both an unset
	// variable and an explicit 0 open the drawer the same way.
	DrawerPulseMs uint8

	// drawerErr carries a drawer setting that was present but unusable. Load
	// can only return a Config, so the reason travels with the value and
	// Validate reports it at startup. This is not a deferred failure the agent
	// then runs with: main refuses to start on it, and the default it fell back
	// to is never used, so a typo cannot become a drawer that silently never
	// opens on a shop floor.
	drawerErr error
}

// defaultDrawerPin is ESC p pin 0, which drives pin 2 of the drawer connector —
// the wiring every RJ11 cash drawer sold for these printers expects.
const defaultDrawerPin uint8 = 0

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// loadUint8 reads a setting that must fit in a byte. An unset variable returns
// def. A set one returns the value, or def plus the reason it could not be
// used: coercing it instead would silently hand the operator a pulse that
// opens nothing.
func loadUint8(key string, def uint8) (uint8, error) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 || n > 255 {
		return def, fmt.Errorf("%s must be a whole number between 0 and 255, got %q", key, v)
	}
	return uint8(n), nil
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
//	PRINT_DRAWER_PIN     ESC p pin 0|1|2|5           (default 0)
//	PRINT_DRAWER_PULSE_MS ESC p pulse length 1-255   (default 25)
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
	// The drawer settings are read after the literal because each can fail and
	// Load has nowhere to return a failure to. The reason is kept until
	// Validate, which main calls before a single byte leaves the agent.
	var errPin, errPulse error
	c.DrawerPin, errPin = loadUint8("PRINT_DRAWER_PIN", defaultDrawerPin)
	// Zero is the deliberate "unset" value: the encoder supplies the width, so
	// an operator who writes 0 gets exactly the same pulse as one who writes
	// nothing.
	c.DrawerPulseMs, errPulse = loadUint8("PRINT_DRAWER_PULSE_MS", 0)
	c.drawerErr = errors.Join(errPin, errPulse)
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
//
// The drawer settings are checked in every environment, development included.
// They are not a safety question, but an unsupported pin produces a pulse that
// opens nothing, and the only moment that becomes obvious is when a customer is
// waiting. A development box driving a real drawer wants the same refusal.
func (c Config) Validate() error {
	if err := c.validateDrawer(); err != nil {
		return err
	}
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

// validateDrawer refuses a cash-drawer configuration that could never open the
// drawer. A zero value is valid: it is an unset variable, pin 0 is the default,
// and a zero pulse width is supplied by the encoder.
func (c Config) validateDrawer() error {
	if c.drawerErr != nil {
		return c.drawerErr
	}
	switch c.DrawerPin {
	case 0, 1, 2, 5:
	default:
		return errors.New("PRINT_DRAWER_PIN must be 0 or 2 for drawer connector pin 2, " +
			"or 1 or 5 for pin 5; any other pin drives nothing")
	}
	return nil
}
