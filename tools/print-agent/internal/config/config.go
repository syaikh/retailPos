// Package config loads print-agent configuration from environment variables.
package config

import (
	"os"
	"strings"
)

// Config holds all print-agent settings.
type Config struct {
	Port           string
	Transport      string
	OutputDir      string
	TCPAddr        string
	SerialDevice   string
	Token          string
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
//	PORT                 listen port                  (default 9123)
//	PRINT_TRANSPORT      file | tcp | serial         (default file)
//	PRINT_OUTPUT_DIR     file output directory        (default os temp dir)
//	PRINT_TCP_ADDR       host:port for tcp           (required if tcp)
//	PRINT_SERIAL_DEVICE  e.g. /dev/ttyUSB0           (required if serial)
//	PRINT_TOKEN          optional bearer token
//	ALLOWED_ORIGINS      comma-separated origins      (default "*")
func Load() Config {
	c := Config{
		Port:         getenv("PORT", "9123"),
		Transport:    getenv("PRINT_TRANSPORT", "file"),
		OutputDir:    getenv("PRINT_OUTPUT_DIR", os.TempDir()),
		TCPAddr:      os.Getenv("PRINT_TCP_ADDR"),
		SerialDevice: os.Getenv("PRINT_SERIAL_DEVICE"),
		Token:        os.Getenv("PRINT_TOKEN"),
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
