package stdlib

import (
	"context"
	"fmt"
	"math"

	"github.com/quvly/Qasmith/pkg/extension"
	"github.com/quvly/Qasmith/pkg/optimization/plan"
	"github.com/quvly/Qasmith/pkg/optimization/qaoa"
	"github.com/quvly/Qasmith/pkg/optimization/tsp"
)

func tspFunction() extension.Function {
	return extension.Function{Name: "tsp", Documentation: "Encodes a closed travelling salesman tour as an optimization problem.", ParameterNames: []string{"distances", "start"}, Parameters: []extension.Parameter{{Rank: 2}, {Type: extension.Int}},
		Return: plan.OptimizationProblem().Name(), CheckTypes: checkTSPTypes, Evaluate: evaluateTSP}
}

func checkTSPTypes(_ extension.StaticLimits, types []extension.Type) error {
	row, n, _ := types[0].Array()
	_, width, _ := row.Array()
	if n < 3 || n != width {
		return fmt.Errorf("TSP requires a square distance matrix with at least three cities")
	}
	return nil
}

func evaluateTSP(ctx extension.StaticContext, args []extension.Value) (extension.Value, error) {
	l := ctx.Limits()
	n := len(args[0].Array.Elements)
	if n > l.MaxMatrixElements/n {
		return extension.Value{}, fmt.Errorf("matrix element limit exceeded (maximum %d)", l.MaxMatrixElements)
	}
	m := n - 1
	if m > l.MaxModelVariables/m {
		return extension.Value{}, fmt.Errorf("optimization variable limit exceeded (maximum %d)", l.MaxModelVariables)
	}
	if args[1].Integer < 0 || args[1].Integer >= int64(n) {
		return extension.Value{}, fmt.Errorf("starting city %d is outside [0, %d)", args[1].Integer, n)
	}
	start := int(args[1].Integer)
	if err := ctx.Spend(n * n); err != nil {
		return extension.Value{}, err
	}
	distance := make([][]float64, n)
	nonzero := 0
	for i, row := range args[0].Array.Elements {
		distance[i] = make([]float64, n)
		for j, v := range row.Array.Elements {
			coefficient, err := numeric(v)
			if err != nil {
				return extension.Value{}, fmt.Errorf("distances[%d][%d]: %w", i, j, err)
			}
			distance[i][j] = coefficient
			if i != start && j != start && i != j && distance[i][j] > 0 {
				nonzero++
			}
		}
	}
	// Linear terms and the two exactly-one constraint families contribute m³
	// nonzero coefficients under the default penalty. Each directed edge is
	// repeated between successive positions. Reserve the offset as one coefficient.
	budget := l.MaxModelCoefficients - 1
	if m*m > budget/m {
		return extension.Value{}, fmt.Errorf("model coefficient limit exceeded (maximum %d)", l.MaxModelCoefficients)
	}
	required := m * m * m
	if nonzero > (budget-required)/(m-1) {
		return extension.Value{}, fmt.Errorf("model coefficient limit exceeded (maximum %d)", l.MaxModelCoefficients)
	}
	if err := ctx.Spend(required + nonzero*(m-1) + 1); err != nil {
		return extension.Value{}, err
	}
	problem, err := tsp.New(distance, start)
	if err != nil {
		return extension.Value{}, err
	}
	app, err := tsp.Prepare(problem, tsp.EncodeOptions{MaxVariables: l.MaxModelVariables})
	if err != nil {
		return extension.Value{}, err
	}
	model, err := plan.FromPrepared(app.QAOA(), func(ctx context.Context, raw qaoa.Evaluation) (plan.DomainResult, error) {
		result, err := app.Report(ctx, raw, tsp.ReferenceOptions{Enabled: true})
		if err != nil {
			return nil, err
		}
		return tsp.ReportMetrics{Result: result}, nil
	})
	if err != nil {
		return extension.Value{}, err
	}
	return plan.OptimizationProblem().Pack(model)
}

func numeric(v extension.Value) (float64, error) {
	if v.Type != extension.Int {
		return v.Float, nil
	}
	value := float64(v.Integer)
	if value >= math.Ldexp(1, 63) || int64(value) != v.Integer {
		return 0, fmt.Errorf("Int %d cannot be represented exactly as Float", v.Integer)
	}
	return value, nil
}
