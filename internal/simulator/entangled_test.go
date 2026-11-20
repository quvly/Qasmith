package simulator_test

import (
	"fmt"
	"math"
	"math/cmplx"
	"os"
	"path/filepath"
	"testing"

	"github.com/quvly/Qasmith/internal/simulator"
	"github.com/quvly/Qasmith/pkg/ir"
	"github.com/quvly/Qasmith/pkg/qasmith"
)

func assertState(t *testing.T, program *ir.Program, want [4]complex128) {
	t.Helper()
	state, _, err := simulator.Statevector(program)
	if err != nil {
		t.Fatal(err)
	}
	if len(state) != len(want) {
		t.Fatalf("got %d amplitudes, want %d", len(state), len(want))
	}
	norm := 0.0
	for i, got := range state {
		norm += real(got)*real(got) + imag(got)*imag(got)
		if !(cmplx.Abs(got-want[i]) <= 1e-12) {
			t.Errorf("amplitude |%02b> = %v, want %v", i, got, want[i])
		}
	}
	if !(math.Abs(norm-1) <= 1e-12) {
		t.Errorf("norm = %g", norm)
	}
}

func TestRZZUnitary(t *testing.T) {
	module, err := os.ReadFile("../../examples/modules/rotations.smith")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "rotations.smith"), module, 0600); err != nil {
		t.Fatal(err)
	}
	main := filepath.Join(dir, "main.smith")
	for _, angle := range []float64{0, 0.73, -1.23, math.Pi} {
		even, odd := cmplx.Exp(complex(0, -angle/2)), cmplx.Exp(complex(0, angle/2))
		diagonal := [4]complex128{even, odd, odd, even}
		tests := []struct {
			name, prepare string
			want          [4]complex128
		}{
			{"00", "", [4]complex128{diagonal[0], 0, 0, 0}},
			{"01", "x(a);", [4]complex128{0, diagonal[1], 0, 0}},
			{"10", "x(b);", [4]complex128{0, 0, diagonal[2], 0}},
			{"11", "x(a); x(b);", [4]complex128{0, 0, 0, diagonal[3]}},
			{"superposition", "h(a); h(b);", [4]complex128{even / 2, odd / 2, odd / 2, even / 2}},
		}
		for _, tc := range tests {
			t.Run(fmt.Sprintf("%g/%s", angle, tc.name), func(t *testing.T) {
				source := fmt.Sprintf(`import { rzz } from "./rotations.smith";
fn main() -> MeasuredResult {
    let a = qubit();
    let b = qubit();
    %s
    rzz(%g, a, b);
    return measure(a, b);
}`, tc.prepare, angle)
				if err := os.WriteFile(main, []byte(source), 0600); err != nil {
					t.Fatal(err)
				}
				program, err := (qasmith.Compiler{}).Lower(main)
				if err != nil {
					t.Fatal(err)
				}
				assertState(t, program, tc.want)
			})
		}
	}
}

func TestRotationsOnEntangledState(t *testing.T) {
	// Basis indices are ordered by q1 then q0. Test rotating each Bell-state qubit.
	k := complex(math.Sqrt2/2, 0)
	c, s := complex(math.Cos(math.Pi/6), 0), complex(0.5, 0)
	for _, tc := range []struct {
		gate   ir.Gate
		target ir.QubitID
		want   [4]complex128
	}{
		{ir.RX, 0, [4]complex128{k * c, -1i * k * s, -1i * k * s, k * c}},
		{ir.RX, 1, [4]complex128{k * c, -1i * k * s, -1i * k * s, k * c}},
		{ir.RY, 0, [4]complex128{k * c, k * s, -k * s, k * c}},
		{ir.RY, 1, [4]complex128{k * c, -k * s, k * s, k * c}},
		{ir.RZ, 0, [4]complex128{k * (c - 1i*s), 0, 0, k * (c + 1i*s)}},
		{ir.RZ, 1, [4]complex128{k * (c - 1i*s), 0, 0, k * (c + 1i*s)}},
	} {
		t.Run(fmt.Sprintf("%s/q%d", tc.gate, tc.target), func(t *testing.T) {
			assertState(t, &ir.Program{Operations: []ir.Operation{
				ir.AllocateQubit{Qubit: 0}, ir.AllocateQubit{Qubit: 1},
				ir.ApplyGate{Gate: ir.H, Qubits: []ir.QubitID{0}},
				ir.ApplyGate{Gate: ir.CX, Qubits: []ir.QubitID{0, 1}},
				ir.ApplyGate{Gate: tc.gate, Qubits: []ir.QubitID{tc.target}, Parameters: []float64{math.Pi / 3}},
			}}, tc.want)
		})
	}
}
