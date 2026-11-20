package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/quvly/Qasmith/pkg/optimization/qubo"
)

func TestExample(t *testing.T) {
	var output bytes.Buffer
	dir := t.TempDir()
	if err := run(context.Background(), &output, dir); err != nil {
		t.Fatal(err)
	}
	for _, needle := range []string{"distance: 80, tied tours: 2", "Exhaustive QUBO minimum: 80", "Optimized feasible samples:"} {
		if !strings.Contains(output.String(), needle) {
			t.Fatalf("example output missing %q", needle)
		}
	}
	data, err := os.ReadFile(filepath.Join(dir, "tsp.qubo.json"))
	if err != nil {
		t.Fatal(err)
	}
	var model qubo.Model
	if err := json.Unmarshal(data, &model); err != nil || model.Variables() != 9 {
		t.Fatalf("model variables %d, %v", model.Variables(), err)
	}
	for _, name := range []string{"initial.qasm", "optimized.qasm"} {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil || !bytes.Contains(data, []byte("qubit[9] q;")) || bytes.Count(data, []byte(" = measure ")) != 9 {
			t.Fatalf("invalid %s: %v", name, err)
		}
	}
}
