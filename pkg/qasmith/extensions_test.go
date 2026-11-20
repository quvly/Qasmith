package qasmith_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/quvly/Qasmith/pkg/extension"
	"github.com/quvly/Qasmith/pkg/ir"
	"github.com/quvly/Qasmith/pkg/qasmith"
)

func flipTwice() extension.Intrinsic {
	return extension.Intrinsic{
		Namespace: "testgates", Name: "flipTwice",
		Signature: extension.Signature{Parameters: []extension.Type{extension.Qubit}, Return: extension.Void},
		Lower: func(ctx extension.LoweringContext, args []extension.Value) (extension.Value, error) {
			for range 2 {
				if err := ctx.ApplyGate(ir.X, args[0].Qubit); err != nil {
					return extension.Value{}, err
				}
			}
			return extension.Value{Type: extension.Void}, nil
		},
	}
}

func TestHostRegistryAndPass(t *testing.T) {
	registry := &extension.Registry{}
	intrinsic := flipTwice()
	if err := registry.Register(intrinsic); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(intrinsic); err == nil {
		t.Fatal("duplicate accepted")
	}
	intrinsic.Signature.Parameters[0] = extension.Int
	got, ok := registry.Lookup("host:testgates", "flipTwice")
	if !ok || got.Signature.Parameters[0] != extension.Qubit {
		t.Fatal("registration aliased caller memory")
	}
	got.Signature.Parameters[0] = extension.Int
	path := fixture(t, map[string]string{"main.smith": `import { flipTwice as flip } from "host:testgates";
		fn main() -> MeasuredResult { let q = qubit(); flip(q); return measure(q); }`})
	plain := qasmith.Compiler{Registry: registry}
	circuit, err := plain.Lower(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(circuit.Operations) != 4 {
		t.Fatalf("expected two emitted gates, got %v", circuit.Operations)
	}
	for _, op := range circuit.Operations[1:3] {
		gate, ok := op.(ir.ApplyGate)
		if !ok || gate.Gate != ir.X || len(gate.Qubits) != 1 || gate.Qubits[0] != 0 {
			t.Fatalf("invalid host gate %v", op)
		}
	}
	optimized := qasmith.Compiler{Registry: registry, Passes: []ir.Pass{qasmith.CancelAdjacentX{}}}
	circuit, err = optimized.Lower(path)
	if err != nil || len(circuit.Operations) != 2 {
		t.Fatalf("optimizer result %v, %v", circuit, err)
	}
	if err := (qasmith.Compiler{}).Check(path); err == nil || !strings.Contains(err.Error(), "unregistered") {
		t.Fatalf("unregistered import error: %v", err)
	}
	bad := fixture(t, map[string]string{"main.smith": `import { flipTwice } from "host:testgates";
		fn main() -> MeasuredResult { let q = qubit(); flipTwice(1); return measure(q); }`})
	if err := plain.Check(bad); err == nil || !strings.Contains(err.Error(), "expects Qubit") {
		t.Fatalf("bad argument error: %v", err)
	}
}

func TestInvalidHostEmission(t *testing.T) {
	path := fixture(t, map[string]string{"main.smith": `import { flipTwice } from "host:testgates";
		fn main() -> MeasuredResult { let q = qubit(); let r = qubit(); flipTwice(q); return measure(r); }`})
	for _, tc := range []struct {
		name     string
		gate     ir.Gate
		ids      []ir.QubitID
		validate func([]extension.Value) error
		want     string
	}{
		{"invalid gate", "invalid", []ir.QubitID{0}, nil, "invalid operands"},
		{"invalid arity", ir.H, nil, nil, "invalid operands"},
		{"foreign qubit", ir.X, []ir.QubitID{1}, nil, "outside intrinsic arguments"},
		{"duplicate operands", ir.CX, []ir.QubitID{0, 0}, nil, "invalid operands"},
		{"validation", ir.X, []ir.QubitID{0}, func([]extension.Value) error { return fmt.Errorf("rejected argument") }, "rejected argument"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			intrinsic := flipTwice()
			intrinsic.Validate = tc.validate
			intrinsic.Lower = func(ctx extension.LoweringContext, _ []extension.Value) (extension.Value, error) {
				_ = ctx.ApplyGate(tc.gate, tc.ids...)
				return extension.Value{Type: extension.Void}, nil
			}
			registry := &extension.Registry{}
			if err := registry.Register(intrinsic); err != nil {
				t.Fatal(err)
			}
			compiler := qasmith.Compiler{Registry: registry}
			if err := compiler.Check(path); err != nil {
				t.Fatalf("check ran a lowering callback: %v", err)
			}
			if ir, err := compiler.Lower(path); ir != nil || err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("wanted %q, got %v, %v", tc.want, ir, err)
			}
		})
	}
}

type exampleBackend struct{ reject bool }

func (exampleBackend) Name() string { return "example" }
func (b exampleBackend) Compile(program *ir.Program) ([]byte, error) {
	if b.reject {
		return nil, fmt.Errorf("rejected circuit")
	}
	layout, err := program.Validate()
	return []byte(fmt.Sprintf("%d qubits, %d bits", layout.Qubits, layout.Bits)), err
}

func TestBackendInterface(t *testing.T) {
	path := fixture(t, map[string]string{"main.smith": `fn main() -> MeasuredResult { let q = qubit(); return measure(q); }`})
	compiler := qasmith.Compiler{Backend: exampleBackend{}}
	got, err := compiler.Compile(path)
	if err != nil || string(got) != "1 qubits, 1 bits" {
		t.Fatalf("backend = %s, %v", got, err)
	}
	compiler.Backend = exampleBackend{reject: true}
	if _, err := compiler.Compile(path); err == nil || !strings.Contains(err.Error(), "E5000") {
		t.Fatalf("backend error = %v", err)
	}
}
