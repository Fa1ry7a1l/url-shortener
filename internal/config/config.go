package config

import (
	"flag"
	"os"
	"strconv"
	"strings"
)

// Config contains runtime settings for the shortener service.
type Config struct {
	// Addr is the HTTP server listen address.
	Addr string
	// PprofAddr is the diagnostics server listen address.
	PprofAddr string
	// BaseURL is the public base URL used to build short links.
	BaseURL string
	// IDLength is the length of generated short IDs.
	IDLength int
	// FileStoragePath is the path to the JSON file storage backend.
	FileStoragePath string
	// DatabaseDSN enables the PostgreSQL storage backend when set.
	DatabaseDSN string
	// AuthSecret signs authentication cookies.
	AuthSecret string
	// EnableHTTPS starts the main web server with TLS enabled.
	EnableHTTPS bool
	// AuditFile enables JSON-lines audit logging when set.
	AuditFile string
	// AuditURL enables remote HTTP audit publishing when set.
	AuditURL string
}

const (
	defaultAddr            = "localhost:8080"
	defaultPprofAddr       = "localhost:6060"
	defaultBaseURL         = "http://localhost:8080"
	defaultIDLength        = 8
	defaultFileStoragePath = ""
	defaultDatabaseDSN     = ""
	defaultAuthSecret      = "url-shortener-secret"
	defaultEnableHTTPS     = false
	defaultAuditFile       = ""
	defaultAuditURL        = ""
)

// Parse reads settings from command-line arguments and environment variables.
func Parse(args []string) (*Config, error) {
	cfg := &Config{}

	fs := flag.NewFlagSet("shortener", flag.ContinueOnError)
	fs.StringVar(&cfg.Addr, "a", defaultAddr, "HTTP server address")
	fs.StringVar(&cfg.PprofAddr, "pprof-address", defaultPprofAddr, "pprof diagnostics server address")
	fs.StringVar(&cfg.BaseURL, "b", defaultBaseURL, "Base URL for short links")
	fs.IntVar(&cfg.IDLength, "l", defaultIDLength, "Length of generated short ID")
	fs.StringVar(&cfg.FileStoragePath, "f", defaultFileStoragePath, "Path to JSON storage file")
	fs.StringVar(&cfg.DatabaseDSN, "d", defaultDatabaseDSN, "PostgreSQL DSN")
	fs.BoolVar(&cfg.EnableHTTPS, "s", defaultEnableHTTPS, "Enable HTTPS server")
	fs.StringVar(&cfg.AuthSecret, "auth-secret", defaultAuthSecret, "JWT cookie signing secret")
	fs.StringVar(&cfg.AuditFile, "audit-file", defaultAuditFile, "Path to audit log file")
	fs.StringVar(&cfg.AuditURL, "audit-url", defaultAuditURL, "Remote audit receiver URL")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	if v := os.Getenv("SERVER_ADDRESS"); v != "" {
		cfg.Addr = v
	}
	if v := os.Getenv("PPROF_ADDRESS"); v != "" {
		cfg.PprofAddr = v
	}
	if v := os.Getenv("BASE_URL"); v != "" {
		cfg.BaseURL = v
	}
	if v := os.Getenv("FILE_STORAGE_PATH"); v != "" {
		cfg.FileStoragePath = v
	}
	if v := os.Getenv("DATABASE_DSN"); v != "" {
		cfg.DatabaseDSN = v
	}
	if v := os.Getenv("AUTH_SECRET"); v != "" {
		cfg.AuthSecret = v
	}
	if v := os.Getenv("ENABLE_HTTPS"); v != "" {
		enableHTTPS, err := strconv.ParseBool(v)
		if err != nil {
			enableHTTPS = true
		}
		cfg.EnableHTTPS = enableHTTPS
	}
	if v := os.Getenv("AUDIT_FILE"); v != "" {
		cfg.AuditFile = v
	}
	if v := os.Getenv("AUDIT_URL"); v != "" {
		cfg.AuditURL = v
	}

	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")

	if cfg.IDLength <= 0 {
		cfg.IDLength = defaultIDLength
	}
	if cfg.FileStoragePath == "" { //останется для сходства с остальными параметрами
		cfg.FileStoragePath = defaultFileStoragePath
	}

	if cfg.AuditFile == "" {
		cfg.AuditFile = defaultAuditFile
	}
	if cfg.AuditURL == "" {
		cfg.AuditURL = defaultAuditURL
	}

	return cfg, nil
}
