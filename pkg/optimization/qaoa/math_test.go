package qaoa_test

import (
	"context"
	"errors"
	"math"
	"math/cmplx"
	"math/rand"
	"reflect"
	"strings"
	"testing"

	"github.com/quvly/Qasmith/pkg/ir"
	"github.com/quvly/Qasmith/pkg/optimization/ising"
	"github.com/quvly/Qasmith/pkg/optimization/qaoa"
	"github.com/quvly/Qasmith/pkg/optimization/qubo"
	"github.com/quvly/Qasmith/pkg/qasmith"
)

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func params(g, b float64) qaoa.Parameters {
	return qaoa.Parameters{Gammas: []float64{g}, Betas: []float64{b}}
}

func model(t *testing.T, n int, offset float64, fields []ising.Field, couplings []ising.Coupling) qaoa.Problem {
	t.Helper()
	h, err := ising.New(n, offset, fields, couplings)
	must(t, err)
	p, err := qaoa.New(h)
	must(t, err)
	return p
}

func fromMatrix(t *testing.T, matrix [][]float64, offset float64) qaoa.Problem {
	t.Helper()
	q, err := qubo.FromMatrix(matrix, offset)
	must(t, err)
	p, err := qaoa.FromQUBO(q)
	must(t, err)
	return p
}

func diagonal(n int, offset float64, fields []ising.Field, couplings []ising.Coupling) []float64 {
	values := make([]float64, 1<<n)
	for basis := range values {
		spin := func(i int) float64 {
			if basis&(1<<i) != 0 {
				return -1
			}
			return 1
		}
		value := offset
		for _, h := range fields {
			value += h.Coefficient * spin(h.Index)
		}
		for _, j := range couplings {
			value += j.Coefficient * spin(j.I) * spin(j.J)
		}
		values[basis] = value
	}
	return values
}

func compareState(t *testing.T, circuit *ir.Program, want []complex128) []complex128 {
	t.Helper()
	state, measured, err := qasmith.Statevector(context.Background(), circuit)
	must(t, err)
	if len(state) != len(want) || len(measured) != 0 {
		t.Fatalf("invalid state layout: %v / %v", state, measured)
	}
	var norm float64
	for i, amp := range state {
		if !(cmplx.Abs(amp-want[i]) <= 2e-12) {
			t.Fatalf("state[%d] = %v, want %v", i, amp, want[i])
		}
		norm += cmplx.Abs(amp) * cmplx.Abs(amp)
	}
	if math.Abs(norm-1) > 2e-12 {
		t.Fatalf("invalid norm %g", norm)
	}
	return state
}

func TestIndependentCostPhases(t *testing.T) {
	for _, tc := range []struct {
		name   string
		n      int
		fields []ising.Field
		pairs  []ising.Coupling
	}{
		{"linear", 1, []ising.Field{{Index: 0, Coefficient: -.72}}, nil},
		{"coupling", 2, nil, []ising.Coupling{{I: 0, J: 1, Coefficient: .31}}},
		{"mixed", 3, []ising.Field{{Index: 0, Coefficient: .12}, {Index: 2, Coefficient: 0}},
			[]ising.Coupling{{I: 0, J: 1, Coefficient: 0}, {I: 1, J: 2, Coefficient: -.83}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := model(t, tc.n, 9.125, tc.fields, tc.pairs)
			circuit, err := p.Build(params(.77, 0), qaoa.BuildOptions{})
			must(t, err)
			diag := diagonal(tc.n, 0, tc.fields, tc.pairs)
			want := make([]complex128, len(diag))
			for i, e := range diag {
				want[i] = cmplx.Exp(complex(0, -.77*e)) / complex(math.Sqrt(float64(len(diag))), 0)
			}
			compareState(t, circuit, want)
			if tc.name == "mixed" && len(circuit.Operations) != 13 {
				t.Fatalf("zero coefficients emitted gates: %v", circuit.Operations)
			}
		})
	}
}

// tensorMixer multiplies the full tensor-product matrix without applying IR gates.
func tensorMixer(state []complex128, n int, beta float64) []complex128 {
	result := make([]complex128, len(state))
	for row := range result {
		for col, amp := range state {
			entry := complex(1, 0)
			for bit := range n {
				if (row^col)&(1<<bit) == 0 {
					entry *= complex(math.Cos(beta), 0)
				} else {
					entry *= complex(0, -math.Sin(beta))
				}
			}
			result[row] += entry * amp
		}
	}
	return result
}

func TestIndependentMixer(t *testing.T) {
	for _, n := range []int{1, 3} {
		p := model(t, n, 0, nil, nil)
		circuit, err := p.Build(params(0, -.61), qaoa.BuildOptions{})
		must(t, err)
		for basis := 0; basis < 1<<n; basis++ {
			ops := append([]ir.Operation{}, circuit.Operations[:n]...)
			for bit := range n {
				if basis&(1<<bit) != 0 {
					ops = append(ops, ir.ApplyGate{Gate: ir.X, Qubits: []ir.QubitID{ir.QubitID(bit)}})
				}
			}
			ops = append(ops, circuit.Operations[2*n:]...)
			state := make([]complex128, 1<<n)
			state[basis] = 1
			compareState(t, &ir.Program{Operations: ops}, tensorMixer(state, n, -.61))
		}
	}
}

func TestIndependentRandomizedCircuits(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	ctx := context.Background()
	for trial := range 24 {
		n := 1 + trial%4
		p := qaoa.Parameters{Gammas: []float64{.24}, Betas: []float64{-.14}}
		if (trial/4)%2 == 1 {
			p.Gammas = append(p.Gammas, -.83)
			p.Betas = append(p.Betas, .19)
		}
		fields := make([]ising.Field, 0, n)
		var pairs []ising.Coupling
		for i := range n {
			fields = append(fields, ising.Field{Index: i, Coefficient: float64(rng.Intn(11)-5) / 7})
			for j := i + 1; j < n; j++ {
				pairs = append(pairs, ising.Coupling{I: i, J: j, Coefficient: float64(rng.Intn(11)-5) / 9})
			}
		}
		offset := 1.25 - .17*float64(trial)
		problem := model(t, n, offset, fields, pairs)
		circuit, err := problem.Build(p, qaoa.BuildOptions{})
		must(t, err)
		_, err = circuit.Validate()
		must(t, err)
		diag := diagonal(n, 0, fields, pairs)
		want := make([]complex128, len(diag))
		for i := range want {
			want[i] = 1 / complex(math.Sqrt(float64(len(want))), 0)
		}
		for i, gamma := range p.Gammas {
			for j, e := range diag {
				want[j] *= cmplx.Exp(complex(0, -gamma*e))
			}
			want = tensorMixer(want, n, p.Betas[i])
		}
		compareState(t, circuit, want)
		expected, err := qaoa.EvaluateExact(ctx, problem, p)
		must(t, err)
		var independent float64
		for i, amp := range want {
			independent += cmplx.Abs(amp) * cmplx.Abs(amp) * (diag[i] + offset)
		}
		if math.Abs(expected.ExpectedEnergy-independent) > 2e-12 || expected.LayerCount != len(p.Gammas) || expected.Mode != qaoa.Exact {
			t.Fatalf("trial %d energy %g, want %g", trial, expected.ExpectedEnergy, independent)
		}
		data, err := problem.Export(p, qaoa.BuildOptions{Measure: true}, nil)
		must(t, err)
		if !strings.Contains(string(data), "OPENQASM 3.0;") || strings.Count(string(data), " = measure ") != n {
			t.Fatalf("bad QASM %s", data)
		}
	}
}

func TestOneQubitGoldenAndBitOrder(t *testing.T) {
	ctx := context.Background()
	p := fromMatrix(t, [][]float64{{1}}, 0)
	a := params(math.Pi/2, -math.Pi/4)
	exact, err := qaoa.EvaluateExact(ctx, p, a)
	must(t, err)
	if math.Abs(exact.ExpectedEnergy) > 1e-14 {
		t.Fatalf("expected ground energy, got %g", exact.ExpectedEnergy)
	}
	for _, tc := range []struct {
		problem qaoa.Problem
		want    string
		n       int
	}{
		{p, "0", 1},
		{fromMatrix(t, [][]float64{{-1, 0}, {0, 1}}, 5), "10", 2},
	} {
		sampled, err := qaoa.EvaluateSampled(ctx, tc.problem, a, 100, 42)
		must(t, err)
		if !reflect.DeepEqual(sampled.Counts, map[string]int{tc.want: 100}) || sampled.BestObserved.Bitstring != tc.want || len(sampled.BestObserved.Assignment) != tc.n {
			t.Fatalf("counts or assignment: %+v", sampled)
		}
		if tc.n == 2 && (!sampled.BestObserved.Assignment[0] || sampled.BestObserved.Assignment[1] || sampled.ExpectedEnergy != 4) {
			t.Fatalf("wrong decoded second fixture: %+v", sampled)
		}
	}
}

func TestQUBOExpectationAndSamples(t *testing.T) {
	ctx := context.Background()
	matrix := [][]float64{{-.5, .8, 0}, {-.1, 1.2, 0}, {0, .3, -.75}}
	p := fromMatrix(t, matrix, 2.75)
	a := qaoa.Parameters{Gammas: []float64{.37, -.15}, Betas: []float64{-.26, .42}}
	circuit, err := p.Build(a, qaoa.BuildOptions{})
	must(t, err)
	state, _, err := qasmith.Statevector(ctx, circuit)
	must(t, err)
	var want float64
	for index, amp := range state {
		cost := 2.75
		for i, row := range matrix {
			for j, c := range row {
				if index&(1<<i) != 0 && index&(1<<j) != 0 {
					cost += c
				}
			}
		}
		want += cost * cmplx.Abs(amp) * cmplx.Abs(amp)
	}
	got, err := qaoa.EvaluateExact(ctx, p, a)
	must(t, err)
	if math.Abs(got.ExpectedEnergy-want) > 2e-12 {
		t.Fatalf("expected %g, got %g", want, got.ExpectedEnergy)
	}
	h, err := qaoa.New(p.Ising())
	must(t, err)
	spin, err := qaoa.EvaluateExact(ctx, h, a)
	must(t, err)
	if spin.ExpectedEnergy != got.ExpectedEnergy {
		t.Fatal("inconsistent problem constructors")
	}
	shots, err := qaoa.EvaluateSampled(ctx, p, a, 20_000, 51)
	must(t, err)
	again, err := qaoa.EvaluateSampled(ctx, p, a, 20_000, 51)
	must(t, err)
	if !reflect.DeepEqual(shots, again) || math.Abs(shots.ExpectedEnergy-want) > .03 || shots.Mode != qaoa.Sampled {
		t.Fatalf("samples %+v, want about %g", shots, want)
	}
}

func TestCircuitAndEvaluationLimits(t *testing.T) {
	ctx := context.Background()
	p := model(t, 2, 0, []ising.Field{{Index: 0, Coefficient: .25}}, nil)
	for _, tc := range []struct {
		p       qaoa.Parameters
		options qaoa.BuildOptions
	}{
		{qaoa.Parameters{}, qaoa.BuildOptions{}},
		{qaoa.Parameters{Gammas: []float64{.2, .4}, Betas: []float64{0}}, qaoa.BuildOptions{}},
		{params(math.NaN(), 0), qaoa.BuildOptions{}},
		{params(.2, math.MaxFloat64), qaoa.BuildOptions{}},
		{params(.2, .1), qaoa.BuildOptions{MaxQubits: 1}},
		{params(.2, .1), qaoa.BuildOptions{MaxGates: 4}},
	} {
		if got, err := p.Build(tc.p, tc.options); got != nil || err == nil {
			t.Fatalf("invalid build %v, %v", got, err)
		}
	}
	// Doubling gamma before multiplying would overflow unnecessarily.
	circuit, err := p.Build(params(math.MaxFloat64, 0), qaoa.BuildOptions{})
	must(t, err)
	if got := circuit.Operations[4].(ir.ApplyGate).Parameters[0]; got != math.MaxFloat64/2 {
		t.Fatalf("angle %g", got)
	}
	overflow := model(t, 1, 0, []ising.Field{{Index: 0, Coefficient: math.MaxFloat64}}, nil)
	if _, err := overflow.Build(params(1, 0), qaoa.BuildOptions{}); err == nil {
		t.Fatal("accepted angle overflow")
	}
	large := model(t, 17, 0, nil, nil)
	if data, err := large.Export(params(0, 0), qaoa.BuildOptions{Measure: true}, nil); err != nil || !strings.Contains(string(data), "qubit[17] q;") {
		t.Fatalf("large export: %s, %v", data, err)
	}
	if _, err := qaoa.EvaluateExact(ctx, large, params(0, 0)); err == nil || !strings.Contains(err.Error(), "simulator limit") {
		t.Fatalf("large eval %v", err)
	}
	for _, shots := range []int{0, -1, qaoa.MaxShots + 1} {
		if _, err := qaoa.EvaluateSampled(ctx, p, params(.2, .1), shots, 0); err == nil {
			t.Fatalf("accepted %d shots", shots)
		}
	}
	if _, err := qaoa.DecodeBitstring("0x1"); err == nil {
		t.Fatal("accepted invalid bitstring")
	}
	done, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := qaoa.EvaluateExact(done, p, params(.2, .1)); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel %v", err)
	}
}

type countBackend struct{ used *bool }

func (countBackend) Name() string { return "test backend" }
func (b countBackend) Compile(p *ir.Program) ([]byte, error) {
	*b.used = true
	_, err := p.Validate()
	return []byte("backend output"), err
}

type reverseMeasurements struct{}

func (reverseMeasurements) Name() string { return "reverse measurement mapping" }
func (reverseMeasurements) Run(p *ir.Program) error {
	p.Operations[len(p.Operations)-2] = ir.Measure{Qubit: 1, Bit: 0}
	p.Operations[len(p.Operations)-1] = ir.Measure{Qubit: 0, Bit: 1}
	return nil
}

func TestBackendAndPassCompatibility(t *testing.T) {
	p := model(t, 2, 0, nil, nil)
	used := false
	opts := qaoa.BuildOptions{Measure: true, Passes: []ir.Pass{qasmith.CancelAdjacentX{}}}
	output, err := p.Export(params(.4, .3), opts, countBackend{used: &used})
	must(t, err)
	if !used || string(output) != "backend output" {
		t.Fatalf("output %q", output)
	}
	opts.Passes = []ir.Pass{reverseMeasurements{}}
	if _, err := p.Build(params(.4, .3), opts); err == nil || !strings.Contains(err.Error(), "variable order") {
		t.Fatalf("accepted changed layout: %v", err)
	}
}
