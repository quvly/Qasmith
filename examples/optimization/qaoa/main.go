// Command qaoa demonstrates fixed-seed hybrid optimization of an embedded QUBO.
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

	"github.com/quvly/Qasmith/pkg/optimization/qaoa"
	"github.com/quvly/Qasmith/pkg/optimization/qubo"
)

//go:embed problem.json
var input []byte

func main() {
	output := flag.String("o", "qaoa.qasm", "optimized OpenQASM output path")
	initial := flag.String("initial-output", "qaoa-initial.qasm", "initial OpenQASM output path")
	flag.Parse()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := run(ctx, os.Stdout, *initial, *output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, out io.Writer, initial, output string) error {
	var model qubo.Model
	if err := json.Unmarshal(input, &model); err != nil {
		return err
	}
	problem, err := qaoa.FromQUBO(model)
	if err != nil {
		return err
	}
	start := qaoa.Parameters{Gammas: []float64{.4}, Betas: []float64{.3}}
	initialCode, err := problem.Export(start, qaoa.BuildOptions{Measure: true}, nil)
	if err != nil {
		return err
	}
	exact, err := qaoa.EvaluateExact(ctx, problem, start)
	if err != nil {
		return err
	}
	sampled, err := qaoa.EvaluateSampled(ctx, problem, start, 1000, 42)
	if err != nil {
		return err
	}
	optimized, err := qaoa.Optimize(ctx, problem, qaoa.OptimizeOptions{InitialParameters: start, Seed: 42, Shots: 1000, Restarts: 2, Reference: true})
	if err != nil {
		return err
	}
	code, err := problem.Export(optimized.BestParameters, qaoa.BuildOptions{Measure: true}, nil)
	if err != nil {
		return err
	}
	if initial == output {
		return fmt.Errorf("initial and optimized output paths must differ")
	}
	if err := os.WriteFile(initial, initialCode, 0644); err != nil {
		return err
	}
	if err := os.WriteFile(output, code, 0644); err != nil {
		return err
	}
	best := optimized.Final.BestObserved
	assignment, err := qaoa.DecodeBitstring(best.Bitstring)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(out,
		"Initial exact energy: %.8f\nInitial sampled energy: %.8f\nOptimized expected energy: %.8f\nBest observed: %s (%v), energy %.8f, frequency %.4f\nClassical minimum: %.8f\nOptimal sample fraction: %.4f\nInitial OpenQASM: %s\nOptimized OpenQASM: %s\n",
		exact.ExpectedEnergy, sampled.ExpectedEnergy, optimized.BestExpectedEnergy,
		best.Bitstring, assignment, best.Energy, best.Frequency, optimized.Classical.MinimumEnergy, optimized.Classical.OptimalSampleFraction, initial, output)
	return err
}
