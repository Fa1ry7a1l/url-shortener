package main

import (
	"testing"

	"github.com/stretchr/testify/require"
	"honnef.co/go/tools/staticcheck"
)

func TestAllAnalyzersContainsEveryStaticcheckAnalyzer(t *testing.T) {
	registered := make(map[string]struct{})
	for _, analyzer := range allAnalyzers() {
		registered[analyzer.Name] = struct{}{}
	}

	for _, analyzer := range staticcheck.Analyzers {
		require.Contains(t, registered, analyzer.Analyzer.Name)
	}
	require.Contains(t, registered, "S1000")
	require.Contains(t, registered, "bodyclose")
	require.Contains(t, registered, "nilerr")
	require.Contains(t, registered, "noosexit")
}
