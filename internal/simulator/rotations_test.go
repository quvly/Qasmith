package simulator

import (
	"math"
	"math/cmplx"
	"testing"

	"github.com/quvly/Qasmith/pkg/ir"
)

func TestRotationAmplitudes(t *testing.T) {
	rotation := func(gate ir.Gate, angle float64) ir.ApplyGate {
		return ir.ApplyGate{Gate: gate, Qubits: []ir.QubitID{0}, Parameters: []float64{angle}}
	}
	x := ir.ApplyGate{Gate: ir.X, Qubits: []ir.QubitID{0}}
	h := ir.ApplyGate{Gate: ir.H, Qubits: []ir.QubitID{0}}
	k := complex(math.Sqrt2/2, 0)
	for _, tc := range []struct {
		name    string
		ops     []ir.Operation
		want    [2]complex128
		outcome string
	}{
		{"RX(pi)", []ir.Operation{rotation(ir.RX, math.Pi)}, [2]complex128{0, -1i}, "1"},
		{"RY(pi)", []ir.Operation{rotation(ir.RY, math.Pi)}, [2]complex128{0, 1}, "1"},
		{"RX(pi/2)", []ir.Operation{rotation(ir.RX, math.Pi/2)}, [2]complex128{k, -1i * k}, ""},
		{"RY from one", []ir.Operation{x, rotation(ir.RY, math.Pi/2)}, [2]complex128{-k, k}, ""},
		{"RZ relative phase", []ir.Operation{h, rotation(ir.RZ, math.Pi)}, [2]complex128{-1i * k, 1i * k}, ""},
		{"RZ interference", []ir.Operation{h, rotation(ir.RZ, math.Pi), h}, [2]complex128{0, -1i}, "1"},
		{"inverse RX", []ir.Operation{rotation(ir.RX, .23), rotation(ir.RX, -.23)}, [2]complex128{1, 0}, "0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			program := &ir.Program{Operations: []ir.Operation{ir.AllocateQubit{Qubit: 0}}}
			program.Operations = append(program.Operations, tc.ops...)
			program.Operations = append(program.Operations, ir.Measure{Qubit: 0, Bit: 0})
			state, _, err := Statevector(program)
			if err != nil {
				t.Fatal(err)
			}
			norm := 0.0
			for i, amplitude := range state {
				norm += math.Pow(cmplx.Abs(amplitude), 2)
				if cmplx.Abs(amplitude-tc.want[i]) > 1e-12 {
					t.Errorf("amplitude %d = %v, want %v", i, amplitude, tc.want[i])
				}
			}
			if math.Abs(norm-1) > 1e-12 {
				t.Errorf("norm = %g", norm)
			}
			if tc.outcome != "" {
				result, err := Run(program, 100, 17)
				if err != nil || result.Counts[tc.outcome] != 100 {
					t.Fatalf("sampled %+v, %v", result, err)
				}
			}
		})
	}
}
