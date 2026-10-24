package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/kOlapsis/speckit-board/internal/cli"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	v := version
	if bi, ok := debug.ReadBuildInfo(); ok && v == "dev" && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		v = bi.Main.Version
	}
	if err := cli.New(cli.Info{Version: v, Commit: commit, Date: date}).Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "speckit-board:", err)
		os.Exit(1)
	}
}
