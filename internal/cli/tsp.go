package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/quvly/Qasmith/pkg/optimization/qaoa"
	"github.com/quvly/Qasmith/pkg/optimization/tsp"
)

const tspUsage = "usage: qasmith tsp <inspect|qubo|compile|run|optimize|solve-classical> problem.json [options]"

type tspOptions struct {
	encode    tsp.EncodeOptions
	search    qaoa.OptimizeOptions
	reference tsp.ReferenceOptions
	output    string
	exact     bool
}

func runTSP(ctx context.Context, args []string, out, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, tspUsage)
		return 2
	}
	command := args[0]
	args = args[1:]
	switch command {
	case "inspect", "qubo", "compile", "run", "optimize", "solve-classical":
	case "help", "-h", "--help":
		fmt.Fprintln(out, tspUsage)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown TSP command %q\n%s\n", command, tspUsage)
		return 2
	}

	flags := flag.NewFlagSet("tsp "+command, flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() {
		fmt.Fprintf(stderr, "usage: qasmith tsp %s problem.json [options]\n", command)
		flags.PrintDefaults()
	}
	var options tspOptions
	var penalty float64
	var gamma, beta string
	if command != "inspect" && command != "solve-classical" {
		flags.Float64Var(&penalty, "penalty", 0, "positive one-hot penalty override")
		flags.IntVar(&options.encode.MaxVariables, "max-variables", 0, "QUBO encoding variable limit (0 uses default)")
	}
	if command == "qubo" || command == "compile" || command == "optimize" {
		flags.StringVar(&options.output, "o", "", "output path")
	}
	if command == "compile" || command == "run" || command == "optimize" {
		flags.IntVar(&options.search.Layers, "layers", 1, "number of QAOA layers")
		flags.StringVar(&gamma, "gammas", "", "comma-separated cost parameters")
		flags.StringVar(&beta, "betas", "", "comma-separated mixer parameters")
		flags.IntVar(&options.search.MaxQubits, "max-qubits", 0, "allocated qubit limit (0 uses default)")
		flags.IntVar(&options.search.MaxGates, "max-gates", 0, "generated gate limit (0 uses default)")
	}
	if command == "run" || command == "optimize" {
		flags.IntVar(&options.search.Shots, "shots", 1000, "number of shots")
		flags.Int64Var(&options.search.Seed, "seed", 0, "deterministic seed")
		flags.BoolVar(&options.reference.Enabled, "reference", true, "run independent bounded tour search")
	}
	if command == "run" || command == "optimize" || command == "solve-classical" {
		flags.IntVar(&options.reference.MaxCities, "max-cities", 0, "classical permutation limit (0 uses default)")
	}
	if command == "run" {
		flags.BoolVar(&options.exact, "exact", false, "also compute the exact expectation")
	}
	if command == "optimize" {
		flags.IntVar(&options.search.MaxEvaluations, "max-evals", 200, "maximum search evaluations")
		flags.IntVar(&options.search.MaxIterations, "max-iters", 100, "maximum coordinate sweeps")
		flags.Float64Var(&options.search.Tolerance, "tolerance", 1e-4, "minimum coordinate step")
		flags.IntVar(&options.search.Restarts, "restarts", 0, "additional seeded starting points")
		flags.StringVar((*string)(&options.search.Mode), "eval-mode", "exact", "objective mode: exact or sampled")
	}
	var source string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		source, args = args[0], args[1:]
	}
	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if source == "" && flags.NArg() == 1 {
		source = flags.Arg(0)
	} else if flags.NArg() != 0 {
		source = ""
	}
	if source == "" {
		flags.Usage()
		return 2
	}
	flags.Visit(func(f *flag.Flag) {
		if f.Name == "penalty" {
			options.encode.Penalty = &penalty
		}
	})
	if command == "compile" || command == "run" || command == "optimize" {
		if options.search.Layers < 1 || command != "compile" && options.search.Shots < 1 {
			fmt.Fprintln(stderr, "layers and shots must be positive where applicable")
			return 2
		}
		if command != "optimize" || gamma != "" || beta != "" {
			params, err := parseParameters(options.search.Layers, gamma, beta)
			if err != nil {
				fmt.Fprintln(stderr, err)
				return 2
			}
			options.search.InitialParameters = params
		}
	}
	stem := strings.TrimSuffix(source, filepath.Ext(source))
	if options.output == "" {
		switch command {
		case "qubo":
			options.output = stem + ".qubo.json"
		case "compile":
			options.output = stem + ".qasm"
		}
	}
	if (command == "compile" || command == "optimize") && options.output != "" && filepath.Ext(options.output) != ".qasm" {
		fmt.Fprintln(stderr, "output must have a .qasm extension")
		return 2
	}
	if err := executeTSP(ctx, command, source, options, out); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func executeTSP(ctx context.Context, command, source string, opts tspOptions, out io.Writer) error {
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	var problem tsp.Problem
	if err := json.Unmarshal(data, &problem); err != nil {
		return fmt.Errorf("%s: %w", source, err)
	}
	switch command {
	case "inspect":
		encoding, err := problem.Encoding()
		if err != nil {
			return err
		}
		penalty, err := problem.DefaultPenalty()
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(out, "Cities: %d\nStart: %d\nSymmetric: %t\nMaximum distance: %g\nQUBO variables: %d\nDefault penalty: %g\nWithin local qubit limit (%d): %t\nUniform feasibility: %.6f%%\n",
			problem.Cities(), problem.Start(), problem.Symmetric(), problem.MaxDistance(), encoding.Variables(), penalty, qaoa.LocalMaxQubits,
			encoding.Variables() <= qaoa.LocalMaxQubits, 100*encoding.UniformFeasibility())
		return err
	case "solve-classical":
		best, err := tsp.SolveClassical(ctx, problem, opts.reference.MaxCities)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(out, "Classical optimal route: %v\nClassical optimal distance: %g\nOptimal tours: %d\nTours checked: %d\n",
			best.Route, best.Distance, best.OptimalTours, best.ToursChecked)
		return err
	case "qubo":
		model, _, err := tsp.ToQUBO(problem, opts.encode)
		if err != nil {
			return err
		}
		formatted, err := json.MarshalIndent(model, "", "  ")
		if err != nil {
			return err
		}
		return writeOutput(opts.output, append(formatted, '\n'))
	}
	app, err := tsp.Prepare(problem, opts.encode)
	if err != nil {
		return err
	}
	save := func(params qaoa.Parameters) error {
		code, err := app.QAOA().Export(params, qaoa.BuildOptions{Measure: true, MaxQubits: opts.search.MaxQubits, MaxGates: opts.search.MaxGates}, nil)
		if err != nil {
			return err
		}
		return writeOutput(opts.output, code)
	}
	switch command {
	case "compile":
		return save(opts.search.InitialParameters)
	case "run":
		result, err := app.Run(ctx, opts.search.InitialParameters, tsp.RunOptions{Shots: opts.search.Shots, Seed: opts.search.Seed,
			MaxQubits: opts.search.MaxQubits, MaxGates: opts.search.MaxGates, Exact: opts.exact, Reference: opts.reference})
		if err != nil {
			return err
		}
		printTSP(out, result)
	case "optimize":
		search, result, err := app.Optimize(ctx, opts.search, opts.reference)
		if err != nil {
			return err
		}
		if opts.output != "" {
			if err := save(search.BestParameters); err != nil {
				return err
			}
		}
		fmt.Fprintf(out, "Objective mode: %s\nInitial expected QUBO energy: %.12g\nOptimized expected QUBO energy: %.12g\nGammas: %v\nBetas: %v\nEvaluations: %d\nTermination: %s\n",
			search.Mode, search.InitialExpectedEnergy, search.BestExpectedEnergy, search.BestParameters.Gammas, search.BestParameters.Betas, search.Evaluations, search.TerminationReason)
		printTSP(out, result)
	}
	return nil
}

func printTSP(out io.Writer, result tsp.Result) {
	fmt.Fprintf(out, "Cities: %d\nQubits: %d\nLayers: %d\nShots: %d\nEstimated expected QUBO energy: %.12g\nFeasible samples: %d\nInfeasible samples: %d\nFeasibility rate: %.6f\n",
		result.CityCount, result.QubitCount, result.QAOALayers, result.Shots, result.ExpectedQUBOEnergy, result.FeasibleSamples, result.InfeasibleSamples, result.FeasibilityRate)
	if result.ExactExpectedEnergy != nil {
		fmt.Fprintf(out, "Exact expected QUBO energy: %.12g\n", *result.ExactExpectedEnergy)
	}
	if result.BestObserved != nil {
		fmt.Fprintf(out, "Best observed QUBO assignment: %s (feasible: %t)\nBest observed QUBO energy: %.12g\n",
			result.BestObserved.Bitstring, result.BestObserved.Decoded.Feasible, result.BestObserved.QUBOEnergy)
	}
	if result.BestFeasible == nil {
		fmt.Fprintln(out, "No feasible tour was observed")
	} else {
		fmt.Fprintf(out, "Best feasible route: %v\nBest feasible distance: %g\n", result.BestFeasible.Decoded.Route, result.BestFeasible.Decoded.Distance)
	}
	for _, v := range result.Feasible {
		fmt.Fprintf(out, "Route %v %s: %d (%.6f), distance %g\n", v.Decoded.Route, v.Bitstring, v.Count, v.Frequency, v.Decoded.Distance)
	}
	if result.Classical == nil {
		fmt.Fprintf(out, "Classical reference not calculated (%s)\n", result.ReferenceStatus)
	} else {
		fmt.Fprintf(out, "Classical optimal route: %v\nClassical optimal distance: %g\nOptimal tour samples: %d\nOptimal tour sample rate: %.6f\n",
			result.Classical.Route, result.Classical.Distance, result.OptimalTourSamples, *result.OptimalSampleRate)
	}
	if len(result.Infeasible) > 0 {
		fmt.Fprintln(out, "Infeasible measurement counts:")
		for _, v := range result.Infeasible {
			fmt.Fprintf(out, "%s: %d\n", v.Bitstring, v.Count)
		}
	}
}
