package integration

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/quvly/Qasmith/pkg/optimization/ising"
	"github.com/quvly/Qasmith/pkg/optimization/qaoa"
	"github.com/quvly/Qasmith/pkg/optimization/tsp"
	"github.com/quvly/Qasmith/pkg/qasmith"
)

func TestReferenceOpenQASMSyntax(t *testing.T) {
	python := os.Getenv("QASMITH_OPENQASM_PYTHON")
	if python == "" {
		t.Skip("set QASMITH_OPENQASM_PYTHON to a Python interpreter with openqasm3[parser]")
	}
	dir := t.TempDir()
	edges := filepath.Join(dir, "edges.smith")
	if err := os.WriteFile(edges, []byte(`fn main() -> MeasuredResult {
    let q = qubit(); rx(-0.0, q); ry(-1e-250, q); rz(1.7976931348623157e308, q); return measure(q);
}`), 0600); err != nil {
		t.Fatal(err)
	}
	const parse = `
import importlib.util  # Load before openqasm3 1.0.1 installs its import hook.
import sys
from openqasm3.parser import parse, QASM3ParsingError
program = parse(sys.stdin.read())
assert program.version == "3.0"
try:
    parse("OPENQASM 3.0; rx(1.0 q[0];")
except QASM3ParsingError:
    pass
else:
    raise AssertionError("parser accepted the malformed control")
`
	for _, path := range []string{"../../bell.smith", "../../tsp.smith", "../../program.smith", "../../examples/optimization/qubo-language/builder.smith", "../../examples/optimization/qubo-language/constraints.smith", "../../examples/optimization/qubo-language/composition.smith", "../../examples/optimization/qubo-language/packaged.smith", "../../examples/optimization/language/imported.smith", "../../examples/rotations.smith", "../../examples/register.smith", "../../examples/register_layer.smith", "../../examples/circuit_builder.smith", edges} {
		t.Run(filepath.Base(path), func(t *testing.T) {
			data, err := (qasmith.Compiler{}).Compile(path)
			if err != nil {
				t.Fatal(err)
			}
			command := exec.Command(python, "-c", parse)
			command.Stdin = bytes.NewReader(data)
			if output, err := command.CombinedOutput(); err != nil {
				t.Fatalf("reference parser rejected generated output: %v\n%s\n%s", err, output, data)
			}
		})
	}
}

func TestReferenceQAOASyntax(t *testing.T) {
	python := os.Getenv("QASMITH_OPENQASM_PYTHON")
	if python == "" {
		t.Skip("set QASMITH_OPENQASM_PYTHON for independent syntax checks")
	}
	rng := rand.New(rand.NewSource(808))
	circuits := make([]string, 0, 24)
	for trial := range 24 {
		n := 1 + trial%4
		var fields []ising.Field
		var pairs []ising.Coupling
		for i := range n {
			fields = append(fields, ising.Field{Index: i, Coefficient: (rng.Float64() - .5) * 2})
			for j := i + 1; j < n; j++ {
				pairs = append(pairs, ising.Coupling{I: i, J: j, Coefficient: (rng.Float64() - .5) * 2})
			}
		}
		h, err := ising.New(n, 1.25, fields, pairs)
		if err != nil {
			t.Fatal(err)
		}
		problem, err := qaoa.New(h)
		if err != nil {
			t.Fatal(err)
		}
		params := qaoa.Parameters{Gammas: []float64{-.3, .15}, Betas: []float64{.4, -.1}}
		if (trial/4)%2 == 0 {
			params.Gammas = params.Gammas[:1]
			params.Betas = params.Betas[:1]
		}
		code, err := problem.Export(params, qaoa.BuildOptions{Measure: true}, nil)
		if err != nil {
			t.Fatal(err)
		}
		circuits = append(circuits, string(code))
	}
	input, err := os.ReadFile("../../examples/optimization/tsp/problem.json")
	if err != nil {
		t.Fatal(err)
	}
	var problem tsp.Problem
	if err := json.Unmarshal(input, &problem); err != nil {
		t.Fatal(err)
	}
	app, err := tsp.Prepare(problem, tsp.EncodeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	code, err := app.QAOA().Export(qaoa.Parameters{Gammas: []float64{.4}, Betas: []float64{.3}}, qaoa.BuildOptions{Measure: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	circuits = append(circuits, string(code))
	data, err := json.Marshal(circuits)
	if err != nil {
		t.Fatal(err)
	}
	const verify = `
import importlib.util
import json
import sys
from openqasm3.parser import parse
for code in json.load(sys.stdin):
    assert parse(code).version == "3.0"
`
	command := exec.Command(python, "-c", verify)
	command.Stdin = bytes.NewReader(data)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("reference parser: %v\n%s", err, output)
	}
}
