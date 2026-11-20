// Package stdlib registers compiler-distributed trusted modules.
package stdlib

import (
	"fmt"

	"github.com/quvly/Qasmith/pkg/extension"
	"github.com/quvly/Qasmith/pkg/optimization/plan"
)

func Register(registry *extension.Registry) error {
	t := plan.OptimizationProblem()
	if current := registry.LookupType(t.Name()); current != t.Descriptor() {
		if err := registry.RegisterType(t.Descriptor()); err != nil {
			return err
		}
	}
	for _, module := range []struct {
		path      string
		functions []extension.Function
	}{
		{"std/optimization/tsp", []extension.Function{tspFunction()}},
		{"std/optimization/qaoa", []extension.Function{qaoaFunction()}},
		{"std/optimization/qubo", quboFunctions()},
	} {
		if err := registry.RegisterModule(module.path, module.functions...); err != nil {
			return fmt.Errorf("standard library: %w", err)
		}
	}
	return nil
}
