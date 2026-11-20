package qubo_test

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/quvly/Qasmith/pkg/optimization/qubo"
)

func TestPenaltyEnergyLandscapes(t *testing.T) {
	for _, tc := range []struct {
		name           string
		weights        []float64
		target, lambda float64
	}{
		{"weighted", []float64{-1.75, 0, 2.125, .2}, .63, 3.1},
		{"exactly one", []float64{1, 1, 1, 1}, 1, 10},
		{"zero strength", []float64{1, 0, -1, 2}, -3, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := builder(t, len(tc.weights))
			must(t, b.SetOffset(.25))
			if tc.name == "exactly one" {
				must(t, b.AddExactlyOne([]int{3, 1, 0, 2}, tc.lambda))
			} else {
				weights := make([]qubo.Weight, len(tc.weights))
				for i, c := range tc.weights {
					weights[len(weights)-1-i] = qubo.Weight{Index: i, Coefficient: c}
				}
				must(t, b.AddSquaredPenalty(weights, tc.target, tc.lambda))
			}
			m := build(t, b)
			assignments(len(tc.weights), func(bits []bool) {
				residual := -tc.target
				for i, bit := range bits {
					if bit {
						residual += tc.weights[i]
					}
				}
				want := .25 + tc.lambda*residual*residual
				got, err := m.Evaluate(bits)
				if err != nil || !near(got, want) {
					t.Fatalf("bits %v: want %g, got %g, %v", bits, want, got, err)
				}
			})
		})
	}
}

func TestCompositionAndScaling(t *testing.T) {
	base, err := qubo.FromMatrix([][]float64{{1, .5}, {.25, -1}}, .25)
	must(t, err)
	penalty := builder(t, 2)
	must(t, penalty.AddExactlyOne([]int{0, 1}, 2))
	penaltyModel := build(t, penalty)
	scaled, err := base.Scale(-.5)
	must(t, err)
	b := builder(t, 2)
	must(t, b.AddModel(penaltyModel))
	must(t, b.AddModel(scaled))
	combined := build(t, b)
	assignments(2, func(bits []bool) {
		a, err := base.Evaluate(bits)
		must(t, err)
		z, err := penaltyModel.Evaluate(bits)
		must(t, err)
		got, err := combined.Evaluate(bits)
		if err != nil || got != -.5*a+z {
			t.Fatalf("bits %v: %g, %v", bits, got, err)
		}
	})
	zero, err := combined.Scale(0)
	must(t, err)
	if zero.Offset() != 0 || len(zero.LinearTerms()) != 0 || len(zero.QuadraticTerms()) != 0 {
		t.Fatal("scaling by zero left coefficients")
	}
	if _, err := combined.Scale(math.Inf(1)); err == nil {
		t.Fatal("accepted infinite scale")
	}
	if _, err := combined.Scale(math.MaxFloat64); err == nil {
		t.Fatal("accepted overflowing scale")
	}
}

func TestPenaltyFailuresAreAtomic(t *testing.T) {
	b := builder(t, 2)
	must(t, b.SetOffset(1))
	must(t, b.AddLinear(1, math.MaxFloat64))
	before, err := json.Marshal(build(t, b))
	must(t, err)
	x := builder(t, 2)
	must(t, x.SetOffset(1))
	must(t, x.AddLinear(1, math.MaxFloat64))
	for _, err := range []error{
		b.AddExactlyOne(nil, 1), b.AddExactlyOne([]int{0, 0}, 1), b.AddExactlyOne([]int{2}, 1),
		b.AddSquaredPenalty(nil, 1, -1), b.AddSquaredPenalty(nil, math.Inf(1), 1),
		b.AddSquaredPenalty(nil, 0, math.NaN()), b.AddSquaredPenalty(nil, math.MaxFloat64, 1),
		b.AddSquaredPenalty([]qubo.Weight{{Index: 0, Coefficient: math.NaN()}}, 0, 0),
		b.AddSquaredPenalty([]qubo.Weight{{Index: 1, Coefficient: 1e154}}, 0, 1),
		b.AddModel(build(t, x)), b.AddModel(build(t, builder(t, 3))),
	} {
		if err == nil {
			t.Fatal("accepted invalid operation")
		}
	}
	after, err := json.Marshal(build(t, b))
	must(t, err)
	if string(after) != string(before) {
		t.Fatalf("failure mutated builder: %s", after)
	}
}

func TestPenaltyAvoidsIntermediateOverflow(t *testing.T) {
	b := builder(t, 1)
	must(t, b.AddSquaredPenalty([]qubo.Weight{{Index: 0, Coefficient: 1e200}}, 1e200, 1e-200))
	m := build(t, b)
	for _, tc := range []struct {
		bit  bool
		want float64
	}{{false, 1e200}, {true, 0}} {
		got, err := m.Evaluate([]bool{tc.bit})
		if err != nil || !near(got, tc.want) {
			t.Fatalf("bit %v: %g, %v", tc.bit, got, err)
		}
	}
}
