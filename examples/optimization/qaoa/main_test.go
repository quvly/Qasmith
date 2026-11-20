package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExample(t *testing.T) {
	var out bytes.Buffer
	path := filepath.Join(t.TempDir(), "result.qasm")
	if err := run(context.Background(), &out, filepath.Join(filepath.Dir(path), "initial.qasm"), path); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(data), " = measure ") != 3 || !strings.Contains(out.String(), "Optimal sample fraction:") {
		t.Fatalf("output %q / %s", out.String(), data)
	}
}
