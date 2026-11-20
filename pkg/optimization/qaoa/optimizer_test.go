package qaoa_test

import (
	"context"
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/quvly/Qasmith/pkg/optimization/ising"
	"github.com/quvly/Qasmith/pkg/optimization/qaoa"
)

func quadratic(_ context.Context, p qaoa.Parameters) (float64, error) {
	g, b := p.Gammas[0]-.16, p.Betas[0]+.53
	return g*g + b*b, nil
}

func TestCoordinateSearch(t *testing.T) {
	ctx := context.Background()
	opt := qaoa.CoordinateSearch{Seed: 17, Restarts: 2, MaxEvaluations: 500, MaxIterations: 110, Tolerance: 1e-4}
	start := params(2, -2)
	visits := 0
	objective := qaoa.ObjectiveFunc(func(ctx context.Context, p qaoa.Parameters) (float64, error) {
		visits++
		value, err := quadratic(ctx, p)
		p.Gammas[0] = math.Inf(1) // Callbacks work on their own parameter slices.
		return value, err
	})
	result, err := opt.Optimize(ctx, objective, start)
	must(t, err)
	if !result.Converged || result.TerminationReason != "step tolerance" || result.BestExpectedEnergy > 1e-7 ||
		result.InitialExpectedEnergy <= result.BestExpectedEnergy || result.Evaluations != visits || start.Gammas[0] != 2 {
		t.Fatalf("result %+v after %d visits", result, visits)
	}
	again, err := opt.Optimize(ctx, qaoa.ObjectiveFunc(quadratic), start)
	must(t, err)
	if !reflect.DeepEqual(again, result) {
		t.Fatalf("seeded runs differ: %+v %+v", result, again)
	}
}

func TestSearchTerminationAndErrors(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		search qaoa.CoordinateSearch
		reason string
		evals  int
	}{
		{qaoa.CoordinateSearch{MaxEvaluations: 2, Tolerance: .5}, "evaluation limit", 2},
		{qaoa.CoordinateSearch{MaxIterations: 1}, "iteration limit", 5},
		{qaoa.CoordinateSearch{MaxEvaluations: 3, Restarts: 4}, "evaluation limit", 3},
	} {
		result, err := tc.search.Optimize(ctx, qaoa.ObjectiveFunc(quadratic), params(.16, -.53))
		must(t, err)
		if result.Converged || result.TerminationReason != tc.reason || result.Evaluations != tc.evals {
			t.Fatalf("result %+v", result)
		}
	}
	for _, tc := range []qaoa.CoordinateSearch{
		{MaxEvaluations: -1}, {MaxIterations: -1}, {Tolerance: math.NaN()}, {Restarts: -1},
	} {
		if _, err := tc.Optimize(ctx, qaoa.ObjectiveFunc(quadratic), params(0, 0)); err == nil {
			t.Fatalf("accepted %+v", tc)
		}
	}
	if _, err := (qaoa.CoordinateSearch{}).Optimize(ctx, qaoa.ObjectiveFunc(quadratic), params(0, math.Inf(1))); err == nil {
		t.Fatal("accepted invalid initial parameters")
	}
	sentinel := errors.New("objective error")
	failure := qaoa.ObjectiveFunc(func(context.Context, qaoa.Parameters) (float64, error) { return 0, sentinel })
	if _, err := (qaoa.CoordinateSearch{}).Optimize(ctx, failure, params(0, 0)); !errors.Is(err, sentinel) {
		t.Fatalf("error %v", err)
	}
	bad := qaoa.ObjectiveFunc(func(context.Context, qaoa.Parameters) (float64, error) { return math.NaN(), nil })
	if _, err := (qaoa.CoordinateSearch{}).Optimize(ctx, bad, params(0, 0)); err == nil || !strings.Contains(err.Error(), "non-finite") {
		t.Fatalf("error %v", err)
	}
	ctx, cancel := context.WithCancel(ctx)
	failure = qaoa.ObjectiveFunc(func(context.Context, qaoa.Parameters) (float64, error) { cancel(); return 0, nil })
	if _, err := (qaoa.CoordinateSearch{}).Optimize(ctx, failure, params(0, 0)); !errors.Is(err, context.Canceled) {
		t.Fatalf("error %v", err)
	}
}

func TestHybridOptimization(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name   string
		matrix [][]float64
		layers int
		mode   qaoa.EvaluationMode
		offset float64
	}{
		{"one", [][]float64{{1}}, 1, qaoa.Exact, 0},
		{"two", [][]float64{{-2, 4}, {0, 3}}, 1, qaoa.Exact, 1},
		{"three sparse", [][]float64{{.3, 0, -.7}, {0, -.7, 0}, {0, 0, .8}}, 2, qaoa.Exact, 2},
		{"three dense sampled", [][]float64{{-.5, .8, -.3}, {0, 1.2, .4}, {0, 0, -.75}}, 2, qaoa.Sampled, 2.75},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := fromMatrix(t, tc.matrix, tc.offset)
			opts := qaoa.OptimizeOptions{Mode: tc.mode, Layers: tc.layers, MaxEvaluations: 150, Restarts: 2, Shots: 256, Seed: 42, Reference: true}
			result, err := qaoa.Optimize(ctx, p, opts)
			must(t, err)
			if result.Classical == nil || result.Final.BestObserved == nil || result.Evaluations > opts.MaxEvaluations ||
				result.BestExpectedEnergy > result.InitialExpectedEnergy || result.BestExpectedEnergy < result.Classical.MinimumEnergy-1e-12 ||
				result.Final.Shots != opts.Shots || result.Classical.OptimalSampleFraction < 0 || result.Classical.OptimalSampleFraction > 1 {
				t.Fatalf("result %+v", result)
			}
			if tc.name == "one" && (result.BestExpectedEnergy > 1e-4 || result.Classical.MinimumEnergy != 0 || result.Final.BestObserved.Bitstring != "0") {
				t.Fatalf("one-variable result %+v", result)
			}
			again, err := qaoa.Optimize(ctx, p, opts)
			must(t, err)
			if !reflect.DeepEqual(result, again) {
				t.Fatalf("different seeded results")
			}
			if tc.mode == qaoa.Sampled {
				objective, err := qaoa.EvaluateSampled(ctx, p, result.BestParameters, opts.Shots, opts.Seed)
				must(t, err)
				if result.BestExpectedEnergy != objective.ExpectedEnergy {
					t.Fatal("search objective disagrees with sampled expectation")
				}
			}
		})
	}
}

type fakeEvaluator struct{}

func (fakeEvaluator) Evaluate(ctx context.Context, p qaoa.Parameters) (qaoa.Evaluation, error) {
	value, err := quadratic(ctx, p)
	return qaoa.Evaluation{ExpectedEnergy: value, BestObserved: &qaoa.Candidate{Energy: -1e9}}, err
}

func TestOptimizerUsesExpectationAndOptionalIsingReference(t *testing.T) {
	ctx := context.Background()
	p := model(t, 2, 5, []ising.Field{{Index: 0, Coefficient: -.5}}, nil)
	opts := qaoa.OptimizeOptions{Evaluator: fakeEvaluator{}, Optimizer: qaoa.CoordinateSearch{},
		InitialParameters: params(1, 1), Reference: true, ReferenceLimit: 1, Shots: 10}
	notCalculated, err := qaoa.Optimize(ctx, p, opts)
	must(t, err)
	if notCalculated.BestExpectedEnergy < 0 || notCalculated.BestExpectedEnergy > 1e-7 ||
		notCalculated.Classical != nil || !strings.Contains(notCalculated.ReferenceStatus, "above") {
		t.Fatalf("result %+v", notCalculated)
	}
	opts.ReferenceLimit = 2
	calculated, err := qaoa.Optimize(ctx, p, opts)
	must(t, err)
	if calculated.Classical == nil || calculated.Classical.MinimumEnergy != 4.5 {
		t.Fatalf("result %+v", calculated)
	}
}
