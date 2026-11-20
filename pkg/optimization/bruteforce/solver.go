// Package bruteforce provides a bounded classical reference solver for QUBOs.
package bruteforce

import (
	"context"
	"fmt"
	"slices"

	"github.com/quvly/Qasmith/pkg/optimization/internal/energy"
	"github.com/quvly/Qasmith/pkg/optimization/qubo"
)

// DefaultMaxVariables is the limit for a zero-value Solver.
const DefaultMaxVariables = 20

// MaxSupportedVariables bounds even explicitly configured exhaustive searches.
const MaxSupportedVariables = 24

// Result contains the first optimal assignment in lexicographic order.
type Result struct {
	Assignment []bool
	Energy     float64
}

// BinaryObjective is an objective over ordered Boolean assignments.
type BinaryObjective interface {
	Variables() int
	Validate() error
	Evaluate([]bool) (float64, error)
}

// Solver configures exhaustive search. Zero MaxVariables uses the default.
type Solver struct{ MaxVariables int }

// Solve runs with the default limits.
func Solve(model qubo.Model) (Result, error) {
	return (Solver{}).Solve(context.Background(), model)
}

// Solve enumerates assignments in lexicographic order, with false before true.
// Energies that compare equal retain the first assignment.
func (s Solver) Solve(ctx context.Context, model qubo.Model) (Result, error) {
	return s.SolveObjective(ctx, model)
}

// SolveObjective uses the same bounded search for another binary objective.
func (s Solver) SolveObjective(ctx context.Context, model BinaryObjective) (Result, error) {
	limit := s.MaxVariables
	if limit == 0 {
		limit = DefaultMaxVariables
	}
	if limit < 1 || limit > MaxSupportedVariables {
		return Result{}, fmt.Errorf("solver variable limit must be between 1 and %d", MaxSupportedVariables)
	}
	if model == nil {
		return Result{}, fmt.Errorf("missing binary objective")
	}
	if err := model.Validate(); err != nil {
		return Result{}, err
	}
	n := model.Variables()
	if n < 1 {
		return Result{}, fmt.Errorf("variable count must be positive")
	}
	if n > limit {
		return Result{}, fmt.Errorf("model has %d variables, exhaustive search limit is %d", n, limit)
	}
	bits := make([]bool, n)
	var result Result
	for ordinal := uint64(0); ordinal < uint64(1)<<n; ordinal++ {
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}
		value, err := model.Evaluate(bits)
		if err != nil {
			return Result{}, fmt.Errorf("assignment %d: %w", ordinal, err)
		}
		if !energy.Finite(value) {
			return Result{}, fmt.Errorf("assignment %d: non-finite energy", ordinal)
		}
		if result.Assignment == nil {
			result = Result{Assignment: slices.Clone(bits), Energy: value}
		}
		if value < result.Energy {
			copy(result.Assignment, bits)
			result.Energy = value
		}
		for i := n - 1; i >= 0; i-- {
			bits[i] = !bits[i]
			if bits[i] {
				break
			}
		}
	}
	return result, nil
}
