package ir

import (
	"math"
	"testing"
)

func TestRotationOperands(t *testing.T) {
	for _, gate := range []Gate{RX, RY, RZ} {
		for _, tc := range []struct {
			qubits     []QubitID
			parameters []float64
			valid      bool
		}{
			{[]QubitID{0}, []float64{-math.Pi}, true},
			{[]QubitID{0}, []float64{math.MaxFloat64}, true},
			{[]QubitID{0}, nil, false},
			{[]QubitID{0}, []float64{0, 1}, false},
			{[]QubitID{0}, []float64{math.NaN()}, false},
			{[]QubitID{0}, []float64{math.Inf(1)}, false},
			{[]QubitID{0}, []float64{math.Inf(-1)}, false},
			{nil, []float64{0}, false},
			{[]QubitID{0, 1}, []float64{0}, false},
			{[]QubitID{2}, []float64{0}, false},
		} {
			program := Program{Operations: []Operation{
				AllocateQubit{Qubit: 0}, AllocateQubit{Qubit: 1},
				ApplyGate{Gate: gate, Qubits: tc.qubits, Parameters: tc.parameters},
			}}
			if _, err := program.Validate(); (err == nil) != tc.valid {
				t.Errorf("%s qubits %v parameters %v: valid=%v, got %v", gate, tc.qubits, tc.parameters, tc.valid, err)
			}
		}
	}
	p := Program{Operations: []Operation{
		AllocateQubit{Qubit: 0},
		ApplyGate{Gate: X, Qubits: []QubitID{0}, Parameters: []float64{0}},
	}}
	if _, err := p.Validate(); err == nil {
		t.Fatal("accepted parameter on X")
	}
}
