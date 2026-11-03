package config

import (
	"net/url"
	"strings"
	"testing"
)

func clearDatabaseEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"DB_HOST",
		"DB_PORT",
		"DB_NAME",
		"DB_USER",
		"DB_PASS",
		"DB_SSLMODE",
		"DB_SSLROOTCERT",
		"DATABASE_URL",
	} {
		t.Setenv(key, "")
	}
}

func TestDatabaseURLPrefersExplicitDATABASEURL(t *testing.T) {
	clearDatabaseEnv(t)
	t.Setenv("DATABASE_URL", "postgres://postgres:postgres@localhost:55432/cestia?sslmode=disable")

	got, err := databaseURL()
	if err != nil {
		t.Fatalf("databaseURL() error = %v", err)
	}
	if got != "postgres://postgres:postgres@localhost:55432/cestia?sslmode=disable" {
		t.Fatalf("databaseURL() = %q", got)
	}
}

func TestDatabaseURLUsesDefaultWhenNothingIsSet(t *testing.T) {
	clearDatabaseEnv(t)

	got, err := databaseURL()
	if err != nil {
		t.Fatalf("databaseURL() error = %v", err)
	}
	if !strings.Contains(got, "localhost:55432/cestia") {
		t.Fatalf("databaseURL() = %q, want the development default", got)
	}
}

func TestDatabaseURLAssemblesDBVariables(t *testing.T) {
	clearDatabaseEnv(t)
	t.Setenv("DB_HOST", "db.internal")
	t.Setenv("DB_PORT", "6432")
	t.Setenv("DB_NAME", "cestia_prod")
	t.Setenv("DB_USER", "cestia_app")
	t.Setenv("DB_PASS", "p@ss word:/")
	t.Setenv("DB_SSLMODE", "require")

	got, err := databaseURL()
	if err != nil {
		t.Fatalf("databaseURL() error = %v", err)
	}
	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatalf("url.Parse(%q) error = %v", got, err)
	}
	if parsed.Scheme != "postgres" {
		t.Errorf("scheme = %q", parsed.Scheme)
	}
	if parsed.Host != "db.internal:6432" {
		t.Errorf("host = %q", parsed.Host)
	}
	if parsed.Path != "/cestia_prod" {
		t.Errorf("path = %q", parsed.Path)
	}
	if parsed.User.Username() != "cestia_app" {
		t.Errorf("user = %q", parsed.User.Username())
	}
	password, _ := parsed.User.Password()
	if password != "p@ss word:/" {
		t.Errorf("password = %q, want the raw password after parsing", password)
	}
	values := parseQuery(t, parsed)
	if values.Get("sslmode") != "require" {
		t.Errorf("sslmode = %q", values.Get("sslmode"))
	}
}

func TestDatabaseURLAddsRootCertificate(t *testing.T) {
	clearDatabaseEnv(t)
	t.Setenv("DB_HOST", "127.0.0.1")
	t.Setenv("DB_USER", "cestia_app")
	t.Setenv("DB_SSLMODE", "verify-full")
	t.Setenv("DB_SSLROOTCERT", "/etc/cestia/rds-ca.pem")

	got, err := databaseURL()
	if err != nil {
		t.Fatalf("databaseURL() error = %v", err)
	}
	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatalf("url.Parse(%q) error = %v", got, err)
	}
	values := parseQuery(t, parsed)
	if values.Get("sslmode") != "verify-full" {
		t.Errorf("sslmode = %q", values.Get("sslmode"))
	}
	if values.Get("sslrootcert") != "/etc/cestia/rds-ca.pem" {
		t.Errorf("sslrootcert = %q", values.Get("sslrootcert"))
	}
	if values.Get("connect_timeout") != "" {
		t.Errorf("unexpected query parameter: %q", parsed.RawQuery)
	}
}

func parseQuery(t *testing.T, parsed *url.URL) url.Values {
	t.Helper()
	values, err := url.ParseQuery(parsed.RawQuery)
	if err != nil {
		t.Fatalf("url.ParseQuery(%q) error = %v", parsed.RawQuery, err)
	}
	return values
}

func TestDatabaseURLRejectsInvalidPort(t *testing.T) {
	clearDatabaseEnv(t)
	t.Setenv("DB_HOST", "127.0.0.1")
	t.Setenv("DB_PORT", "not-a-port")
	t.Setenv("DB_USER", "cestia_app")

	if _, err := databaseURL(); err == nil {
		t.Fatal("databaseURL() succeeded with an invalid DB_PORT")
	}
}

func TestDatabaseURLRequiresUserWhenHostIsSet(t *testing.T) {
	clearDatabaseEnv(t)
	t.Setenv("DB_HOST", "127.0.0.1")

	if _, err := databaseURL(); err == nil {
		t.Fatal("databaseURL() succeeded without DB_USER")
	}
}

func TestLoadKeepsProductionGuards(t *testing.T) {
	clearDatabaseEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("DB_HOST", "127.0.0.1")
	t.Setenv("DB_USER", "cestia_app")
	t.Setenv("TOKEN_SECRET", "short")

	if _, err := Load(); err == nil {
		t.Fatal("Load() accepted a weak production TOKEN_SECRET")
	}
}
