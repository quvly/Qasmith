package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/quvly/Qasmith/pkg/optimization/qaoa"
	"github.com/quvly/Qasmith/pkg/optimization/qubo"
)

const qaoaUsage = "usage: qasmith qaoa <compile|run|optimize> problem.json [options]"

func runQAOA(ctx context.Context, args []string, out, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, qaoaUsage)
		return 2
	}
	command := args[0]
	args = args[1:]
	switch command {
	case "compile", "run", "optimize":
	case "-h", "--help", "help":
		fmt.Fprintln(out, qaoaUsage)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown QAOA command %q\n%s\n", command, qaoaUsage)
		return 2
	}
	flags := flag.NewFlagSet("qaoa "+command, flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() {
		fmt.Fprintf(stderr, "usage: qasmith qaoa %s problem.json [options]\n", command)
		flags.PrintDefaults()
	}
	var gamma, beta, output string
	var options qaoa.OptimizeOptions
	flags.IntVar(&options.Layers, "layers", 1, "number of QAOA layers")
	flags.StringVar(&gamma, "gammas", "", "comma-separated cost parameters")
	flags.StringVar(&beta, "betas", "", "comma-separated mixer parameters")
	flags.IntVar(&options.MaxGates, "max-gates", 0, "maximum generated gates (0 uses default)")
	flags.IntVar(&options.MaxQubits, "max-qubits", 0, "maximum allocated qubits (0 uses default)")
	if command != "run" {
		flags.StringVar(&output, "o", "", "output .qasm file")
	}
	if command != "compile" {
		flags.IntVar(&options.Shots, "shots", 1000, "number of shots")
		flags.Int64Var(&options.Seed, "seed", 0, "deterministic seed")
	}
	if command == "optimize" {
		flags.IntVar(&options.MaxEvaluations, "max-evals", 200, "maximum search evaluations")
		flags.IntVar(&options.MaxIterations, "max-iters", 100, "maximum search coordinate sweeps")
		flags.Float64Var(&options.Tolerance, "tolerance", 1e-4, "minimum coordinate step")
		flags.IntVar(&options.Restarts, "restarts", 0, "additional seeded starting points")
		flags.StringVar((*string)(&options.Mode), "eval-mode", "exact", "objective mode: exact or sampled")
		flags.BoolVar(&options.Reference, "reference", true, "compare against bounded exhaustive search")
		flags.IntVar(&options.ReferenceLimit, "reference-limit", 0, "classical reference variable limit (0 uses default)")
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
	if source == "" || options.Layers < 1 || command != "compile" && options.Shots < 1 {
		fmt.Fprintln(stderr, "a problem file, positive layer count, and positive shot count are required where applicable")
		flags.Usage()
		return 2
	}
	if command != "optimize" || gamma != "" || beta != "" {
		params, err := parseParameters(options.Layers, gamma, beta)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
		options.InitialParameters = params
	}
	if command == "compile" && output == "" {
		output = strings.TrimSuffix(source, filepath.Ext(source)) + ".qasm"
	}
	if output != "" && filepath.Ext(output) != ".qasm" {
		fmt.Fprintln(stderr, "output must have a .qasm extension")
		return 2
	}
	if err := executeQAOA(ctx, command, source, output, options, out); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func parseParameters(layers int, gamma, beta string) (qaoa.Parameters, error) {
	parse := func(name, value string) ([]float64, error) {
		values := strings.Split(value, ",")
		if value == "" || len(values) != layers {
			return nil, fmt.Errorf("--%s expects %d comma-separated values", name, layers)
		}
		result := make([]float64, layers)
		for i, v := range values {
			n, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
			if err != nil {
				return nil, fmt.Errorf("--%s value %d: %w", name, i+1, err)
			}
			result[i] = n
		}
		return result, nil
	}
	g, err := parse("gammas", gamma)
	if err != nil {
		return qaoa.Parameters{}, err
	}
	b, err := parse("betas", beta)
	if err != nil {
		return qaoa.Parameters{}, err
	}
	p := qaoa.Parameters{Gammas: g, Betas: b}
	return p, p.Validate(layers)
}

func executeQAOA(ctx context.Context, command, source, output string, options qaoa.OptimizeOptions, out io.Writer) error {
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	var model qubo.Model
	if err := json.Unmarshal(data, &model); err != nil {
		return fmt.Errorf("%s: %w", source, err)
	}
	problem, err := qaoa.FromQUBO(model)
	if err != nil {
		return err
	}
	save := func(params qaoa.Parameters) error {
		code, err := problem.Export(params, qaoa.BuildOptions{Measure: true, MaxQubits: options.MaxQubits, MaxGates: options.MaxGates}, nil)
		if err != nil {
			return err
		}
		return writeOutput(output, code)
	}
	switch command {
	case "compile":
		return save(options.InitialParameters)
	case "run":
		evaluator, err := qaoa.NewLocalEvaluator(ctx, problem, qaoa.EvaluationOptions{Mode: qaoa.Sampled,
			Shots: options.Shots, Seed: options.Seed, MaxQubits: options.MaxQubits, MaxGates: options.MaxGates})
		if err != nil {
			return err
		}
		result, err := evaluator.Evaluate(ctx, options.InitialParameters)
		if err != nil {
			return err
		}
		printQAOASamples(out, options.InitialParameters, result)
	case "optimize":
		result, err := qaoa.Optimize(ctx, problem, options)
		if err != nil {
			return err
		}
		if output != "" {
			if err := save(result.BestParameters); err != nil {
				return err
			}
		}
		fmt.Fprintf(out, "Objective mode: %s\nInitial expected energy: %.12g\nBest expected energy: %.12g\nEvaluations: %d\nIterations: %d\nConverged: %t\nTermination: %s\n",
			result.Mode, result.InitialExpectedEnergy, result.BestExpectedEnergy, result.Evaluations, result.Iterations, result.Converged, result.TerminationReason)
		printQAOASamples(out, result.BestParameters, result.Final)
		if result.Classical != nil {
			fmt.Fprintf(out, "Classical minimum energy: %.12g\nOptimal sample fraction: %.6f\n", result.Classical.MinimumEnergy, result.Classical.OptimalSampleFraction)
		} else {
			fmt.Fprintf(out, "Classical minimum not calculated (%s)\n", result.ReferenceStatus)
		}
	}
	return nil
}

func printQAOASamples(out io.Writer, params qaoa.Parameters, result qaoa.Evaluation) {
	fmt.Fprintf(out, "Qubits: %d\nLayers: %d\nGammas: %v\nBetas: %v\nShots: %d\nEstimated expected energy: %.12g\n",
		result.QubitCount, result.LayerCount, params.Gammas, params.Betas, result.Shots, result.ExpectedEnergy)
	keys := make([]string, 0, len(result.Counts))
	for pattern := range result.Counts {
		keys = append(keys, pattern)
	}
	slices.Sort(keys)
	fmt.Fprintln(out, "Measurement counts:")
	for _, pattern := range keys {
		fmt.Fprintf(out, "%s: %d\n", pattern, result.Counts[pattern])
	}
	if best := result.BestObserved; best != nil {
		fmt.Fprintf(out, "Best observed assignment: %s\nBest observed energy: %.12g\nBest observed frequency: %.6f\n", best.Bitstring, best.Energy, best.Frequency)
	}
}
