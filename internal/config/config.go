package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

// Config contains runtime settings for the shortener service.
type Config struct {
	// Addr is the HTTP server listen address.
	Addr string `json:"server_address"`
	// GRPCAddr is the gRPC server listen address.
	GRPCAddr string `json:"grpc_server_address"`
	// PprofAddr is the diagnostics server listen address.
	PprofAddr string `json:"pprof_address"`
	// BaseURL is the public base URL used to build short links.
	BaseURL string `json:"base_url"`
	// IDLength is the length of generated short IDs.
	IDLength int `json:"id_length"`
	// FileStoragePath is the path to the JSON file storage backend.
	FileStoragePath string `json:"file_storage_path"`
	// DatabaseDSN enables the PostgreSQL storage backend when set.
	DatabaseDSN string `json:"database_dsn"`
	// AuthSecret signs authentication cookies.
	AuthSecret string `json:"auth_secret"`
	// EnableHTTPS starts the main web server with TLS enabled.
	EnableHTTPS bool `json:"enable_https"`
	// AuditFile enables JSON-lines audit logging when set.
	AuditFile string `json:"audit_file"`
	// AuditURL enables remote HTTP audit publishing when set.
	AuditURL string `json:"audit_url"`
	// TrustedSubnet allows access to internal endpoints from matching client IPs.
	TrustedSubnet string `json:"trusted_subnet"`
}

type fileConfig struct {
	Addr            *string `json:"server_address"`
	GRPCAddr        *string `json:"grpc_server_address"`
	PprofAddr       *string `json:"pprof_address"`
	BaseURL         *string `json:"base_url"`
	IDLength        *int    `json:"id_length"`
	FileStoragePath *string `json:"file_storage_path"`
	DatabaseDSN     *string `json:"database_dsn"`
	AuthSecret      *string `json:"auth_secret"`
	EnableHTTPS     *bool   `json:"enable_https"`
	AuditFile       *string `json:"audit_file"`
	AuditURL        *string `json:"audit_url"`
	TrustedSubnet   *string `json:"trusted_subnet"`
}

const (
	defaultAddr            = "localhost:8080"
	defaultGRPCAddr        = "localhost:3200"
	defaultPprofAddr       = "localhost:6060"
	defaultBaseURL         = "http://localhost:8080"
	defaultIDLength        = 8
	defaultFileStoragePath = ""
	defaultDatabaseDSN     = ""
	defaultAuthSecret      = "url-shortener-secret"
	defaultEnableHTTPS     = false
	defaultAuditFile       = ""
	defaultAuditURL        = ""
	defaultTrustedSubnet   = ""
)

// Parse reads settings from defaults, JSON config, command-line arguments, and environment variables.
func Parse(args []string) (*Config, error) {
	cfg := defaultConfig()
	flagCfg := defaultConfig()
	var configPath string

	fs := flag.NewFlagSet("shortener", flag.ContinueOnError)
	fs.StringVar(&flagCfg.Addr, "a", defaultAddr, "HTTP server address")
	fs.StringVar(&flagCfg.GRPCAddr, "g", defaultGRPCAddr, "gRPC server address")
	fs.StringVar(&flagCfg.GRPCAddr, "grpc-address", defaultGRPCAddr, "gRPC server address")
	fs.StringVar(&flagCfg.GRPCAddr, "grpc-server-address", defaultGRPCAddr, "gRPC server address")
	fs.StringVar(&flagCfg.PprofAddr, "pprof-address", defaultPprofAddr, "pprof diagnostics server address")
	fs.StringVar(&flagCfg.BaseURL, "b", defaultBaseURL, "Base URL for short links")
	fs.IntVar(&flagCfg.IDLength, "l", defaultIDLength, "Length of generated short ID")
	fs.StringVar(&flagCfg.FileStoragePath, "f", defaultFileStoragePath, "Path to JSON storage file")
	fs.StringVar(&flagCfg.DatabaseDSN, "d", defaultDatabaseDSN, "PostgreSQL DSN")
	fs.BoolVar(&flagCfg.EnableHTTPS, "s", defaultEnableHTTPS, "Enable HTTPS server")
	fs.StringVar(&flagCfg.AuthSecret, "auth-secret", defaultAuthSecret, "JWT cookie signing secret")
	fs.StringVar(&flagCfg.AuditFile, "audit-file", defaultAuditFile, "Path to audit log file")
	fs.StringVar(&flagCfg.AuditURL, "audit-url", defaultAuditURL, "Remote audit receiver URL")
	fs.StringVar(&flagCfg.TrustedSubnet, "t", defaultTrustedSubnet, "Trusted CIDR subnet for internal endpoints")
	fs.StringVar(&configPath, "c", "", "Path to JSON config file")
	fs.StringVar(&configPath, "config", "", "Path to JSON config file")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	if v := os.Getenv("CONFIG"); v != "" {
		configPath = v
	}

	if configPath != "" {
		if err := applyFileConfig(cfg, configPath); err != nil {
			return nil, err
		}
	}

	applyFlagConfig(cfg, flagCfg, fs)
	if err := applyEnvConfig(cfg); err != nil {
		return nil, err
	}
	normalize(cfg)
	if err := validate(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func defaultConfig() *Config {
	return &Config{
		Addr:            defaultAddr,
		GRPCAddr:        defaultGRPCAddr,
		PprofAddr:       defaultPprofAddr,
		BaseURL:         defaultBaseURL,
		IDLength:        defaultIDLength,
		FileStoragePath: defaultFileStoragePath,
		DatabaseDSN:     defaultDatabaseDSN,
		AuthSecret:      defaultAuthSecret,
		EnableHTTPS:     defaultEnableHTTPS,
		AuditFile:       defaultAuditFile,
		AuditURL:        defaultAuditURL,
		TrustedSubnet:   defaultTrustedSubnet,
	}
}

func applyFileConfig(cfg *Config, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}

	var fileCfg fileConfig
	if err := json.Unmarshal(data, &fileCfg); err != nil {
		return fmt.Errorf("parse config file: %w", err)
	}

	if fileCfg.Addr != nil {
		cfg.Addr = *fileCfg.Addr
	}
	if fileCfg.GRPCAddr != nil {
		cfg.GRPCAddr = *fileCfg.GRPCAddr
	}
	if fileCfg.PprofAddr != nil {
		cfg.PprofAddr = *fileCfg.PprofAddr
	}
	if fileCfg.BaseURL != nil {
		cfg.BaseURL = *fileCfg.BaseURL
	}
	if fileCfg.IDLength != nil {
		cfg.IDLength = *fileCfg.IDLength
	}
	if fileCfg.FileStoragePath != nil {
		cfg.FileStoragePath = *fileCfg.FileStoragePath
	}
	if fileCfg.DatabaseDSN != nil {
		cfg.DatabaseDSN = *fileCfg.DatabaseDSN
	}
	if fileCfg.AuthSecret != nil {
		cfg.AuthSecret = *fileCfg.AuthSecret
	}
	if fileCfg.EnableHTTPS != nil {
		cfg.EnableHTTPS = *fileCfg.EnableHTTPS
	}
	if fileCfg.AuditFile != nil {
		cfg.AuditFile = *fileCfg.AuditFile
	}
	if fileCfg.AuditURL != nil {
		cfg.AuditURL = *fileCfg.AuditURL
	}
	if fileCfg.TrustedSubnet != nil {
		cfg.TrustedSubnet = *fileCfg.TrustedSubnet
	}

	return nil
}

func applyFlagConfig(cfg, flagCfg *Config, fs *flag.FlagSet) {
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "a":
			cfg.Addr = flagCfg.Addr
		case "g", "grpc-address", "grpc-server-address":
			cfg.GRPCAddr = flagCfg.GRPCAddr
		case "pprof-address":
			cfg.PprofAddr = flagCfg.PprofAddr
		case "b":
			cfg.BaseURL = flagCfg.BaseURL
		case "l":
			cfg.IDLength = flagCfg.IDLength
		case "f":
			cfg.FileStoragePath = flagCfg.FileStoragePath
		case "d":
			cfg.DatabaseDSN = flagCfg.DatabaseDSN
		case "s":
			cfg.EnableHTTPS = flagCfg.EnableHTTPS
		case "auth-secret":
			cfg.AuthSecret = flagCfg.AuthSecret
		case "audit-file":
			cfg.AuditFile = flagCfg.AuditFile
		case "audit-url":
			cfg.AuditURL = flagCfg.AuditURL
		case "t":
			cfg.TrustedSubnet = flagCfg.TrustedSubnet
		}
	})
}

func applyEnvConfig(cfg *Config) error {
	if v := os.Getenv("SERVER_ADDRESS"); v != "" {
		cfg.Addr = v
	}
	if v := os.Getenv("GRPC_ADDRESS"); v != "" {
		cfg.GRPCAddr = v
	}
	if v := os.Getenv("GRPC_SERVER_ADDRESS"); v != "" {
		cfg.GRPCAddr = v
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
			return fmt.Errorf("parse ENABLE_HTTPS: %w", err)
		}
		cfg.EnableHTTPS = enableHTTPS
	}
	if v := os.Getenv("AUDIT_FILE"); v != "" {
		cfg.AuditFile = v
	}
	if v := os.Getenv("AUDIT_URL"); v != "" {
		cfg.AuditURL = v
	}
	if v := os.Getenv("TRUSTED_SUBNET"); v != "" {
		cfg.TrustedSubnet = v
	}

	return nil
}

func normalize(cfg *Config) {
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")

	if cfg.IDLength <= 0 {
		cfg.IDLength = defaultIDLength
	}
	if cfg.FileStoragePath == "" {
		cfg.FileStoragePath = defaultFileStoragePath
	}
	if cfg.AuditFile == "" {
		cfg.AuditFile = defaultAuditFile
	}
	if cfg.AuditURL == "" {
		cfg.AuditURL = defaultAuditURL
	}
	if cfg.TrustedSubnet == "" {
		cfg.TrustedSubnet = defaultTrustedSubnet
	}
}

func validate(cfg *Config) error {
	if cfg.TrustedSubnet == "" {
		return nil
	}

	if _, _, err := net.ParseCIDR(cfg.TrustedSubnet); err != nil {
		return fmt.Errorf("parse trusted subnet: %w", err)
	}

	return nil
}
