// Command muninn serves, validates and reviews an AI-generated product atlas
// stored as files in a project repository.
package main

import (
	"os"

	"github.com/lamngockhuong/muninn/internal/cli"
)

// version is set at build time with -ldflags "-X main.version=<tag>".
var version = "dev"

func main() {
	os.Exit(cli.Execute(version))
}
