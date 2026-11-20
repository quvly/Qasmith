package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestExample(t *testing.T) {
	var out bytes.Buffer
	if err := run(&out); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`"schema": "qasmith.qubo.v1"`, `"schema": "qasmith.ising.v1"`,
		"Assignment [1 0 0] has energy -1.75 in both representations",
		"Best assignment [1 0 0] with objective energy -1.75",
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("missing %q in %s", want, out.String())
		}
	}
}
