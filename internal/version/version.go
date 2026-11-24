// Package version provides the Qasmith CLI version.
//
// The version is embedded from the VERSION file at the repository root via
// the go:embed directive. GoReleaser overrides it at build time through -ldflags so the
// released binary reports the git tag.
package version

import (
	_ "embed"
	"strings"
)

// Version is the Qasmith release version without the leading "v".
//
// It defaults to the contents of the repository VERSION file and is replaced
// at build time when -ldflags sets version.Version (see .goreleaser.yaml).
var Version = embedded

//go:embed version.txt
var embedded string

// String returns the display form "Qasmith v<version>".
func String() string {
	return "Qasmith v" + strings.TrimSpace(Version)
}
