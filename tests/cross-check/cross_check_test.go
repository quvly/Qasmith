// Package crosscheck verifies the Qasmith simulator against analytically
// computed statevectors for a corpus of circuits.
//
// Each test circuit has a known expected statevector (up to a global phase).
// The test compiles and simulates each circuit, then compares the output
// statevector against the expected amplitudes with a tolerance of 1e-9.
//
// The expected amplitudes are computed independently using the standard
// matrix representation of each gate:
//
//	H = (1/sqrt(2)) * [[1, 1], [1, -1]]
//	X = [[0, 1], [1, 0]]
//	Rx(θ) = [[cos(θ/2), -i*sin(θ/2)], [-i*sin(θ/2), cos(θ/2)]]
//	Ry(θ) = [[cos(θ/2), -sin(θ/2)], [sin(θ/2), cos(θ/2)]]
//	Rz(θ) = [[exp(-iθ/2), 0], [0, exp(iθ/2)]]
//	CX = controlled NOT (CNOT)
package crosscheck

import (
	"math"
	"math/cmplx"
	"path/filepath"
	"strings"
	"testing"

	"github.com/quvly/Qasmith/internal/simulator"
	"github.com/quvly/Qasmith/pkg/qasmith"
)

const tolerance = 1e-9

type complexSlice []complex128

// corpus maps a corpus .smith filename to its expected statevector amplitudes
// listed in |00...⟩, |01...⟩, ... order.
var corpus = map[string]complexSlice{
	"identity.smith":           {1, 0},
	"x_gate.smith":             {0, 1},
	"x_twice.smith":            {1, 0},
	"superposition.smith":      {1 / math.Sqrt2, 1 / math.Sqrt2},
	"h_double.smith":           {1, 0},
	"bell.smith":               {1 / math.Sqrt2, 0, 0, 1 / math.Sqrt2},
	"controlled_x.smith":       {1 / math.Sqrt2, 0, 0, 1 / math.Sqrt2},
	"bell_measure_first.smith": {1 / math.Sqrt2, 0, 0, 1 / math.Sqrt2},
	"ghz.smith":                {1 / math.Sqrt2, 0, 0, 0, 0, 0, 0, 1 / math.Sqrt2},
	"register3.smith":          {1 / math.Sqrt2, 0, 0, 0, 0, 0, 0, 1 / math.Sqrt2},
	"two_bell.smith":           bellPairTensorProduct(),
	"rx_pi.smith":              {0, -1i},
	"ry_pi.smith":              {0, 1},
	"rz_pi.smith":              expectedHRzH(math.Pi),
	"rx_half_pi.smith":         expectedRx(math.Pi / 2),
	"ry_half_pi.smith":         expectedRy(math.Pi / 2),
	"rx_ry_rz.smith":           expectedRxRyRz(),
	"cx_reverse.smith":         {0, 0, 1, 0},
	"h_rz_h.smith":             expectedHRzH(math.Pi / 2),
	"measure_order.smith":      {1 / math.Sqrt2, 0, 0, 1 / math.Sqrt2},
}

func bellPairTensorProduct() complexSlice {
	bell := complexSlice{1 / math.Sqrt2, 0, 0, 1 / math.Sqrt2}
	result := make(complexSlice, 16)
	for i, a := range bell {
		for j, b := range bell {
			result[i*4+j] = a * b
		}
	}
	return result
}

func expectedRx(theta float64) complexSlice {
	c := math.Cos(theta / 2)
	s := math.Sin(theta / 2)
	return complexSlice{complex(c, 0), complex(0, -s)}
}

func expectedRy(theta float64) complexSlice {
	c := math.Cos(theta / 2)
	s := math.Sin(theta / 2)
	return complexSlice{complex(c, 0), complex(s, 0)}
}

// expectedHRzH computes H·Rz(θ)·H|0⟩.
// H|0⟩ = (|0⟩+|1⟩)/√2
// Rz(θ)(|0⟩+|1⟩)/√2 = (e^(-iθ/2)|0⟩ + e^(iθ/2)|1⟩)/√2
// H applied to that:
// = (e^(-iθ/2)(|0⟩+|1⟩) + e^(iθ/2)(|0⟩-|1⟩)) / (2)
// = (e^(-iθ/2)+e^(iθ/2))/2 |0⟩ + (e^(-iθ/2)-e^(iθ/2))/2 |1⟩
// = cos(θ/2)|0⟩ - i·sin(θ/2)|1⟩
func expectedHRzH(theta float64) complexSlice {
	c := math.Cos(theta / 2)
	s := math.Sin(theta / 2)
	return complexSlice{complex(c, 0), complex(0, -s)}
}

func expectedRxRyRz() complexSlice {
	c4 := math.Cos(math.Pi / 8)
	s4 := math.Sin(math.Pi / 8)
	state := complexSlice{complex(c4, 0), complex(0, -s4)}

	c3 := math.Cos(math.Pi / 6)
	s3 := math.Sin(math.Pi / 6)
	state = complexSlice{
		complex(c3, 0)*state[0] - complex(s3, 0)*state[1],
		complex(s3, 0)*state[0] + complex(c3, 0)*state[1],
	}

	rz0 := cmplx.Exp(complex(0, -math.Pi/12))
	rz1 := cmplx.Exp(complex(0, math.Pi/12))
	state = complexSlice{rz0 * state[0], rz1 * state[1]}

	return state
}

func equalUpToGlobalPhase(got, want complexSlice) bool {
	if len(got) != len(want) {
		return false
	}
	var phase complex128
	for i, w := range want {
		if cmplx.Abs(w) > tolerance {
			if cmplx.Abs(got[i]) < tolerance {
				return false
			}
			phase = got[i] / w
			break
		}
	}
	if phase == 0 {
		for _, g := range got {
			if cmplx.Abs(g) > tolerance {
				return false
			}
		}
		return true
	}
	for i := range want {
		diff := got[i] - phase*want[i]
		if cmplx.Abs(diff) > tolerance {
			return false
		}
	}
	return true
}

func TestCorpusStatevectors(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	corpusDir := filepath.Join(root, "tests", "cross-check", "corpus")
	for name, want := range corpus {
		t.Run(strings.TrimSuffix(name, ".smith"), func(t *testing.T) {
			path := filepath.Join(corpusDir, name)
			c := qasmith.Compiler{}
			program, err := c.Lower(path)
			if err != nil {
				t.Fatalf("lower: %v", err)
			}
			got, _, err := simulator.Statevector(program)
			if err != nil {
				t.Fatalf("statevector: %v", err)
			}
			if !equalUpToGlobalPhase(got, want) {
				t.Errorf("statevector mismatch for %s", name)
				t.Errorf("  got:  %v", got)
				t.Errorf("  want: %v", want)
			}
		})
	}
}
