package config

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all backend server and database configuration.
type Config struct {
	Port              string
	AppEnv            string
	DatabaseURL       string
	AllowedOrigins    []string
	MaxConns          int32
	MinConns          int32
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
}

// Load loads configuration from environment variables, optionally reading a local .env file.
func Load() (*Config, error) {
	// Attempt to load .env if present (non-fatal if absent)
	loadDotEnv(".env")

	port := getEnv("PORT", "8080")
	appEnv := getEnv("APP_ENV", "development")

	// Determine database connection URL
	dbURL := os.Getenv("DATABASE_URL")
	if strings.TrimSpace(dbURL) == "" {
		dbURL = buildDatabaseURLFromParts()
	}

	// Configure allowed CORS origins
	allowedOriginsRaw := os.Getenv("ALLOWED_ORIGINS")
	if strings.TrimSpace(allowedOriginsRaw) == "" {
		allowedOriginsRaw = os.Getenv("FRONTEND_URL")
	}
	var allowedOrigins []string
	if strings.TrimSpace(allowedOriginsRaw) != "" {
		for _, part := range strings.Split(allowedOriginsRaw, ",") {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				allowedOrigins = append(allowedOrigins, trimmed)
			}
		}
	} else {
		// Default to wildcard for local development convenience
		allowedOrigins = []string{"*"}
	}

	maxConns := int32(getEnvAsInt("DB_MAX_CONNS", 10))
	minConns := int32(getEnvAsInt("DB_MIN_CONNS", 2))
	maxConnLifetime := getEnvAsDuration("DB_MAX_CONN_LIFETIME", time.Hour)
	maxConnIdleTime := getEnvAsDuration("DB_MAX_CONN_IDLE_TIME", 30*time.Minute)

	cfg := &Config{
		Port:            port,
		AppEnv:          appEnv,
		DatabaseURL:     dbURL,
		AllowedOrigins:  allowedOrigins,
		MaxConns:        maxConns,
		MinConns:        minConns,
		MaxConnLifetime: maxConnLifetime,
		MaxConnIdleTime: maxConnIdleTime,
	}

	return cfg, nil
}

// HasDatabaseConfig reports whether database connection settings are present.
func (c *Config) HasDatabaseConfig() bool {
	return strings.TrimSpace(c.DatabaseURL) != ""
}

// SanitizedDatabaseURL returns the connection URL with the password masked for safe logging.
func (c *Config) SanitizedDatabaseURL() string {
	u, err := url.Parse(c.DatabaseURL)
	if err != nil {
		return "[REDACTED_INVALID_URL]"
	}
	if u.User != nil {
		if _, hasPassword := u.User.Password(); hasPassword {
			u.User = url.UserPassword(u.User.Username(), "REDACTED")
		}
	}
	return u.String()
}

func buildDatabaseURLFromParts() string {
	host := os.Getenv("DB_HOST")
	user := os.Getenv("DB_USER")
	name := os.Getenv("DB_NAME")

	// If none of these are defined, return empty to trigger validation error
	if host == "" && user == "" && name == "" {
		return ""
	}

	if host == "" {
		host = "localhost"
	}
	port := getEnv("DB_PORT", "5432")
	password := os.Getenv("DB_PASSWORD")
	sslMode := getEnv("DB_SSLMODE", "disable")

	var userInfo *url.Userinfo
	if password != "" {
		userInfo = url.UserPassword(user, password)
	} else if user != "" {
		userInfo = url.User(user)
	}

	u := url.URL{
		Scheme: "postgres",
		User:   userInfo,
		Host:   fmt.Sprintf("%s:%s", host, port),
		Path:   name,
	}

	q := u.Query()
	q.Set("sslmode", sslMode)
	u.RawQuery = q.Encode()

	return u.String()
}

func getEnv(key, defaultVal string) string {
	val := os.Getenv(key)
	if strings.TrimSpace(val) == "" {
		return defaultVal
	}
	return strings.TrimSpace(val)
}

func getEnvAsInt(key string, defaultVal int) int {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return defaultVal
	}
	return val
}

func getEnvAsDuration(key string, defaultVal time.Duration) time.Duration {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultVal
	}
	d, err := time.ParseDuration(valStr)
	if err != nil {
		return defaultVal
	}
	return d
}

// loadDotEnv parses a key-value .env file and sets variables that are not already set in the environment.
func loadDotEnv(filepath string) {
	file, err := os.Open(filepath)
	if err != nil {
		return // File does not exist, ignore
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		// Strip surrounding quotes if present
		if len(val) >= 2 && ((val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'')) {
			val = val[1 : len(val)-1]
		}

		// Don't overwrite existing environment variables
		if os.Getenv(key) == "" {
			_ = os.Setenv(key, val)
		}
	}
}
