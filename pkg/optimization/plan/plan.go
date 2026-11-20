// Package plan bridges immutable Smith optimization values, circuit preparation,
// and generic QAOA execution. It is independent of any application decoder.
package plan

import (
	"context"
	"fmt"
	"slices"

	"github.com/quvly/Qasmith/pkg/extension"
	"github.com/quvly/Qasmith/pkg/ir"
	"github.com/quvly/Qasmith/pkg/optimization/qaoa"
	"github.com/quvly/Qasmith/pkg/optimization/qubo"
)

// Metric is a display field supplied by a typed domain report.
type Metric struct{ Label, Value string }

// DomainResult can carry a typed application report alongside display metrics.
type DomainResult interface{ Metrics() []Metric }

// Decoder interprets raw measurements without replacing them.
type Decoder func(context.Context, qaoa.Evaluation) (DomainResult, error)

// Problem is an immutable QUBO-backed objective with an optional decoder.
// A decoder must capture only immutable state and support concurrent calls.
type Problem struct {
	quantum   qaoa.Problem
	objective qubo.Model
	decode    Decoder
}

// FromQUBO validates and converts an objective once.
func FromQUBO(model qubo.Model, decode Decoder) (Problem, error) {
	p, err := qaoa.FromQUBO(model)
	if err != nil {
		return Problem{}, err
	}
	return FromPrepared(p, decode)
}

// FromModel retains a validated generic objective. Conversion is deferred
// until the final circuit plan is created, so intermediate Smith models
// can be composed or scaled before Ising representation is required.
func FromModel(model qubo.Model) (Problem, error) {
	if err := model.Validate(); err != nil {
		return Problem{}, err
	}
	return Problem{objective: model}, nil
}

// HasDecoder reports whether changing this objective may invalidate a decoder.
func (p Problem) HasDecoder() bool { return p.decode != nil }

// FromPrepared reuses a QUBO-derived generic QAOA problem.
func FromPrepared(p qaoa.Problem, decode Decoder) (Problem, error) {
	q, ok := p.OriginalQUBO()
	if !ok {
		return Problem{}, fmt.Errorf("optimization problem requires a validated QUBO objective")
	}
	return Problem{quantum: p, objective: q, decode: decode}, nil
}

func (p Problem) QUBO() qubo.Model { return p.objective }

// QAOA returns the converted problem for prepared values. For an intermediate
// value from FromModel, this is available after creating a circuit plan.
func (p Problem) QAOA() qaoa.Problem { return p.quantum }

// OptimizationProblem identifies the public Smith type and its Go payload.
var optimizationType = newOptimizationType()

func newOptimizationType() *extension.OpaqueType[Problem] {
	t, err := extension.NewCheckedOpaqueType("OptimizationProblem", func(p Problem) (Problem, error) {
		if p.objective.Variables() < 1 {
			return Problem{}, fmt.Errorf("invalid optimization problem")
		}
		return p, nil
	}, func(ctx extension.StaticContext, p Problem) error {
		l := ctx.Limits()
		return validateProblem(p, l.MaxModelVariables, l.MaxModelCoefficients)
	})
	if err != nil {
		panic(err)
	}
	return t
}

func OptimizationProblem() *extension.OpaqueType[Problem] { return optimizationType }

// Plan is a reusable measured QAOA recipe. Build has no execution side effects.
type Plan struct {
	problem    Problem
	parameters qaoa.Parameters
	options    qaoa.BuildOptions
}

// New validates parameters and model limits without executing the circuit.
func New(problem Problem, params qaoa.Parameters, ctx extension.StaticContext) (Plan, error) {
	l := ctx.Limits()
	if err := validateProblem(problem, l.MaxModelVariables, l.MaxModelCoefficients); err != nil {
		return Plan{}, err
	}
	if len(params.Gammas) > l.MaxCircuitLayers {
		return Plan{}, fmt.Errorf("circuit layers limit exceeded (maximum %d)", l.MaxCircuitLayers)
	}
	if err := params.Validate(len(params.Gammas)); err != nil {
		return Plan{}, err
	}
	linear, pairs := problem.objective.TermCounts()
	for _, amount := range []int{problem.objective.Variables(), linear, pairs, len(params.Gammas)} {
		if err := ctx.Spend(amount); err != nil {
			return Plan{}, err
		}
	}
	if problem.quantum.Ising().Variables() == 0 {
		quantum, err := qaoa.FromQUBO(problem.objective)
		if err != nil {
			return Plan{}, err
		}
		problem.quantum = quantum
	}
	return Plan{problem: problem, parameters: copyParameters(params), options: qaoa.BuildOptions{Measure: true, MaxQubits: l.MaxQubits, MaxGates: l.MaxGates}}, nil
}

func validateProblem(p Problem, maxVariables, maxCoefficients int) error {
	n := p.objective.Variables()
	if n < 1 {
		return fmt.Errorf("invalid optimization problem")
	}
	if n > maxVariables {
		return fmt.Errorf("optimization variable limit exceeded (maximum %d)", maxVariables)
	}
	linear, pairs := p.objective.TermCounts()
	if linear >= maxCoefficients || pairs > maxCoefficients-1-linear {
		return fmt.Errorf("model coefficient limit exceeded (maximum %d)", maxCoefficients)
	}
	return nil
}

func copyParameters(p qaoa.Parameters) qaoa.Parameters {
	return qaoa.Parameters{Gammas: slices.Clone(p.Gammas), Betas: slices.Clone(p.Betas)}
}

func (p Plan) Problem() Problem            { return p.problem }
func (p Plan) Parameters() qaoa.Parameters { return copyParameters(p.parameters) }
func (p Plan) Layers() int                 { return len(p.parameters.Gammas) }
func (p Plan) Variables() int              { return p.problem.objective.Variables() }

func (p Plan) Build() (*ir.Program, error) { return p.problem.quantum.Build(p.parameters, p.options) }

// Optimize searches the existing objective and returns a revised immutable plan.
// The caller may build and sample that revised plan separately.
func (p Plan) Optimize(ctx context.Context, opts qaoa.OptimizeOptions) (qaoa.Result, Plan, error) {
	opts.Layers = p.Layers()
	opts.InitialParameters = p.Parameters()
	opts.MaxQubits, opts.MaxGates = p.options.MaxQubits, p.options.MaxGates
	opts.Reference, opts.SkipFinalSample = false, true
	result, err := qaoa.Optimize(ctx, p.problem.quantum, opts)
	if err != nil {
		return qaoa.Result{}, Plan{}, err
	}
	p.parameters = copyParameters(result.BestParameters)
	return result, p, nil
}
