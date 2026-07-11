package buildinfo

import (
	"fmt"
	"io"
	"os"
)

const NotAvailable = "N/A"

type Info struct {
	Version string
	Date    string
	Commit  string
}

func Print(info Info) {
	fprint(os.Stdout, info)
}

func fprint(w io.Writer, info Info) {
	fmt.Fprintf(
		w,
		"Build version: %s\nBuild date: %s\nBuild commit: %s\n",
		info.Version,
		info.Date,
		info.Commit,
	)
}
