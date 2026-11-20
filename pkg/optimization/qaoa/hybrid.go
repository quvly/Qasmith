package qaoa

import (
	"context"
	"fmt"

	"github.com/quvly/Qasmith/pkg/optimization/bruteforce"
	"github.com/quvly/Qasmith/pkg/optimization/internal/energy"
	"github.com/quvly/Qasmith/pkg/optimization/ising"
)

// OptimizeOptions configures local hybrid optimization. Non-nil Evaluator and
// Optimizer replace only the search objective evaluator and search strategy.
type OptimizeOptions struct {
	Layers            int
	InitialParameters Parameters
	MaxEvaluations    int
	MaxIterations     int
	Tolerance         float64
	Seed              int64
	Restarts          int
	Mode              EvaluationMode
	Shots             int
	MaxQubits         int
	MaxGates          int
	Reference         bool
	SkipFinalSample   bool
	ReferenceLimit    int
	Evaluator         Evaluator
	Optimizer         Optimizer
}

// ClassicalReference reports an independently computed minimum and the
// fraction of final samples whose reference energy equals that optimum.
type ClassicalReference struct {
	MinimumEnergy         float64
	Assignment            []bool
	OptimalSampleFraction float64
}

// Result distinguishes the best search expectation from final sampled results
// and an optional classical optimum.
type Result struct {
	SearchResult
	Mode            EvaluationMode
	Final           Evaluation
	Classical       *ClassicalReference
	ReferenceStatus string
}

// Optimize runs the search and measures its best parameters on the local
// simulator. The final sample uses a separate deterministic seed.
func Optimize(ctx context.Context, p Problem, opts OptimizeOptions) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if opts.Layers == 0 {
		opts.Layers = 1
	}
	if opts.Shots == 0 {
		opts.Shots = 1000
	}
	if opts.Mode == "" {
		opts.Mode = Exact
	}
	if opts.SkipFinalSample && opts.Reference {
		return Result{}, fmt.Errorf("reference sampling requires a final sample")
	}
	if opts.Mode == Sampled || !opts.SkipFinalSample {
		if err := checkShots(opts.Shots); err != nil {
			return Result{}, err
		}
	}
	budgets := BuildOptions{MaxQubits: opts.MaxQubits, MaxGates: opts.MaxGates}
	if _, _, err := p.gateBudget(opts.Layers, budgets); err != nil {
		return Result{}, err
	}
	initial := opts.InitialParameters
	if initial.Gammas == nil && initial.Betas == nil {
		initial = Parameters{Gammas: make([]float64, opts.Layers), Betas: make([]float64, opts.Layers)}
		for i := range opts.Layers {
			initial.Gammas[i], initial.Betas[i] = .4, .3
		}
	}
	if err := initial.Validate(opts.Layers); err != nil {
		return Result{}, err
	}
	search := opts.Optimizer
	if search == nil {
		settings := CoordinateSearch{MaxEvaluations: opts.MaxEvaluations, MaxIterations: opts.MaxIterations,
			Tolerance: opts.Tolerance, Seed: opts.Seed, Restarts: opts.Restarts}
		if err := settings.resolve(); err != nil {
			return Result{}, err
		}
		search = settings
	}
	local, err := NewLocalEvaluator(ctx, p, EvaluationOptions{Mode: opts.Mode, Shots: opts.Shots, Seed: opts.Seed,
		MaxQubits: opts.MaxQubits, MaxGates: opts.MaxGates})
	if err != nil {
		return Result{}, err
	}
	evaluator := opts.Evaluator
	if evaluator == nil {
		evaluator = local
	}
	objective := ObjectiveFunc(func(ctx context.Context, params Parameters) (float64, error) {
		if err := params.Validate(opts.Layers); err != nil {
			return 0, err
		}
		evaluation, err := evaluator.Evaluate(ctx, params)
		if err != nil {
			return 0, err
		}
		if !energy.Finite(evaluation.ExpectedEnergy) {
			return 0, fmt.Errorf("evaluator returned non-finite expected energy")
		}
		return evaluation.ExpectedEnergy, nil
	})
	optimized, err := search.Optimize(ctx, objective, initial.clone())
	if err != nil {
		return Result{}, err
	}
	if err := optimized.BestParameters.Validate(opts.Layers); err != nil {
		return Result{}, err
	}
	if !energy.Finite(optimized.InitialExpectedEnergy) || !energy.Finite(optimized.BestExpectedEnergy) {
		return Result{}, fmt.Errorf("optimizer returned non-finite expected energy")
	}
	result := Result{SearchResult: optimized, Mode: opts.Mode, ReferenceStatus: "disabled"}
	if opts.SkipFinalSample {
		return result, nil
	}
	final, err := local.evaluate(ctx, optimized.BestParameters, Sampled, opts.Shots, FinalSampleSeed(opts.Seed))
	if err != nil {
		return Result{}, err
	}
	result.Final = final
	if opts.Reference {
		result.Classical, result.ReferenceStatus, err = p.reference(ctx, final, opts.ReferenceLimit)
		if err != nil {
			return Result{}, err
		}
	}
	return result, nil
}

func (p Problem) reference(ctx context.Context, final Evaluation, limit int) (*ClassicalReference, string, error) {
	if limit == 0 {
		limit = bruteforce.DefaultMaxVariables
	}
	if limit < 1 || limit > bruteforce.MaxSupportedVariables {
		return nil, "", fmt.Errorf("reference variable limit must be between 1 and %d", bruteforce.MaxSupportedVariables)
	}
	var objective bruteforce.BinaryObjective = spinObjective{p.model}
	if p.original != nil {
		objective = *p.original
	}
	if objective.Variables() > limit {
		return nil, fmt.Sprintf("above the %d-variable reference limit", limit), nil
	}
	optimum, err := (bruteforce.Solver{MaxVariables: limit}).SolveObjective(ctx, objective)
	if err != nil {
		return nil, "", err
	}
	optimal := 0
	for _, pattern := range sortedPatterns(final.Counts) {
		if err := ctx.Err(); err != nil {
			return nil, "", err
		}
		bits, err := DecodeBitstring(pattern)
		if err != nil {
			return nil, "", err
		}
		value, err := objective.Evaluate(bits)
		if err != nil {
			return nil, "", err
		}
		if value == optimum.Energy {
			optimal += final.Counts[pattern]
		}
	}
	return &ClassicalReference{MinimumEnergy: optimum.Energy, Assignment: optimum.Assignment,
		OptimalSampleFraction: float64(optimal) / float64(final.Shots)}, "completed", nil
}

type spinObjective struct{ ising.Model }

func (s spinObjective) Evaluate(bits []bool) (float64, error) {
	return s.Model.Evaluate(ising.Spins(bits))
}

// FinalSampleSeed separates the final measurement stream from the search stream.
func FinalSampleSeed(seed int64) int64 { return seed ^ 0x5deece66d }
