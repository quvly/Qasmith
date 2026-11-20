package qasmith

import (
	"context"

	"github.com/quvly/Qasmith/internal/diagnostic"
	"github.com/quvly/Qasmith/internal/lower"
	"github.com/quvly/Qasmith/pkg/optimization/plan"
	"github.com/quvly/Qasmith/pkg/optimization/qaoa"
)

// Execution preserves raw simulation output and optional optimization metadata.
type Execution struct {
	Raw          Simulation
	Optimization *plan.Result
}

// Execute samples a prepared circuit and invokes its registered decoder, if any.
// When decoding fails the returned result still contains the raw measurements.
func (p Prepared) Execute(ctx context.Context, shots int, seed int64) (Execution, error) {
	if recipe, ok := p.Plan.(plan.Plan); ok {
		result, err := recipe.Run(ctx, p.Circuit, shots, seed)
		out := Execution{Raw: Simulation{Qubits: result.Raw.QubitCount, Shots: result.Raw.Shots, Counts: result.Raw.Counts},
			Optimization: &result}
		if err != nil {
			return out, diagnostic.Wrap(diagnostic.Position(p.Origin.Location), "E5001", err, "execute optimization plan")
		}
		return out, nil
	}
	raw, err := SimulateIR(ctx, p.Circuit, shots, seed)
	return Execution{Raw: raw}, err
}

// Optimized contains the generic search metrics, revised circuit, and optional samples.
type Optimized struct {
	Search   qaoa.Result
	Prepared Prepared
	Samples  *Execution
}

// Optimize searches the source plan's objective using its initial parameters.
// Final sampling may be disabled with sample=false. No source files are modified.
func (p Prepared) Optimize(ctx context.Context, opts qaoa.OptimizeOptions, sample bool) (Optimized, error) {
	current, ok := p.Plan.(plan.Plan)
	if !ok {
		return Optimized{}, diagnostic.At(p.entry, "E5002", "program has no supported variational optimization plan")
	}
	origin := diagnostic.Position(p.Origin.Location)
	if sample {
		if opts.Shots == 0 {
			opts.Shots = 1000
		}
		if opts.Shots < 1 || opts.Shots > qaoa.MaxShots {
			return Optimized{}, diagnostic.At(origin, "E5002", "shots must be between 1 and %d", qaoa.MaxShots)
		}
	}
	search, next, err := current.Optimize(ctx, opts)
	if err != nil {
		return Optimized{}, diagnostic.Wrap(origin, "E5002", err, "optimize plan")
	}
	circuit, err := next.Build()
	if err != nil {
		return Optimized{}, diagnostic.Wrap(origin, "E5002", err, "build optimized circuit")
	}
	p.Circuit, err = lower.PrepareProduced(circuit, p.Origin, p.limits)
	if err != nil {
		return Optimized{}, diagnostic.Wrap(origin, "E5002", err, "validate optimized circuit")
	}
	p.Plan = next
	if err := p.validatePasses(); err != nil {
		return Optimized{}, err
	}
	result := Optimized{Search: search, Prepared: p}
	if sample {
		final, err := p.Execute(ctx, opts.Shots, qaoa.FinalSampleSeed(opts.Seed))
		result.Samples = &final
		if err != nil {
			return result, err
		}
	}
	return result, nil
}
