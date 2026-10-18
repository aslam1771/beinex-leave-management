package config

import (
	"os"
	"strings"
	"testing"
)

func TestSanitizedDatabaseURL(t *testing.T) {
	cfg := &Config{
		DatabaseURL: "postgres://myuser:secret123@localhost:5432/beinex_leave_db?sslmode=disable",
	}

	sanitized := cfg.SanitizedDatabaseURL()
	if strings.Contains(sanitized, "secret123") {
		t.Fatalf("SanitizedDatabaseURL leaked plain password: %s", sanitized)
	}

	if !strings.Contains(sanitized, "REDACTED") {
		t.Fatalf("SanitizedDatabaseURL did not replace password with mask: %s", sanitized)
	}
}

func TestBuildDatabaseURLFromParts(t *testing.T) {
	os.Setenv("DB_HOST", "db.example.com")
	os.Setenv("DB_PORT", "5433")
	os.Setenv("DB_USER", "leave_user")
	os.Setenv("DB_PASSWORD", "p@ssword!")
	os.Setenv("DB_NAME", "leave_db")
	os.Setenv("DB_SSLMODE", "require")
	defer func() {
		os.Unsetenv("DB_HOST")
		os.Unsetenv("DB_PORT")
		os.Unsetenv("DB_USER")
		os.Unsetenv("DB_PASSWORD")
		os.Unsetenv("DB_NAME")
		os.Unsetenv("DB_SSLMODE")
	}()

	url := buildDatabaseURLFromParts()
	if !strings.HasPrefix(url, "postgres://leave_user:") {
		t.Fatalf("unexpected constructed url: %s", url)
	}
	if !strings.Contains(url, "db.example.com:5433") {
		t.Fatalf("host or port missing in url: %s", url)
	}
	if !strings.Contains(url, "sslmode=require") {
		t.Fatalf("sslmode missing in url: %s", url)
	}
}
