package config

import "os"

type Config struct {
	Env        string
	CORSOrigin string
	JWTSecret  string
}

func Load() *Config {
	env := os.Getenv("ENV")
	if env == "" {
		env = "development"
	}

	corsOrigin := os.Getenv("CORS_ORIGIN")
	if corsOrigin == "" {
		corsOrigin = "http://localhost:5173"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "your-secret-key-change-in-production"
	}

	return &Config{
		Env:        env,
		CORSOrigin: corsOrigin,
		JWTSecret:  jwtSecret,
	}
}