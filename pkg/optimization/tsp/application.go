package tsp

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/quvly/Qasmith/pkg/optimization/internal/energy"
	"github.com/quvly/Qasmith/pkg/optimization/qaoa"
	"github.com/quvly/Qasmith/pkg/optimization/qubo"
)

// Application retains a validated TSP encoding and its generic QAOA problem.
type Application struct {
	problem  Problem
	encoding Encoding
	model    qubo.Model
	quantum  qaoa.Problem
	penalty  float64
}

// Prepare builds the QUBO and converts it to the reusable generic QAOA problem.
func Prepare(p Problem, opts EncodeOptions) (Application, error) {
	model, encoding, err := ToQUBO(p, opts)
	if err != nil {
		return Application{}, err
	}
	quantum, err := qaoa.FromQUBO(model)
	if err != nil {
		return Application{}, err
	}
	penalty, err := resolvePenalty(p, opts.Penalty)
	if err != nil {
		return Application{}, err
	}
	return Application{problem: p, encoding: encoding, model: model, quantum: quantum, penalty: penalty}, nil
}

func (a Application) QUBO() qubo.Model   { return a.model }
func (a Application) Encoding() Encoding { return a.encoding }
func (a Application) QAOA() qaoa.Problem { return a.quantum }
func (a Application) Penalty() float64   { return a.penalty }

// ReferenceOptions enables independent tour search, subject to its city limit.
type ReferenceOptions struct {
	Enabled   bool
	MaxCities int
}

// RunOptions configures measurement and optional additional exact evaluation.
type RunOptions struct {
	Shots     int
	Seed      int64
	MaxQubits int
	MaxGates  int
	Exact     bool
	Reference ReferenceOptions
}

// Outcome describes one unique observed bitstring and its objective components.
// Each feasible bitstring encodes exactly one distinct closed route.
type Outcome struct {
	Bitstring         string
	Count             int
	Frequency         float64
	QUBOEnergy        float64
	TravelObjective   float64
	ConstraintPenalty float64
	Decoded           Decoded
}

// Result keeps measurements, valid route metrics, and reference results separate.
// BestFeasible is nil when no valid tour was observed. OptimalSampleRate is nil
// when the independent reference was not calculated.
type Result struct {
	CityCount, QubitCount, QAOALayers, Shots int
	ExpectedQUBOEnergy                       float64
	ExactExpectedEnergy                      *float64
	FeasibleSamples, InfeasibleSamples       int
	FeasibilityRate                          float64
	Raw                                      qaoa.Evaluation
	Feasible, Infeasible                     []Outcome
	BestFeasible, BestObserved               *Outcome
	Classical                                *ClassicalResult
	ReferenceStatus                          string
	OptimalTourSamples                       int
	OptimalSampleRate                        *float64
}

// Run samples the generic QAOA circuit and decodes its outcomes.
func (a Application) Run(ctx context.Context, params qaoa.Parameters, opts RunOptions) (Result, error) {
	if err := params.Validate(len(params.Gammas)); err != nil {
		return Result{}, err
	}
	settings := qaoa.EvaluationOptions{Mode: qaoa.Sampled, Shots: opts.Shots, Seed: opts.Seed, MaxQubits: opts.MaxQubits, MaxGates: opts.MaxGates}
	sampler, err := qaoa.NewLocalEvaluator(ctx, a.quantum, settings)
	if err != nil {
		return Result{}, err
	}
	samples, err := sampler.Evaluate(ctx, params)
	if err != nil {
		return Result{}, err
	}
	result, err := a.Report(ctx, samples, opts.Reference)
	if err != nil {
		return Result{}, err
	}
	if opts.Exact {
		exact, err := sampler.Exact(ctx, params)
		if err != nil {
			return Result{}, err
		}
		result.ExactExpectedEnergy = &exact.ExpectedEnergy
	}
	return result, nil
}

// Optimize delegates search to the generic QAOA engine, then interprets its final
// measurements. Reference options here control independent tour enumeration.
func (a Application) Optimize(ctx context.Context, opts qaoa.OptimizeOptions, reference ReferenceOptions) (qaoa.Result, Result, error) {
	search, err := qaoa.Optimize(ctx, a.quantum, opts)
	if err != nil {
		return qaoa.Result{}, Result{}, err
	}
	result, err := a.Report(ctx, search.Final, reference)
	return search, result, err
}

// Report decodes a set of measured bitstrings and evaluates QUBO energy with the
// original model. It also accepts externally acquired counts with the same ordering.
func (a Application) Report(ctx context.Context, raw qaoa.Evaluation, reference ReferenceOptions) (Result, error) {
	if a.encoding.variables == 0 || raw.Mode != qaoa.Sampled || raw.Shots < 1 || raw.QubitCount != a.encoding.variables || raw.LayerCount < 1 {
		return Result{}, fmt.Errorf("invalid sampled evaluation for the TSP encoding")
	}
	raw.Counts = maps.Clone(raw.Counts)
	result := Result{CityCount: a.problem.Cities(), QubitCount: raw.QubitCount, QAOALayers: raw.LayerCount, Shots: raw.Shots, Raw: raw, ReferenceStatus: "disabled"}
	patterns := make([]string, 0, len(raw.Counts))
	for pattern := range raw.Counts {
		patterns = append(patterns, pattern)
	}
	slices.Sort(patterns)
	var expected energy.Sum
	seen := 0
	for _, pattern := range patterns {
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}
		count := raw.Counts[pattern]
		if count < 1 || count > raw.Shots-seen {
			return Result{}, fmt.Errorf("invalid count for %q", pattern)
		}
		seen += count
		if len(pattern) != a.encoding.variables {
			return Result{}, fmt.Errorf("invalid bitstring length for %q", pattern)
		}
		bits, err := qaoa.DecodeBitstring(pattern)
		if err != nil {
			return Result{}, fmt.Errorf("bitstring %q: %w", pattern, err)
		}
		decoded, err := a.problem.Decode(bits)
		if err != nil {
			return Result{}, err
		}
		value, err := a.model.Evaluate(bits)
		if err != nil {
			return Result{}, err
		}
		travel, penalty, err := a.components(bits, decoded)
		if err != nil {
			return Result{}, err
		}
		frequency := float64(count) / float64(raw.Shots)
		if err := expected.Add(frequency * value); err != nil {
			return Result{}, err
		}
		item := Outcome{Bitstring: pattern, Count: count, Frequency: frequency, QUBOEnergy: value, TravelObjective: travel, ConstraintPenalty: penalty, Decoded: decoded}
		if result.BestObserved == nil || value < result.BestObserved.QUBOEnergy {
			best := item
			result.BestObserved = &best
		}
		if decoded.Feasible {
			result.FeasibleSamples += count
			result.Feasible = append(result.Feasible, item)
			if result.BestFeasible == nil || decoded.Distance < result.BestFeasible.Decoded.Distance ||
				decoded.Distance == result.BestFeasible.Decoded.Distance && slices.Compare(decoded.Route, result.BestFeasible.Decoded.Route) < 0 {
				best := item
				result.BestFeasible = &best
			}
		} else {
			result.InfeasibleSamples += count
			result.Infeasible = append(result.Infeasible, item)
		}
	}
	if seen != raw.Shots {
		return Result{}, fmt.Errorf("counts total %d, expected %d shots", seen, raw.Shots)
	}
	var err error
	result.ExpectedQUBOEnergy, err = expected.Value()
	if err != nil {
		return Result{}, err
	}
	result.FeasibilityRate = float64(result.FeasibleSamples) / float64(raw.Shots)
	if err := a.attachReference(ctx, &result, reference); err != nil {
		return Result{}, err
	}
	return result, nil
}

func (a Application) components(bits []bool, decoded Decoded) (float64, float64, error) {
	if decoded.Feasible {
		return decoded.Distance, 0, nil
	}
	var travel, penalty energy.Sum
	m := a.encoding.cities - 1
	start := a.encoding.start
	for i := range m {
		city := a.encoding.city(i)
		if bits[i] {
			if err := travel.Add(a.problem.distances[start][city]); err != nil {
				return 0, 0, err
			}
		}
		if bits[(m-1)*m+i] {
			if err := travel.Add(a.problem.distances[city][start]); err != nil {
				return 0, 0, err
			}
		}
		for pos := 0; pos < m-1; pos++ {
			if !bits[pos*m+i] {
				continue
			}
			for j := range m {
				if bits[(pos+1)*m+j] {
					if err := travel.Add(a.problem.distances[city][a.encoding.city(j)]); err != nil {
						return 0, 0, err
					}
				}
			}
		}
	}
	for _, v := range decoded.Violations {
		delta := float64(v.Count - v.Expected)
		if err := penalty.Add(a.penalty * delta * delta); err != nil {
			return 0, 0, err
		}
	}
	d, err := travel.Value()
	if err != nil {
		return 0, 0, err
	}
	p, err := penalty.Value()
	return d, p, err
}

func (a Application) attachReference(ctx context.Context, result *Result, opts ReferenceOptions) error {
	if !opts.Enabled {
		return nil
	}
	limit := opts.MaxCities
	if limit == 0 {
		limit = DefaultClassicalLimit
	}
	if limit < 3 || limit > MaxClassicalLimit {
		return fmt.Errorf("classical city limit must be between 3 and %d", MaxClassicalLimit)
	}
	if a.problem.Cities() > limit {
		result.ReferenceStatus = fmt.Sprintf("above the %d-city permutation limit", limit)
		return nil
	}
	best, err := SolveClassical(ctx, a.problem, limit)
	if err != nil {
		return err
	}
	result.Classical, result.ReferenceStatus = &best, "completed"
	for _, sample := range result.Feasible {
		if sample.Decoded.Distance == best.Distance {
			result.OptimalTourSamples += sample.Count
		}
	}
	rate := float64(result.OptimalTourSamples) / float64(result.Shots)
	result.OptimalSampleRate = &rate
	return nil
}
