// Command tonychan is Tony Chan's site, yukccy.com, in a terminal.
package main

import (
	"os"
	"runtime/debug"

	"github.com/yukccy/tonychan-cli/internal/cli"
)

// version is set by GoReleaser. `go install …@v1.2.3` builds leave it empty, so fall back to
// the module version Go recorded in the binary.
var version = ""

func main() {
	os.Exit(cli.Execute(resolveVersion()))
}

func resolveVersion() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}
