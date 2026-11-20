// Command custom-module embeds Qasmith with a typed objective factory.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/quvly/Qasmith/pkg/extension"
	"github.com/quvly/Qasmith/pkg/optimization/plan"
	"github.com/quvly/Qasmith/pkg/optimization/qubo"
	"github.com/quvly/Qasmith/pkg/qasmith"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	path := "examples/optimization/language/custom-module/program.smith"
	if len(os.Args) == 2 {
		path = os.Args[1]
	}
	registry := &extension.Registry{}
	kind := plan.OptimizationProblem()
	if err := registry.RegisterType(kind.Descriptor()); err != nil {
		return err
	}
	err := registry.RegisterModule("host:custom-optimization", extension.Function{
		Name: "createProblem", Parameters: []extension.Parameter{{Type: extension.Float}}, Return: kind.Name(),
		Evaluate: func(ctx extension.StaticContext, values []extension.Value) (extension.Value, error) {
			if err := ctx.Spend(2); err != nil {
				return extension.Value{}, err
			}
			model, err := qubo.FromMatrix([][]float64{{values[0].Float}}, 2)
			if err != nil {
				return extension.Value{}, err
			}
			objective, err := plan.FromQUBO(model, nil)
			if err != nil {
				return extension.Value{}, err
			}
			return kind.Pack(objective)
		},
	})
	if err != nil {
		return err
	}
	compiler := qasmith.Compiler{Registry: registry}
	prepared, err := compiler.Prepare(path)
	if err != nil {
		return err
	}
	code, err := prepared.Export()
	if err != nil {
		return err
	}
	result, err := prepared.Execute(context.Background(), 100, 42)
	if err != nil {
		return err
	}
	_, err = fmt.Printf("%s\nExpected QUBO energy estimate: %g\nCounts: %v\n",
		code, result.Optimization.ExpectedQUBOEnergy, result.Raw.Counts)
	return err
}
