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
}

const (
	defaultAddr            = "localhost:8080"
	defaultBaseURL         = "http://localhost:8080"
	defaultIDLength        = 8
	defaultFileStoragePath = "shortener-db.json"
)

func Parse(args []string) (*Config, error) {
	cfg := &Config{}

	fs := flag.NewFlagSet("shortener", flag.ContinueOnError)
	fs.StringVar(&cfg.Addr, "a", defaultAddr, "HTTP server address (e.g. localhost:8888)")
	fs.StringVar(&cfg.BaseURL, "b", defaultBaseURL, "Base URL for short links (e.g. http://localhost:8000)")
	fs.IntVar(&cfg.IDLength, "l", defaultIDLength, "Length of generated short ID")
	fs.StringVar(&cfg.FileStoragePath, "f", defaultFileStoragePath, "Path to JSON storage file")

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

	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")

	if cfg.IDLength <= 0 {
		cfg.IDLength = defaultIDLength
	}
	if cfg.FileStoragePath == "" {
		cfg.FileStoragePath = defaultFileStoragePath
	}

	return cfg, nil
}
