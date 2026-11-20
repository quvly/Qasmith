package openqasm

import (
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/quvly/Qasmith/pkg/ir"
)

func TestFloatParametersRoundTrip(t *testing.T) {
	for _, want := range []float64{math.Pi, math.MaxFloat64, math.SmallestNonzeroFloat64, -1e-250, 2, math.Copysign(0, -1)} {
		program := &ir.Program{Operations: []ir.Operation{ir.AllocateQubit{Qubit: 0},
			ir.ApplyGate{Gate: ir.RZ, Qubits: []ir.QubitID{0}, Parameters: []float64{want}}}}
		code, err := (Backend{}).Compile(program)
		if err != nil {
			t.Fatal(err)
		}
		_, rest, ok := strings.Cut(string(code), "rz(")
		literal, _, closed := strings.Cut(rest, ") q[0];")
		if !ok || !closed || !strings.ContainsAny(literal, ".e") {
			t.Fatalf("bad rotation syntax: %s", code)
		}
		got, err := strconv.ParseFloat(literal, 64)
		if err != nil || math.Float64bits(got) != math.Float64bits(want) {
			t.Fatalf("%q parsed as %g, want %g: %v", literal, got, want, err)
		}
	}
}
