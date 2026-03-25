// Package main demonstrates using gref for loading configuration
// from environment variables into nested structs using dot notation.
package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/vladimirvivien/gref"
)

// Config represents application configuration with nested structs
type Config struct {
	AppName  string         `env:"APP_NAME"`
	Debug    bool           `env:"DEBUG"`
	Port     int            `env:"PORT"`
	Database DatabaseConfig `env:"DATABASE"`
	Cache    CacheConfig    `env:"CACHE"`
}

type DatabaseConfig struct {
	Host     string `env:"HOST"`
	Port     int    `env:"PORT"`
	Username string `env:"USERNAME"`
	Password string `env:"PASSWORD"`
	Name     string `env:"NAME"`
}

type CacheConfig struct {
	Enabled bool  `env:"ENABLED"`
	TTL     int   `env:"TTL"`
	Redis   Redis `env:"REDIS"`
}

type Redis struct {
	Host string `env:"HOST"`
	Port int    `env:"PORT"`
}

// setFieldValue sets a field value from a string, handling type conversion
func setFieldValue(field gref.Field, value string) {
	switch field.Kind() {
	case gref.String:
		field.Set(value)
	case gref.Int:
		if v, err := strconv.Atoi(value); err == nil {
			field.Set(v)
		}
	case gref.Bool:
		if v, err := strconv.ParseBool(value); err == nil {
			field.Set(v)
		}
	}
}

// LoadFromEnv loads configuration from environment variables into a struct.
// It maps environment variables to struct fields using the `env` tag.
// Nested structs are supported via underscore-separated names.
// Example: DATABASE_HOST maps to Config.Database.Host
func LoadFromEnv(cfg any, prefix string) error {
	s := gref.From(cfg).Struct()

	// Build a map of env var names to field paths
	envMap := buildEnvMap(s, prefix, "")

	for envName, fieldPath := range envMap {
		value := os.Getenv(envName)
		if value == "" {
			continue
		}

		if field, ok := s.TryField(fieldPath).Value(); ok {
			setFieldValue(field, value)
		}
	}

	return nil
}

// buildEnvMap recursively builds a map of environment variable names to field paths
func buildEnvMap(s gref.Struct, prefix, pathPrefix string) map[string]string {
	result := make(map[string]string)

	s.Fields().Each(func(field gref.Field) bool {
		tag := field.ParsedTag("env")
		if tag.Name == "" {
			return true
		}

		envName := tag.Name
		if prefix != "" {
			envName = prefix + "_" + envName
		}

		fieldPath := field.Name()
		if pathPrefix != "" {
			fieldPath = pathPrefix + "." + field.Name()
		}

		// If this is a nested struct, recurse into it
		if field.Kind() == gref.StructKind {
			nested := field.Struct()
			nestedMap := buildEnvMap(nested, envName, fieldPath)
			for k, v := range nestedMap {
				result[k] = v
			}
		} else {
			result[envName] = fieldPath
		}
		return true
	})

	return result
}

// PrintConfig prints the configuration for demonstration
func PrintConfig(cfg *Config) {
	fmt.Printf("  AppName:  %s\n", cfg.AppName)
	fmt.Printf("  Debug:    %v\n", cfg.Debug)
	fmt.Printf("  Port:     %d\n", cfg.Port)
	fmt.Println("  Database:")
	fmt.Printf("    Host:     %s\n", cfg.Database.Host)
	fmt.Printf("    Port:     %d\n", cfg.Database.Port)
	fmt.Printf("    Username: %s\n", cfg.Database.Username)
	fmt.Printf("    Password: %s\n", cfg.Database.Password)
	fmt.Printf("    Name:     %s\n", cfg.Database.Name)
	fmt.Println("  Cache:")
	fmt.Printf("    Enabled:  %v\n", cfg.Cache.Enabled)
	fmt.Printf("    TTL:      %d\n", cfg.Cache.TTL)
	fmt.Println("    Redis:")
	fmt.Printf("      Host:   %s\n", cfg.Cache.Redis.Host)
	fmt.Printf("      Port:   %d\n", cfg.Cache.Redis.Port)
}

func main() {
	fmt.Println("=== gref Config Loader Example ===")
	fmt.Println()

	// Set some environment variables for demonstration
	os.Setenv("APP_NAME", "MyApp")
	os.Setenv("DEBUG", "true")
	os.Setenv("PORT", "8080")
	os.Setenv("DATABASE_HOST", "localhost")
	os.Setenv("DATABASE_PORT", "5432")
	os.Setenv("DATABASE_USERNAME", "admin")
	os.Setenv("DATABASE_PASSWORD", "secret")
	os.Setenv("DATABASE_NAME", "myapp_db")
	os.Setenv("CACHE_ENABLED", "true")
	os.Setenv("CACHE_TTL", "3600")
	os.Setenv("CACHE_REDIS_HOST", "redis.local")
	os.Setenv("CACHE_REDIS_PORT", "6379")

	fmt.Println("Environment variables set:")
	for _, env := range []string{
		"APP_NAME", "DEBUG", "PORT",
		"DATABASE_HOST", "DATABASE_PORT", "DATABASE_USERNAME", "DATABASE_PASSWORD", "DATABASE_NAME",
		"CACHE_ENABLED", "CACHE_TTL", "CACHE_REDIS_HOST", "CACHE_REDIS_PORT",
	} {
		fmt.Printf("  %s=%s\n", env, os.Getenv(env))
	}
	fmt.Println()

	// Load configuration from environment
	cfg := &Config{}
	if err := LoadFromEnv(cfg, ""); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Loaded configuration:")
	PrintConfig(cfg)
	fmt.Println()

	// Demonstrate field path access using dot notation
	fmt.Println("Direct field access using gref dot notation:")
	s := gref.From(cfg).Struct()

	// Use dot notation - Field("Database.Host") automatically navigates
	dbHost := s.Field("Database.Host")
	fmt.Printf("  Database.Host = %v\n", dbHost.Interface())

	redisPort := s.Field("Cache.Redis.Port")
	fmt.Printf("  Cache.Redis.Port = %v\n", redisPort.Interface())
}
