// Command tsp runs the four-city reference example through the public APIs.
package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/quvly/Qasmith/pkg/optimization/bruteforce"
	"github.com/quvly/Qasmith/pkg/optimization/qaoa"
	"github.com/quvly/Qasmith/pkg/optimization/tsp"
)

//go:embed problem.json
var input []byte

func main() {
	out := flag.String("out", "examples/optimization/tsp/generated", "directory for generated files")
	flag.Parse()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := run(ctx, os.Stdout, *out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, out io.Writer, dir string) error {
	var problem tsp.Problem
	if err := json.Unmarshal(input, &problem); err != nil {
		return err
	}
	app, err := tsp.Prepare(problem, tsp.EncodeOptions{})
	if err != nil {
		return err
	}
	best, err := tsp.SolveClassical(ctx, problem, 0)
	if err != nil {
		return err
	}
	binary, err := (bruteforce.Solver{}).Solve(ctx, app.QUBO())
	if err != nil {
		return err
	}
	decoded, err := problem.Decode(binary.Assignment)
	if err != nil {
		return err
	}
	if !decoded.Feasible || decoded.Distance != best.Distance || binary.Energy != best.Distance {
		return fmt.Errorf("classical and QUBO minima disagree")
	}
	params := qaoa.Parameters{Gammas: []float64{.4}, Betas: []float64{.3}}
	ref := tsp.ReferenceOptions{Enabled: true}
	result, err := app.Run(ctx, params, tsp.RunOptions{Shots: 1000, Seed: 42, Exact: true, Reference: ref})
	if err != nil {
		return err
	}
	search, optimized, err := app.Optimize(ctx, qaoa.OptimizeOptions{Layers: 1, Seed: 42, Shots: 1000, MaxEvaluations: 200, Restarts: 3}, ref)
	if err != nil {
		return err
	}
	qJSON, err := json.MarshalIndent(app.QUBO(), "", "  ")
	if err != nil {
		return err
	}
	initial, err := app.QAOA().Export(params, qaoa.BuildOptions{Measure: true}, nil)
	if err != nil {
		return err
	}
	final, err := app.QAOA().Export(search.BestParameters, qaoa.BuildOptions{Measure: true}, nil)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	for _, file := range []struct {
		name string
		data []byte
	}{{"tsp.qubo.json", append(qJSON, '\n')}, {"initial.qasm", initial}, {"optimized.qasm", final}} {
		if err := os.WriteFile(filepath.Join(dir, file.name), file.data, 0644); err != nil {
			return err
		}
	}
	fmt.Fprintf(out, "Cities: %d, fixed start: %d, variables: %d, penalty: %g\n", problem.Cities(), problem.Start(), app.Encoding().Variables(), app.Penalty())
	fmt.Fprintf(out, "Uniform feasibility: %.6f%%\n", 100*app.Encoding().UniformFeasibility())
	for i := range app.Encoding().Variables() {
		city, pos, err := app.Encoding().Variable(i)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "Variable %d: city %d, position %d\n", i, city, pos)
	}
	fmt.Fprintf(out, "Classical optimal route: %v, distance: %g, tied tours: %d\n", best.Route, best.Distance, best.OptimalTours)
	fmt.Fprintf(out, "Exhaustive QUBO minimum: %g, decoded route: %v\n", binary.Energy, decoded.Route)
	fmt.Fprintf(out, "Initial exact expected QUBO energy: %.8f\n", *result.ExactExpectedEnergy)
	printSample(out, "Initial", result)
	fmt.Fprintf(out, "Optimized expected QUBO energy: %.8f, evaluations: %d, termination: %s\n",
		search.BestExpectedEnergy, search.Evaluations, search.TerminationReason)
	printSample(out, "Optimized", optimized)
	return nil
}

func printSample(out io.Writer, label string, result tsp.Result) {
	fmt.Fprintf(out, "%s sampled expected QUBO energy: %.8f\n", label, result.ExpectedQUBOEnergy)
	fmt.Fprintf(out, "%s feasible samples: %d/%d (%.2f%%), optimal samples: %d (%.2f%%)\n",
		label, result.FeasibleSamples, result.Shots, 100*result.FeasibilityRate, result.OptimalTourSamples, 100*(*result.OptimalSampleRate))
	if result.BestFeasible == nil {
		fmt.Fprintf(out, "%s: no feasible tour observed\n", label)
		return
	}
	fmt.Fprintf(out, "%s best feasible route: %v, distance: %g, bitstring: %s\n",
		label, result.BestFeasible.Decoded.Route, result.BestFeasible.Decoded.Distance, result.BestFeasible.Bitstring)
}
