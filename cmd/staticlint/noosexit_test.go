package main

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestNoOSExitAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(
		t,
		testdata,
		NoOSExitAnalyzer,
		"direct",
		"alias",
		"dotimport",
		"generated",
		"indirect",
		"notmain",
	)
}
