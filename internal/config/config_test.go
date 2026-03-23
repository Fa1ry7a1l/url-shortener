package config_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Fa1ry7a1l/url-shortener/internal/config"
)

func TestParse_Defaults(t *testing.T) {
	t.Setenv("SERVER_ADDRESS", "")
	t.Setenv("BASE_URL", "")

	cfg, err := config.Parse(nil)
	require.NoError(t, err)

	require.Equal(t, "localhost:8080", cfg.Addr)
	require.Equal(t, "http://localhost:8080", cfg.BaseURL)
	require.Equal(t, 8, cfg.IDLength)
}

func TestParse_Flags(t *testing.T) {
	t.Setenv("SERVER_ADDRESS", "")
	t.Setenv("BASE_URL", "")

	cfg, err := config.Parse([]string{
		"-a", "localhost:9999",
		"-b", "http://localhost:1111/",
		"-l", "12",
	})
	require.NoError(t, err)

	require.Equal(t, "localhost:9999", cfg.Addr)
	require.Equal(t, "http://localhost:1111", cfg.BaseURL)
	require.Equal(t, 12, cfg.IDLength)
}

func TestParse_EnvOverridesFlags(t *testing.T) {
	t.Setenv("SERVER_ADDRESS", "localhost:7777")
	t.Setenv("BASE_URL", "http://localhost:7777/")

	cfg, err := config.Parse([]string{
		"-a", "localhost:9999",
		"-b", "http://localhost:1111",
		"-l", "12",
	})
	require.NoError(t, err)

	require.Equal(t, "localhost:7777", cfg.Addr)
	require.Equal(t, "http://localhost:7777", cfg.BaseURL)
	require.Equal(t, 12, cfg.IDLength)
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
