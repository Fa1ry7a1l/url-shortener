package buildinfo

import (
	"bytes"
	"testing"
)

func TestFprint(t *testing.T) {
	var buf bytes.Buffer

	fprint(&buf, Info{
		Version: "v1.2.3",
		Date:    "2026-06-27",
		Commit:  "abc123",
	})

	const want = "Build version: v1.2.3\nBuild date: 2026-06-27\nBuild commit: abc123\n"
	if got := buf.String(); got != want {
		t.Fatalf("fprint() = %q, want %q", got, want)
	}
}
