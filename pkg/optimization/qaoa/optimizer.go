package qaoa

import (
	"context"
	"fmt"
	"math"
	"math/rand"

	"github.com/quvly/Qasmith/pkg/optimization/internal/energy"
)

// Objective returns the expectation to minimize for a set of parameters.
type Objective interface {
	Evaluate(context.Context, Parameters) (float64, error)
}

// ObjectiveFunc adapts a function into an Objective.
type ObjectiveFunc func(context.Context, Parameters) (float64, error)

func (f ObjectiveFunc) Evaluate(ctx context.Context, p Parameters) (float64, error) { return f(ctx, p) }

// Optimizer is a replaceable classical parameter search.
type Optimizer interface {
	Optimize(context.Context, Objective, Parameters) (SearchResult, error)
}

// SearchResult reports the lowest expectation evaluated by the search.
type SearchResult struct {
	BestParameters        Parameters
	BestExpectedEnergy    float64
	InitialExpectedEnergy float64
	Evaluations           int
	Iterations            int
	Converged             bool
	TerminationReason     string
}

// CoordinateSearch polls both signs of each coordinate. Restarts adds seeded
// starts beyond the supplied initial parameters, sharing the same budgets.
type CoordinateSearch struct {
	MaxEvaluations int
	MaxIterations  int
	Tolerance      float64
	Seed           int64
	Restarts       int
}

type searchPoint struct {
	params      Parameters
	value, step float64
}

const (
	defaultEvaluations = 200
	defaultIterations  = 100
	defaultTolerance   = 1e-4
	maxSearchWork      = 100_000
	maxRestarts        = 64
)

// Optimize minimizes an objective using round-robin coordinate sweeps.
// Converged means that every requested start reached the step tolerance.
func (s CoordinateSearch) Optimize(ctx context.Context, obj Objective, initial Parameters) (SearchResult, error) {
	if err := ctx.Err(); err != nil {
		return SearchResult{}, err
	}
	if obj == nil {
		return SearchResult{}, fmt.Errorf("missing objective")
	}
	layers := len(initial.Gammas)
	if err := initial.Validate(layers); err != nil {
		return SearchResult{}, err
	}
	if err := s.resolve(); err != nil {
		return SearchResult{}, err
	}
	var result SearchResult
	evaluate := func(p Parameters) (float64, error) {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		value, err := obj.Evaluate(ctx, p.clone())
		if err != nil {
			return 0, err
		}
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		if !energy.Finite(value) {
			return 0, fmt.Errorf("objective returned non-finite expected energy")
		}
		if result.Evaluations == 0 {
			result.InitialExpectedEnergy = value
		}
		if result.Evaluations == 0 || value < result.BestExpectedEnergy {
			result.BestExpectedEnergy = value
			result.BestParameters = p.clone()
		}
		result.Evaluations++
		return value, nil
	}
	random := rand.New(rand.NewSource(s.Seed))
	points := make([]searchPoint, 0, min(s.Restarts+1, s.MaxEvaluations))
	for i := 0; i <= s.Restarts && result.Evaluations < s.MaxEvaluations; i++ {
		p := initial.clone()
		if i != 0 {
			for k := range p.Gammas {
				p.Gammas[k] = (2*random.Float64() - 1) * math.Pi
				p.Betas[k] = (random.Float64() - .5) * math.Pi
			}
		}
		value, err := evaluate(p)
		if err != nil {
			return SearchResult{}, err
		}
		points = append(points, searchPoint{params: p, value: value, step: math.Pi / 4})
	}

	for {
		active := false
		for i := range points {
			point := &points[i]
			if point.step <= s.Tolerance {
				continue
			}
			if result.Evaluations == s.MaxEvaluations || result.Iterations == s.MaxIterations {
				break
			}
			active = true
			result.Iterations++
			improved, complete := false, true
		poll:
			for _, coordinates := range [][]float64{point.params.Gammas, point.params.Betas} {
				for j, center := range coordinates {
					best := center
					for _, sign := range []float64{-1, 1} {
						if result.Evaluations == s.MaxEvaluations {
							complete = false
							break
						}
						coordinates[j] = center + sign*point.step
						if coordinates[j] == center || !energy.Finite(coordinates[j]) {
							continue
						}
						value, err := evaluate(point.params)
						if err != nil {
							return SearchResult{}, err
						}
						if value < point.value {
							point.value = value
							best = coordinates[j]
							improved = true
						}
					}
					coordinates[j] = best
					if !complete {
						break poll
					}
				}
			}
			if !improved && complete {
				point.step /= 2
			}
		}
		allConverged := len(points) == s.Restarts+1
		for _, point := range points {
			allConverged = allConverged && point.step <= s.Tolerance
		}
		switch {
		case allConverged:
			result.Converged, result.TerminationReason = true, "step tolerance"
		case result.Evaluations == s.MaxEvaluations:
			result.TerminationReason = "evaluation limit"
		case result.Iterations == s.MaxIterations:
			result.TerminationReason = "iteration limit"
		case !active:
			return SearchResult{}, fmt.Errorf("coordinate search could not continue")
		default:
			continue
		}
		return result, nil
	}
}

func (s *CoordinateSearch) resolve() error {
	if s.MaxEvaluations == 0 {
		s.MaxEvaluations = defaultEvaluations
	}
	if s.MaxIterations == 0 {
		s.MaxIterations = defaultIterations
	}
	if s.Tolerance == 0 {
		s.Tolerance = defaultTolerance
	}
	if s.MaxEvaluations < 1 || s.MaxEvaluations > maxSearchWork || s.MaxIterations < 1 || s.MaxIterations > maxSearchWork {
		return fmt.Errorf("evaluation and iteration limits must be between 1 and %d", maxSearchWork)
	}
	if !energy.Finite(s.Tolerance) || s.Tolerance <= 0 {
		return fmt.Errorf("tolerance must be positive and finite")
	}
	if s.Restarts < 0 || s.Restarts > maxRestarts {
		return fmt.Errorf("restarts must be between 0 and %d", maxRestarts)
	}
	return nil
}
