package syntax

import "testing"

// FuzzLex exercises the Smith lexer with arbitrary input.
// The seed corpus under testdata/fuzz/FuzzLex contains the repository's
// example programs.
func FuzzLex(f *testing.F) {
	f.Fuzz(func(t *testing.T, source string) {
		// Lexemes must not panic on any input.
		_ = Lexemes("fuzz.smith", source)
	})
}
