package tsp_test

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/quvly/Qasmith/pkg/ir"
	"github.com/quvly/Qasmith/pkg/optimization/qaoa"
	"github.com/quvly/Qasmith/pkg/optimization/tsp"
)

func TestTSPQAOAPipeline(t *testing.T) {
	ctx := context.Background()
	p := symmetric(t)
	app, err := tsp.Prepare(p, tsp.EncodeOptions{})
	must(t, err)
	params := qaoa.Parameters{Gammas: []float64{.4, .2}, Betas: []float64{.3, .6}}
	circuit, err := app.QAOA().Build(params, qaoa.BuildOptions{Measure: true})
	must(t, err)
	layout, err := circuit.Validate()
	must(t, err)
	if layout != (ir.Layout{Qubits: 9, Bits: 9}) {
		t.Fatalf("layout %+v", layout)
	}
	for i, op := range circuit.Operations[len(circuit.Operations)-9:] {
		m, ok := op.(ir.Measure)
		if !ok || m.Qubit != ir.QubitID(i) || m.Bit != ir.BitID(i) {
			t.Fatalf("measurement %v", op)
		}
	}
	code, err := app.QAOA().Export(params, qaoa.BuildOptions{Measure: true}, nil)
	must(t, err)
	if !strings.Contains(string(code), "qubit[9] q;") || !strings.Contains(string(code), "c[8] = measure q[8];") {
		t.Fatalf("bad QASM: %s", code)
	}
	opts := tsp.RunOptions{Shots: 1000, Seed: 42, Exact: true, Reference: tsp.ReferenceOptions{Enabled: true}}
	result, err := app.Run(ctx, params, opts)
	must(t, err)
	again, err := app.Run(ctx, params, opts)
	must(t, err)
	if !reflect.DeepEqual(result, again) || result.QAOALayers != 2 || result.FeasibleSamples+result.InfeasibleSamples != 1000 ||
		result.InfeasibleSamples == 0 || result.Classical.Distance != 80 || result.ExactExpectedEnergy == nil ||
		!close(result.ExpectedQUBOEnergy, result.Raw.ExpectedEnergy) {
		t.Fatalf("unexpected results: %+v", result)
	}
	for _, sample := range append(result.Feasible[:len(result.Feasible):len(result.Feasible)], result.Infeasible...) {
		if !close(sample.QUBOEnergy, sample.TravelObjective+sample.ConstraintPenalty) {
			t.Fatalf("components %+v", sample)
		}
	}
	uniform := qaoa.Parameters{Gammas: []float64{0}, Betas: []float64{0}}
	baseline, err := app.Run(ctx, uniform, opts)
	must(t, err)
	if baseline.FeasibleSamples == 0 || baseline.InfeasibleSamples == 0 || baseline.BestFeasible == nil || baseline.BestFeasible.Decoded.Route == nil {
		t.Fatalf("expected both classes from fixed-seed uniform circuit: %+v", baseline)
	}
	var mean float64
	for mask := range 512 {
		bits := make([]bool, 9)
		for i := range bits {
			bits[i] = mask&(1<<i) != 0
		}
		v, err := app.QUBO().Evaluate(bits)
		must(t, err)
		mean += v / 512
	}
	if !close(*baseline.ExactExpectedEnergy, mean) {
		t.Fatalf("uniform expected energy %g, want %g", *baseline.ExactExpectedEnergy, mean)
	}
	search, optimized, err := app.Optimize(ctx, qaoa.OptimizeOptions{Layers: 1, MaxEvaluations: 16, Seed: 42, Shots: 128}, opts.Reference)
	must(t, err)
	if search.Evaluations > 16 || search.BestExpectedEnergy > search.InitialExpectedEnergy ||
		optimized.Classical.Distance != 80 || optimized.Shots != 128 {
		t.Fatalf("optimizer result %+v", search)
	}
}

func TestTSPExportAboveSimulatorLimit(t *testing.T) {
	distances := make([][]float64, 6)
	for i := range distances {
		distances[i] = make([]float64, 6)
	}
	p, err := tsp.New(distances, 4)
	must(t, err)
	app, err := tsp.Prepare(p, tsp.EncodeOptions{})
	must(t, err)
	params := qaoa.Parameters{Gammas: []float64{.4}, Betas: []float64{.3}}
	code, err := app.QAOA().Export(params, qaoa.BuildOptions{Measure: true}, nil)
	must(t, err)
	if !strings.Contains(string(code), "qubit[25] q;") {
		t.Fatal("wrong circuit size")
	}
	if _, err := app.Run(context.Background(), params, tsp.RunOptions{Shots: 1}); err == nil || !strings.Contains(err.Error(), "simulator limit") {
		t.Fatalf("wrong simulation error %v", err)
	}
}
