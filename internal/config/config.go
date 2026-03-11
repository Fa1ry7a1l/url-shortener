package config

import (
	"flag"
	"strings"
)

type Config struct {
	Addr     string
	BaseURL  string
	IDLength int
}

const (
	defaultAddr     = "localhost:8080"
	defaultBaseURL  = "http://localhost:8080"
	defaultIDLength = 8
)

func Parse(args []string) (*Config, error) {
	cfg := &Config{}

	fs := flag.NewFlagSet("shortener", flag.ContinueOnError)
	fs.StringVar(&cfg.Addr, "a", defaultAddr, "HTTP server address (e.g. localhost:8888)")
	fs.StringVar(&cfg.BaseURL, "b", defaultBaseURL, "Base URL for short links (e.g. http://localhost:8000)")
	fs.IntVar(&cfg.IDLength, "l", defaultIDLength, "Length of generated short ID")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")

	if cfg.IDLength <= 0 {
		cfg.IDLength = defaultIDLength
	}

	return cfg, nil
}
