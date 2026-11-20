package ir

import "testing"

func TestValidate(t *testing.T) {
	q0, q1 := AllocateQubit{Qubit: 0}, AllocateQubit{Qubit: 1}
	h := ApplyGate{Gate: H, Qubits: []QubitID{0}}
	measure := Measure{Qubit: 0, Bit: 0}
	valid := &Program{[]Operation{q0, h, q1, ApplyGate{Gate: CX, Qubits: []QubitID{0, 1}}, measure, Measure{Qubit: 1, Bit: 1}}}
	if layout, err := valid.Validate(); err != nil || layout != (Layout{2, 2}) {
		t.Fatalf("layout %v: %v", layout, err)
	}
	for _, ops := range [][]Operation{
		{q1}, {q0, q0}, {h}, {q0, ApplyGate{Gate: "MYSTERY", Qubits: []QubitID{0}}},
		{q0, ApplyGate{Gate: CX, Qubits: []QubitID{0, 0}}}, {q0, ApplyGate{Gate: H}},
		{q0, Measure{Qubit: 0, Bit: 1}}, {q0, measure, h}, {q0, measure, measure}, {q0, measure, q1},
	} {
		if _, err := (&Program{ops}).Validate(); err == nil {
			t.Errorf("accepted invalid operations %v", ops)
		}
	}
}
