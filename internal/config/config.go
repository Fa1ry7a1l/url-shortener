package config

import (
	"flag"
	"strings"
	"sync"
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

var (
	instance *Config
	once     sync.Once
)

func New() *Config {
	once.Do(func() {
		cfg := &Config{}

		flag.StringVar(&cfg.Addr, "a", defaultAddr, "HTTP server address (e.g. localhost:8888)")
		flag.StringVar(&cfg.BaseURL, "b", defaultBaseURL, "Base URL for short links (e.g. http://localhost:8000)")
		flag.IntVar(&cfg.IDLength, "l", defaultIDLength, "Length of generated short ID")

		flag.Parse()

		cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")

		if cfg.IDLength <= 0 {
			cfg.IDLength = defaultIDLength
		}

		instance = cfg
	})

	return instance
}
