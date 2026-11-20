package plan

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/quvly/Qasmith/pkg/ir"
	"github.com/quvly/Qasmith/pkg/optimization/internal/energy"
	"github.com/quvly/Qasmith/pkg/optimization/qaoa"
)

// Observation retains one measured bitstring, its count, probability, and original QUBO energy.
type Observation struct {
	Bitstring   string
	Count       int
	Probability float64
	QUBOEnergy  float64
}

// Result preserves raw data even if a domain decoder reports an error.
type Result struct {
	Raw                qaoa.Evaluation
	Observations       []Observation
	ExpectedQUBOEnergy float64
	BestObserved       *Observation
	Domain             DomainResult
}

// Run samples the supplied prepared circuit and interprets its QUBO assignments.
func (p Plan) Run(ctx context.Context, circuit *ir.Program, shots int, seed int64) (Result, error) {
	evaluator, err := qaoa.NewLocalEvaluator(ctx, p.problem.quantum, qaoa.EvaluationOptions{Mode: qaoa.Sampled, Shots: shots, Seed: seed,
		MaxQubits: p.options.MaxQubits, MaxGates: p.options.MaxGates})
	if err != nil {
		return Result{}, err
	}
	raw, err := evaluator.SampleCircuit(ctx, circuit, p.Layers(), shots, seed)
	if err != nil {
		return Result{}, err
	}
	result := Result{Raw: raw}
	keys := make([]string, 0, len(raw.Counts))
	for bits := range raw.Counts {
		keys = append(keys, bits)
	}
	slices.Sort(keys)
	var sum energy.Sum
	for _, pattern := range keys {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		assignment, err := qaoa.DecodeBitstring(pattern)
		if err != nil {
			return result, err
		}
		value, err := p.problem.objective.Evaluate(assignment)
		if err != nil {
			return result, err
		}
		count := raw.Counts[pattern]
		weight := float64(count) / float64(shots)
		record := Observation{Bitstring: pattern, Count: count, Probability: weight, QUBOEnergy: value}
		result.Observations = append(result.Observations, record)
		if result.BestObserved == nil || value < result.BestObserved.QUBOEnergy {
			best := record
			result.BestObserved = &best
		}
		if err := sum.Add(weight * value); err != nil {
			return result, err
		}
	}
	result.ExpectedQUBOEnergy, err = sum.Value()
	if err != nil {
		return result, err
	}
	if p.problem.decode != nil {
		input := raw
		input.Counts = maps.Clone(raw.Counts)
		if raw.BestObserved != nil {
			best := *raw.BestObserved
			best.Assignment = slices.Clone(best.Assignment)
			input.BestObserved = &best
		}
		result.Domain, err = p.problem.decode(ctx, input)
		if err != nil {
			return result, fmt.Errorf("result decoder: %w", err)
		}
	}
	return result, nil
}
