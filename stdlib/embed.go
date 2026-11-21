// Package stdlib embeds the Qasmith standard library files.
package stdlib

import "embed"

// CoreFS contains the core standard library .smith files.
//go:embed core/*.smith
var CoreFS embed.FS
