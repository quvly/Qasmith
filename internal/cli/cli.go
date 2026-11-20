package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/quvly/Qasmith/internal/lsp"
	"github.com/quvly/Qasmith/pkg/optimization/qaoa"
	"github.com/quvly/Qasmith/pkg/qasmith"
)

const usage = "usage: qasmith <compile|run|check|eval|deps|optimize|qaoa|tsp|lsp|version> [file] [options]"

func Run(args []string, out, stderr io.Writer) int {
	return RunContext(context.Background(), args, out, stderr)
}

func RunContext(ctx context.Context, args []string, out, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, usage)
		return 2
	}
	command := args[0]
	args = args[1:]
	if command == "lsp" {
		if len(args) != 1 || args[0] != "--stdio" {
			fmt.Fprintln(stderr, "usage: qasmith lsp --stdio")
			return 2
		}
		if err := lsp.Serve(os.Stdin, out, stderr); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	}
	if command == "tsp" {
		return runTSP(ctx, args, out, stderr)
	}
	if command == "qaoa" {
		return runQAOA(ctx, args, out, stderr)
	}
	if command == "version" && len(args) == 0 {
		fmt.Fprintln(out, "Qasmith v0.9.1")
		return 0
	}
	if command == "help" || command == "--help" || command == "-h" {
		fmt.Fprintln(out, usage)
		return 0
	}
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(stderr)
	var output string
	var shots int
	var seed int64
	var optimization qaoa.OptimizeOptions
	var noSample bool
	compiler := qasmith.Compiler{}
	switch command {
	case "compile":
		flags.StringVar(&output, "o", "", "output file")
	case "run":
		flags.IntVar(&shots, "shots", 1000, "number of samples")
		flags.Int64Var(&seed, "seed", time.Now().UnixNano(), "random seed")
	case "optimize":
		flags.IntVar(&optimization.MaxEvaluations, "max-evals", 200, "maximum objective evaluations")
		flags.IntVar(&optimization.MaxIterations, "max-iters", 100, "maximum coordinate sweeps")
		flags.Float64Var(&optimization.Tolerance, "tolerance", 1e-4, "minimum coordinate step")
		flags.IntVar(&optimization.Restarts, "restarts", 0, "additional seeded starting points")
		flags.IntVar(&optimization.Shots, "shots", 1000, "shots for sampled search or final measurement")
		flags.Int64Var(&optimization.Seed, "seed", 0, "deterministic seed")
		flags.StringVar((*string)(&optimization.Mode), "eval-mode", "exact", "objective mode: exact or sampled")
		flags.StringVar(&output, "output-qasm", "", "save the optimized circuit")
		flags.BoolVar(&noSample, "no-sample", false, "skip final measurement")
	case "check", "deps", "eval":
	default:
		fmt.Fprintf(stderr, "unknown command or unexpected arguments: %s\n%s\n", command, usage)
		return 2
	}
	flags.IntVar(&compiler.Limits.MaxLoopIterations, "max-loop-iterations", 0, "maximum total loop iterations (0 uses default)")
	flags.IntVar(&compiler.Limits.MaxGates, "max-gates", 0, "maximum generated gates (0 uses default)")
	flags.IntVar(&compiler.Limits.MaxQubits, "max-qubits", 0, "maximum allocated qubits (0 uses default)")
	flags.IntVar(&compiler.Limits.MaxArrayElements, "max-array-elements", 0, "maximum compile-time array elements (0 uses default)")
	flags.IntVar(&compiler.Limits.MaxDepth, "max-depth", 0, "maximum static evaluation depth (0 uses default)")
	flags.IntVar(&compiler.Limits.MaxSteps, "max-steps", 0, "maximum static evaluation steps (0 uses default)")
	flags.IntVar(&compiler.Limits.MaxMatrixElements, "max-matrix-elements", 0, "trusted matrix element limit (0 uses default)")
	flags.IntVar(&compiler.Limits.MaxModelVariables, "max-model-variables", 0, "optimization model variable limit (0 uses default)")
	flags.IntVar(&compiler.Limits.MaxModelCoefficients, "max-model-coefficients", 0, "optimization coefficient limit (0 uses default)")
	flags.IntVar(&compiler.Limits.MaxStructuredWork, "max-structured-work", 0, "structured evaluation work limit (0 uses default)")
	flags.IntVar(&compiler.Limits.MaxCircuitLayers, "max-circuit-layers", 0, "terminal circuit layer limit (0 uses default)")
	flags.Usage = func() {
		fmt.Fprintf(stderr, "usage: qasmith %s file.smith [options]\n", command)
		flags.PrintDefaults()
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
	var err error
	if command == "optimize" {
		if optimization.Shots < 1 || output != "" && filepath.Ext(output) != ".qasm" {
			fmt.Fprintln(stderr, "shots must be positive and output must have a .qasm extension")
			return 2
		}
		err = optimizeSmith(ctx, compiler, source, output, optimization, !noSample, out)
	} else {
		err = execute(ctx, compiler, command, source, output, shots, seed, out)
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func execute(ctx context.Context, compiler qasmith.Compiler, command, source, output string, shots int, seed int64, out io.Writer) error {
	switch command {
	case "compile":
		if output == "" {
			output = strings.TrimSuffix(source, filepath.Ext(source)) + ".qasm"
		}
		if filepath.Ext(output) != ".qasm" {
			return fmt.Errorf("output must have a .qasm extension")
		}
		data, err := compiler.Compile(source)
		if err != nil {
			return err
		}
		return writeOutput(output, data)
	case "check":
		return compiler.Check(source)
	case "eval":
		value, err := compiler.Eval(source)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(out, value)
		return err
	case "deps":
		return compiler.Dependencies(source, func(path, prefix string, repeated bool) {
			if repeated {
				path += " (shared)"
			}
			fmt.Fprintln(out, prefix+path)
		})
	case "run":
		prepared, err := compiler.Prepare(source)
		if err != nil {
			return err
		}
		execution, err := prepared.Execute(ctx, shots, seed)
		if execution.Optimization != nil && execution.Raw.Shots > 0 {
			printPlanned(out, *execution.Optimization)
			return err
		}
		if err != nil {
			return err
		}
		result := execution.Raw
		fmt.Fprintf(out, "Qasmith Quantum Simulator\n\nQubits: %d\nShots: %d\n\nResults:\n", result.Qubits, result.Shots)
		var outcomes []string
		for bits := range result.Counts {
			outcomes = append(outcomes, bits)
		}
		slices.Sort(outcomes)
		for _, bits := range outcomes {
			count := result.Counts[bits]
			fmt.Fprintf(out, "%s: %d (%.1f%%)\n", bits, count, 100*float64(count)/float64(result.Shots))
		}
	}
	return nil
}

func writeOutput(path string, data []byte) error {
	file, err := os.CreateTemp(filepath.Dir(path), ".qasmith-*.qasm")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	_, writeErr := file.Write(data)
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(file.Name(), path)
}
