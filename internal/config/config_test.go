package config_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Fa1ry7a1l/url-shortener/internal/config"
)

func TestParse_Defaults(t *testing.T) {
	t.Setenv("SERVER_ADDRESS", "")
	t.Setenv("PPROF_ADDRESS", "")
	t.Setenv("BASE_URL", "")
	t.Setenv("ENABLE_HTTPS", "")
	t.Setenv("AUDIT_FILE", "")
	t.Setenv("AUDIT_URL", "")

	cfg, err := config.Parse(nil)
	require.NoError(t, err)

	require.Equal(t, "localhost:8080", cfg.Addr)
	require.Equal(t, "localhost:6060", cfg.PprofAddr)
	require.Equal(t, "http://localhost:8080", cfg.BaseURL)
	require.Equal(t, 8, cfg.IDLength)
	require.False(t, cfg.EnableHTTPS)
	require.Equal(t, "", cfg.AuditFile)
	require.Equal(t, "", cfg.AuditURL)
}

func TestParse_Flags(t *testing.T) {
	t.Setenv("SERVER_ADDRESS", "")
	t.Setenv("PPROF_ADDRESS", "")
	t.Setenv("BASE_URL", "")
	t.Setenv("ENABLE_HTTPS", "")

	cfg, err := config.Parse([]string{
		"-a", "localhost:9999",
		"--pprof-address", "localhost:6061",
		"-b", "http://localhost:1111/",
		"-l", "12",
		"-s",
	})
	require.NoError(t, err)

	require.Equal(t, "localhost:9999", cfg.Addr)
	require.Equal(t, "localhost:6061", cfg.PprofAddr)
	require.Equal(t, "http://localhost:1111", cfg.BaseURL)
	require.Equal(t, 12, cfg.IDLength)
	require.True(t, cfg.EnableHTTPS)
}

func TestParse_EnvOverridesFlags(t *testing.T) {
	t.Setenv("SERVER_ADDRESS", "localhost:7777")
	t.Setenv("PPROF_ADDRESS", "localhost:6062")
	t.Setenv("BASE_URL", "http://localhost:7777/")
	t.Setenv("ENABLE_HTTPS", "true")

	cfg, err := config.Parse([]string{
		"-a", "localhost:9999",
		"--pprof-address", "localhost:6061",
		"-b", "http://localhost:1111",
		"-l", "12",
	})
	require.NoError(t, err)

	require.Equal(t, "localhost:7777", cfg.Addr)
	require.Equal(t, "localhost:6062", cfg.PprofAddr)
	require.Equal(t, "http://localhost:7777", cfg.BaseURL)
	require.Equal(t, 12, cfg.IDLength)
	require.True(t, cfg.EnableHTTPS)
}

func TestParse_EnableHTTPS_EnvFalseOverridesFlag(t *testing.T) {
	t.Setenv("ENABLE_HTTPS", "false")

	cfg, err := config.Parse([]string{"-s"})
	require.NoError(t, err)

	require.False(t, cfg.EnableHTTPS)
}

func TestParse_AuthSecret_LongFlag(t *testing.T) {
	t.Setenv("AUTH_SECRET", "")

	cfg, err := config.Parse([]string{"--auth-secret", "flag-secret"})
	require.NoError(t, err)

	require.Equal(t, "flag-secret", cfg.AuthSecret)
}

func TestParse_InvalidIDLength_FallsBackToDefault(t *testing.T) {
	t.Setenv("SERVER_ADDRESS", "")
	t.Setenv("BASE_URL", "")

	cfg, err := config.Parse([]string{"-l", "0"})
	require.NoError(t, err)

	require.Equal(t, 8, cfg.IDLength)
}

func TestParse_InvalidFlag_ReturnsError(t *testing.T) {
	t.Setenv("SERVER_ADDRESS", "")
	t.Setenv("BASE_URL", "")

	cfg, err := config.Parse([]string{"-unknown", "value"})
	require.Error(t, err)
	require.Nil(t, cfg)
}

func TestParse_FileStoragePath_Default(t *testing.T) {
	t.Setenv("FILE_STORAGE_PATH", "")

	cfg, err := config.Parse(nil)
	require.NoError(t, err)
	require.Equal(t, "", cfg.FileStoragePath)
}

func TestParse_FileStoragePath_Flag(t *testing.T) {
	t.Setenv("FILE_STORAGE_PATH", "")

	cfg, err := config.Parse([]string{"-f", "/tmp/test.json"})
	require.NoError(t, err)
	require.Equal(t, "/tmp/test.json", cfg.FileStoragePath)
}

func TestParse_FileStoragePath_EnvOverridesFlag(t *testing.T) {
	t.Setenv("FILE_STORAGE_PATH", "/env/test.json")

	cfg, err := config.Parse([]string{"-f", "/flag/test.json"})
	require.NoError(t, err)
	require.Equal(t, "/env/test.json", cfg.FileStoragePath)
}

func TestParse_DatabaseDSN_Default(t *testing.T) {
	t.Setenv("DATABASE_DSN", "")

	cfg, err := config.Parse(nil)
	require.NoError(t, err)
	require.Equal(t, "", cfg.DatabaseDSN)
}

func TestParse_DatabaseDSN_Flag(t *testing.T) {
	t.Setenv("DATABASE_DSN", "")

	cfg, err := config.Parse([]string{"-d", "postgres://user:pass@localhost:5432/shortener?sslmode=disable"})
	require.NoError(t, err)
	require.Equal(t, "postgres://user:pass@localhost:5432/shortener?sslmode=disable", cfg.DatabaseDSN)
}

func TestParse_DatabaseDSN_EnvOverridesFlag(t *testing.T) {
	t.Setenv("DATABASE_DSN", "postgres://env:env@localhost:5432/envdb?sslmode=disable")

	cfg, err := config.Parse([]string{"-d", "postgres://flag:flag@localhost:5432/flagdb?sslmode=disable"})
	require.NoError(t, err)
	require.Equal(t, "postgres://env:env@localhost:5432/envdb?sslmode=disable", cfg.DatabaseDSN)
}

func TestParse_AuditFlags(t *testing.T) {
	t.Setenv("AUDIT_FILE", "")
	t.Setenv("AUDIT_URL", "")

	cfg, err := config.Parse([]string{
		"--audit-file", "/tmp/audit.log",
		"--audit-url", "http://localhost:9090/audit",
	})
	require.NoError(t, err)

	require.Equal(t, "/tmp/audit.log", cfg.AuditFile)
	require.Equal(t, "http://localhost:9090/audit", cfg.AuditURL)
}

func TestParse_AuditEnvOverridesFlags(t *testing.T) {
	t.Setenv("AUDIT_FILE", "/env/audit.log")
	t.Setenv("AUDIT_URL", "http://localhost:9091/audit")

	cfg, err := config.Parse([]string{
		"--audit-file", "/flag/audit.log",
		"--audit-url", "http://localhost:9090/audit",
	})
	require.NoError(t, err)

	require.Equal(t, "/env/audit.log", cfg.AuditFile)
	require.Equal(t, "http://localhost:9091/audit", cfg.AuditURL)
}
