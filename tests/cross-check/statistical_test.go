// Statistical tests for the Qasmith simulator's sampling.
//
// These tests verify that the simulator produces samples consistent with
// the theoretical probability distribution. The chi-square test at the
// documented significance level determines whether observed counts are
// consistent with the expected probabilities.
//
// Significance level: α = 0.001 (documented below). This conservative level
// avoids false failures from statistical fluctuation while still catching
// genuinely wrong distributions.
package crosscheck

import (
	"math"
	"path/filepath"
	"testing"

	"github.com/quvly/Qasmith/pkg/qasmith"
)

// significanceLevel is the probability of a false positive for the chi-square
// tests in this file. Set at 0.001 (0.1%) to avoid flaky tests.
const significanceLevel = 0.001

// chiSquareCritical returns the critical value for the chi-square distribution
// at the given significance level and degrees of freedom, using the Wilson-Hilferty
// approximation. This is accurate enough for our purposes (df ≥ 1).
func chiSquareCritical(alpha float64, df int) float64 {
	if df <= 0 {
		return 0
	}
	// Normal quantile for upper-tail probability alpha.
	// Use Abramowitz and Stegun rational approximation.
	p := 1 - alpha
	t := math.Sqrt(-2 * math.Log(1-p))
	z := t - (2.515517+0.802853*t+0.010328*t*t)/(1+1.432788*t+0.189269*t*t+0.001308*t*t*t)
	// Wilson-Hilferty: chi-square ≈ df * (1 - 2/(9*df) + z*sqrt(2/(9*df)))^3
	r := 1 - 2.0/(9*float64(df)) + z*math.Sqrt(2.0/(9*float64(df)))
	return float64(df) * r * r * r
}

// chiSquare computes the chi-square statistic from observed and expected counts.
func chiSquare(observed, expected []float64) float64 {
	var sum float64
	for i := range observed {
		if expected[i] > 0 {
			diff := observed[i] - expected[i]
			sum += diff * diff / expected[i]
		}
	}
	return sum
}

func TestChiSquareBellState(t *testing.T) {
	// Bell state |Φ+⟩ = (|00⟩ + |11⟩) / √2
	// Expected: 50% 00, 50% 11, 0% 01, 10
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "tests", "cross-check", "corpus", "bell.smith")
	c := qasmith.Compiler{}
	result, err := c.Run(path, 4000, 42)
	if err != nil {
		t.Fatal(err)
	}
	// result.Counts maps bit strings to counts.
	// Bell state should only produce 00 and 11.
	counts := result.Counts
	n00 := float64(counts["00"])
	n01 := float64(counts["01"])
	n10 := float64(counts["10"])
	n11 := float64(counts["11"])
	total := n00 + n01 + n10 + n11

	if total == 0 {
		t.Fatal("no counts")
	}

	// Expected: 50% 00, 50% 11, 0% others
	// For the chi-square test with df=1 (only two categories: 00 and 11)
	// we drop the zero-expectation categories.
	expected := []float64{total / 2, total / 2}
	observed := []float64{n00, n11}
	stat := chiSquare(observed, expected)
	critical := chiSquareCritical(significanceLevel, 1)
	if stat > critical {
		t.Errorf("Bell state chi-square = %.3f > critical %.3f (α=%.4f, df=1)", stat, critical, significanceLevel)
		t.Errorf("  observed: 00=%v, 01=%v, 10=%v, 11=%v", int(n00), int(n01), int(n10), int(n11))
	}
	// 01 and 10 should be exactly zero for the ideal simulator.
	if n01 != 0 || n10 != 0 {
		t.Errorf("Bell state produced unexpected outcomes: 01=%v, 10=%v", int(n01), int(n10))
	}
}

func TestChiSquareSuperposition(t *testing.T) {
	// H|0⟩ = (|0⟩ + |1⟩) / √2 → 50% 0, 50% 1
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "tests", "cross-check", "corpus", "superposition.smith")
	c := qasmith.Compiler{}
	result, err := c.Run(path, 4000, 42)
	if err != nil {
		t.Fatal(err)
	}
	counts := result.Counts
	n0 := float64(counts["0"])
	n1 := float64(counts["1"])
	total := n0 + n1
	if total == 0 {
		t.Fatal("no counts")
	}
	expected := []float64{total / 2, total / 2}
	observed := []float64{n0, n1}
	stat := chiSquare(observed, expected)
	critical := chiSquareCritical(significanceLevel, 1)
	if stat > critical {
		t.Errorf("superposition chi-square = %.3f > critical %.3f (α=%.4f)", stat, critical, significanceLevel)
	}
}

func TestChiSquareRxHalfPi(t *testing.T) {
	// Rx(π/2)|0⟩ = cos(π/4)|0⟩ - i·sin(π/4)|1⟩
	// P(0) = cos²(π/4) = 0.5, P(1) = sin²(π/4) = 0.5
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "tests", "cross-check", "corpus", "rx_half_pi.smith")
	c := qasmith.Compiler{}
	result, err := c.Run(path, 4000, 42)
	if err != nil {
		t.Fatal(err)
	}
	counts := result.Counts
	n0 := float64(counts["0"])
	n1 := float64(counts["1"])
	total := n0 + n1
	if total == 0 {
		t.Fatal("no counts")
	}
	expected := []float64{total / 2, total / 2}
	observed := []float64{n0, n1}
	stat := chiSquare(observed, expected)
	critical := chiSquareCritical(significanceLevel, 1)
	if stat > critical {
		t.Errorf("rx(pi/2) chi-square = %.3f > critical %.3f", stat, critical)
	}
}
