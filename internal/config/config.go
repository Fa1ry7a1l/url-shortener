package config

import (
	"flag"
	"os"
	"strings"
)

type Config struct {
	Addr            string
	BaseURL         string
	IDLength        int
	FileStoragePath string
	DatabaseDSN     string
	AuthSecret      string
	AuditFile       string
	AuditURL        string
}

const (
	defaultAddr            = "localhost:8080"
	defaultBaseURL         = "http://localhost:8080"
	defaultIDLength        = 8
	defaultFileStoragePath = ""
	defaultDatabaseDSN     = ""
	defaultAuthSecret      = "url-shortener-secret"
	defaultAuditFile       = ""
	defaultAuditURL        = ""
)

func Parse(args []string) (*Config, error) {
	cfg := &Config{}

	fs := flag.NewFlagSet("shortener", flag.ContinueOnError)
	fs.StringVar(&cfg.Addr, "a", defaultAddr, "HTTP server address")
	fs.StringVar(&cfg.BaseURL, "b", defaultBaseURL, "Base URL for short links")
	fs.IntVar(&cfg.IDLength, "l", defaultIDLength, "Length of generated short ID")
	fs.StringVar(&cfg.FileStoragePath, "f", defaultFileStoragePath, "Path to JSON storage file")
	fs.StringVar(&cfg.DatabaseDSN, "d", defaultDatabaseDSN, "PostgreSQL DSN")
	fs.StringVar(&cfg.AuthSecret, "s", defaultAuthSecret, "JWT cookie signing secret")
	fs.StringVar(&cfg.AuditFile, "audit-file", defaultAuditFile, "Path to audit log file")
	fs.StringVar(&cfg.AuditURL, "audit-url", defaultAuditURL, "Remote audit receiver URL")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	if v := os.Getenv("SERVER_ADDRESS"); v != "" {
		cfg.Addr = v
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
