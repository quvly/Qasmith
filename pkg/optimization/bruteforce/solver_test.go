package bruteforce_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/quvly/Qasmith/pkg/optimization/bruteforce"
	"github.com/quvly/Qasmith/pkg/optimization/qubo"
)

func TestSolverMinimumAndTie(t *testing.T) {
	for _, tc := range []struct {
		matrix         [][]float64
		offset, energy float64
		want           []bool
	}{
		{[][]float64{{-2, 4}, {0, 3}}, 1, -1, []bool{true, false}},
		{[][]float64{{-1, 2}, {0, -1}}, 0, -1, []bool{false, true}},
		{[][]float64{{0, 0}, {0, 0}}, -3, -3, []bool{false, false}},
	} {
		m, err := qubo.FromMatrix(tc.matrix, tc.offset)
		if err != nil {
			t.Fatal(err)
		}
		result, err := bruteforce.Solve(m)
		if err != nil || result.Energy != tc.energy || !reflect.DeepEqual(result.Assignment, tc.want) {
			t.Fatalf("result %+v, %v", result, err)
		}
		if got, err := m.Evaluate(result.Assignment); err != nil || got != result.Energy {
			t.Fatalf("energy %g, %v", got, err)
		}
	}
}

func TestSolverBoundsAndCancellation(t *testing.T) {
	b, err := qubo.New(bruteforce.DefaultMaxVariables + 1)
	if err != nil {
		t.Fatal(err)
	}
	large, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bruteforce.Solve(large); err == nil {
		t.Fatal("default bound ignored")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := (bruteforce.Solver{MaxVariables: 21}).Solve(ctx, large); !errors.Is(err, context.Canceled) {
		t.Fatalf("passed configured bound but expected cancellation, got %v", err)
	}
	b, err = qubo.New(2)
	if err != nil {
		t.Fatal(err)
	}
	small, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}
	for _, limit := range []int{-1, 1, bruteforce.MaxSupportedVariables + 1} {
		if _, err := (bruteforce.Solver{MaxVariables: limit}).Solve(context.Background(), small); err == nil {
			t.Fatalf("accepted limit %d", limit)
		}
	}
}
