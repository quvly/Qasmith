package simulator_test

import (
	"math/cmplx"
	"reflect"
	"testing"

	"github.com/quvly/Qasmith/internal/simulator"
	"github.com/quvly/Qasmith/pkg/ir"
	"github.com/quvly/Qasmith/pkg/qasmith"
)

func TestCircuitBuilderStatevector(t *testing.T) {
	compiler := qasmith.Compiler{}
	program, err := compiler.Lower("../../examples/circuit_builder.smith")
	if err != nil {
		t.Fatal(err)
	}
	pairs := []struct {
		a, b  ir.QubitID
		angle float64
	}{
		{0, 1, .2}, {0, 2, .4}, {0, 3, .6}, {1, 2, .8}, {1, 3, 1}, {2, 3, 1.2},
	}
	if len(program.Operations) != 30 {
		t.Fatalf("want 4 allocations, 22 gates and 4 measurements, got %d operations", len(program.Operations))
	}
	for id := range 4 {
		if got := program.Operations[id].(ir.AllocateQubit).Qubit; got != ir.QubitID(id) {
			t.Fatalf("allocation %d = %d", id, got)
		}
		h := program.Operations[4+id].(ir.ApplyGate)
		if h.Gate != ir.H || len(h.Qubits) != 1 || h.Qubits[0] != ir.QubitID(id) {
			t.Fatalf("H gate %+v", h)
		}
		m := program.Operations[26+id].(ir.Measure)
		if m.Bit != ir.BitID(id) || m.Qubit != ir.QubitID(id) {
			t.Fatalf("measurement %+v", m)
		}
	}
	for i, pair := range pairs {
		start := 8 + 3*i
		want := []ir.ApplyGate{
			{Gate: ir.CX, Qubits: []ir.QubitID{pair.a, pair.b}},
			{Gate: ir.RZ, Qubits: []ir.QubitID{pair.b}, Parameters: []float64{pair.angle}},
			{Gate: ir.CX, Qubits: []ir.QubitID{pair.a, pair.b}},
		}
		for j, expected := range want {
			gate := program.Operations[start+j].(ir.ApplyGate)
			if len(gate.Origin.CallSites) != 2 {
				t.Fatalf("missing source call chain: %+v", gate.Origin)
			}
			gate.Origin = ir.Origin{}
			if !reflect.DeepEqual(gate, expected) {
				t.Fatalf("pair %d step %d = %+v, want %+v", i, j, gate, expected)
			}
		}
	}
	state, _, err := simulator.Statevector(program)
	if err != nil || len(state) != 16 {
		t.Fatalf("statevector = %v, %v", state, err)
	}
	for basis, got := range state {
		// Compute the diagonal phase analytically from the six bit parities.
		var phase float64
		for _, pair := range pairs {
			if ((basis>>pair.a)^(basis>>pair.b))&1 == 1 {
				phase += pair.angle / 2
			} else {
				phase -= pair.angle / 2
			}
		}
		want := cmplx.Exp(complex(0, phase)) / 4
		if !(cmplx.Abs(got-want) <= 1e-12) {
			t.Errorf("|%04b> = %v, want %v", basis, got, want)
		}
	}
	for _, path := range []string{"../../examples/register.smith", "../../examples/circuit_builder.smith"} {
		a, err := compiler.Run(path, 4096, 17)
		if err != nil {
			t.Fatal(err)
		}
		b, err := compiler.Run(path, 4096, 17)
		if err != nil || !reflect.DeepEqual(a, b) || len(a.Counts) != 16 {
			t.Fatalf("sampling %s: %v, %v", path, a, err)
		}
		for pattern, count := range a.Counts {
			if count < 190 || count > 325 {
				t.Errorf("%s outcome %s: %d; expected about 256", path, pattern, count)
			}
		}
	}
}
