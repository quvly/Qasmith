package simulator

import (
	"math"
	"reflect"
	"testing"

	"github.com/quvly/Qasmith/pkg/ir"
)

func TestStatevectorAndSampling(t *testing.T) {
	q0, q1 := ir.AllocateQubit{Qubit: 0}, ir.AllocateQubit{Qubit: 1}
	h := ir.ApplyGate{Gate: ir.H, Qubits: []ir.QubitID{0}}
	x := ir.ApplyGate{Gate: ir.X, Qubits: []ir.QubitID{0}}
	cx := ir.ApplyGate{Gate: ir.CX, Qubits: []ir.QubitID{0, 1}}
	m0, m1 := ir.Measure{Qubit: 0, Bit: 0}, ir.Measure{Qubit: 1, Bit: 1}
	for _, test := range []struct {
		name   string
		ops    []ir.Operation
		counts map[string]int
	}{
		{"x", []ir.Operation{q0, x, m0}, map[string]int{"1": 2000}},
		{"h", []ir.Operation{q0, h, m0}, map[string]int{"0": 1000, "1": 1000}},
		{"bell", []ir.Operation{q0, q1, h, cx, m0, m1}, map[string]int{"00": 1000, "11": 1000}},
		{"ordering", []ir.Operation{q0, q1, x, ir.Measure{Qubit: 1, Bit: 0}, ir.Measure{Qubit: 0, Bit: 1}}, map[string]int{"01": 2000}},
		{"marginal", []ir.Operation{q0, q1, h, cx, m0}, map[string]int{"0": 1000, "1": 1000}},
	} {
		t.Run(test.name, func(t *testing.T) {
			p := &ir.Program{Operations: test.ops}
			state, _, err := Statevector(p)
			if err != nil {
				t.Fatal(err)
			}
			var norm float64
			for _, a := range state {
				norm += real(a)*real(a) + imag(a)*imag(a)
			}
			if math.Abs(norm-1) > 1e-12 {
				t.Fatalf("norm %g", norm)
			}
			result, err := Run(p, 2000, 17)
			if err != nil {
				t.Fatal(err)
			}
			same, err := Run(p, 2000, 17)
			if err != nil || !reflect.DeepEqual(result, same) {
				t.Fatalf("seeded results differ: %v %v", result, same)
			}
			if len(result.Counts) != len(test.counts) {
				t.Fatalf("unexpected outcomes %v", result.Counts)
			}
			for key, want := range test.counts {
				got := result.Counts[key]
				if got == 0 || math.Abs(float64(got-want)) > 100 || (want == 2000 && got != want) {
					t.Errorf("%s got %d, expected approximately %d", key, got, want)
				}
			}
		})
	}
}

func TestQubitLimit(t *testing.T) {
	p := &ir.Program{}
	for i := 0; i <= MaxQubits; i++ {
		p.Operations = append(p.Operations, ir.AllocateQubit{Qubit: ir.QubitID(i)})
	}
	if _, _, err := Statevector(p); err == nil {
		t.Fatal("simulated a circuit above the qubit limit")
	}
}
