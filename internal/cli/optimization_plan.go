package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/quvly/Qasmith/pkg/optimization/plan"
	"github.com/quvly/Qasmith/pkg/optimization/qaoa"
	"github.com/quvly/Qasmith/pkg/qasmith"
)

func optimizeSmith(ctx context.Context, compiler qasmith.Compiler, source, output string, options qaoa.OptimizeOptions, sample bool, out io.Writer) error {
	prepared, err := compiler.Prepare(source)
	if err != nil {
		return err
	}
	result, err := prepared.Optimize(ctx, options, sample)
	if err != nil && result.Search.Evaluations == 0 {
		return err
	}
	if err == nil && output != "" {
		code, err := result.Prepared.Export()
		if err != nil {
			return err
		}
		if err := writeOutput(output, code); err != nil {
			return err
		}
	}
	search := result.Search
	fmt.Fprintf(out, "Objective mode: %s\nInitial expected energy: %.12g\nBest expected energy: %.12g\nGammas: %v\nBetas: %v\nEvaluations: %d\nIterations: %d\nConverged: %t\nTermination: %s\n",
		search.Mode, search.InitialExpectedEnergy, search.BestExpectedEnergy, search.BestParameters.Gammas, search.BestParameters.Betas,
		search.Evaluations, search.Iterations, search.Converged, search.TerminationReason)
	if result.Samples != nil && result.Samples.Optimization != nil && result.Samples.Raw.Shots > 0 {
		printPlanned(out, *result.Samples.Optimization)
	}
	return err
}

func printPlanned(out io.Writer, result plan.Result) {
	fmt.Fprintf(out, "Qubits: %d\nLayers: %d\nShots: %d\nEstimated expected QUBO energy: %.12g\nMeasurement counts:\n",
		result.Raw.QubitCount, result.Raw.LayerCount, result.Raw.Shots, result.ExpectedQUBOEnergy)
	for _, v := range result.Observations {
		fmt.Fprintf(out, "%s: %d (%.6f), QUBO energy %.12g\n", v.Bitstring, v.Count, v.Probability, v.QUBOEnergy)
	}
	if best := result.BestObserved; best != nil {
		fmt.Fprintf(out, "Best observed assignment: %s\nBest observed QUBO energy: %.12g\n", best.Bitstring, best.QUBOEnergy)
	}
	if result.Domain != nil {
		for _, metric := range result.Domain.Metrics() {
			fmt.Fprintf(out, "%s: %s\n", metric.Label, metric.Value)
		}
	}
}
