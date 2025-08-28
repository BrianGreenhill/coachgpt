package config

import (
	"log"
	"os"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	Port        string `env:"PORT,required"`
	Env         string `env:"APP_ENV" envDefault:"dev"`
	DatabaseURL string `env:"DATABASE_URL,required"`
	SessionKey  string `env:"SESSION_KEY,required"`
	JWTSecret   string `env:"JWT_SECRET,required"`
	BaseURL     string `env:"BASE_URL,required"`
	Strava      StravaConfig

	RedisAddr string `env:"REDIS_ADDR,required"`
}

type StravaConfig struct {
	ClientID     string `env:"STRAVA_CLIENT_ID,required"`
	ClientSecret string `env:"STRAVA_CLIENT_SECRET,required"`
}

func Load() Config {
	var cfg Config

	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("Failed to parse env vars: %v", err)
	}
	return cfg
}

func MstGetenv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("Environment variable %s not set", key)
	}
	return v
}
