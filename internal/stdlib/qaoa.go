package stdlib

import (
	"fmt"

	"github.com/quvly/Qasmith/pkg/extension"
	"github.com/quvly/Qasmith/pkg/optimization/plan"
	"github.com/quvly/Qasmith/pkg/optimization/qaoa"
)

func qaoaFunction() extension.Function {
	return extension.Function{Name: "qaoa", Documentation: "Builds a measured QAOA circuit at the supplied layer parameters.", ParameterNames: []string{"problem", "gammas", "betas"}, Parameters: []extension.Parameter{{Type: plan.OptimizationProblem().Name()}, {Rank: 1}, {Rank: 1}},
		Return: extension.MeasuredResult, CheckTypes: func(l extension.StaticLimits, types []extension.Type) error {
			_, gammas, _ := types[1].Array()
			_, betas, _ := types[2].Array()
			if gammas < 1 || gammas != betas {
				return fmt.Errorf("QAOA gamma and beta arrays must have equal, nonzero lengths")
			}
			if gammas > l.MaxCircuitLayers {
				return fmt.Errorf("circuit layers limit exceeded (maximum %d)", l.MaxCircuitLayers)
			}
			return nil
		}, Produce: produceQAOA}
}

func produceQAOA(ctx extension.StaticContext, args []extension.Value) (extension.CircuitPlan, error) {
	model, err := plan.OptimizationProblem().Unpack(args[0])
	if err != nil {
		return nil, err
	}
	n := len(args[1].Array.Elements)
	if n > ctx.Limits().MaxCircuitLayers {
		return nil, fmt.Errorf("circuit layers limit exceeded (maximum %d)", ctx.Limits().MaxCircuitLayers)
	}
	params := qaoa.Parameters{Gammas: make([]float64, n), Betas: make([]float64, n)}
	for i := range n {
		params.Gammas[i], err = numeric(args[1].Array.Elements[i])
		if err != nil {
			return nil, fmt.Errorf("gamma[%d]: %w", i, err)
		}
		params.Betas[i], err = numeric(args[2].Array.Elements[i])
		if err != nil {
			return nil, fmt.Errorf("beta[%d]: %w", i, err)
		}
	}
	return plan.New(model, params, ctx)
}
