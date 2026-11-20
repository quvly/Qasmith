// Package qasmith provides the compiler pipeline and public extension entry points.
package qasmith

import (
	"context"
	"github.com/quvly/Qasmith/internal/diagnostic"
	"github.com/quvly/Qasmith/internal/limits"
	"github.com/quvly/Qasmith/internal/lower"
	"github.com/quvly/Qasmith/internal/modules"
	"github.com/quvly/Qasmith/internal/openqasm"
	"github.com/quvly/Qasmith/internal/semantic"
	"github.com/quvly/Qasmith/internal/simulator"
	"github.com/quvly/Qasmith/internal/stdlib"
	"github.com/quvly/Qasmith/internal/syntax"
	"github.com/quvly/Qasmith/pkg/extension"
	"github.com/quvly/Qasmith/pkg/ir"
	"slices"
)

type Diagnostic = diagnostic.Error
type Limits = limits.Limits

// DefaultLimits returns the limits used by a zero-value Compiler.
func DefaultLimits() Limits { return limits.Default() }

// Compiler uses the default OpenQASM backend when Backend is nil.
// Passes run in slice order. The zero value runs no optimization passes.
type Compiler struct {
	Registry *extension.Registry
	Backend  ir.Backend
	Passes   []ir.Pass
	Limits   Limits
}

func (c Compiler) analyze(path string, resolved Limits) (*semantic.Program, error) {
	graph, registry, err := c.load(path)
	if err != nil {
		return nil, err
	}
	return semantic.Analyze(graph, registry, resolved)
}

func (c Compiler) load(path string) (*modules.Graph, *extension.Registry, error) {
	registry := c.Registry.Clone()
	if err := stdlib.Register(registry); err != nil {
		return nil, nil, err
	}
	graph, err := modules.LoadWithRegistry(path, registry.HasModule)
	return graph, registry, err
}

// Check validates the module graph and resolves static constructs without executing host callbacks.
func (c Compiler) Check(path string) error {
	resolved, err := c.Limits.Resolve()
	if err != nil {
		return err
	}
	program, err := c.analyze(path, resolved)
	if err != nil || !program.Static && c.Limits == (Limits{}) {
		return err
	}
	_, err = lower.Execute(program, resolved, true)
	return err
}

// Dependencies visits resolved source modules. A repeated module is reported but not expanded.
func (c Compiler) Dependencies(path string, visit func(path, prefix string, repeated bool)) error {
	resolved, err := c.Limits.Resolve()
	if err != nil {
		return err
	}
	graph, registry, err := c.load(path)
	if err != nil {
		return err
	}
	if _, err := semantic.Analyze(graph, registry, resolved); err != nil {
		return err
	}
	graph.Walk(visit)
	return nil
}

func require(program *semantic.Program, expected extension.Type) error {
	if program.Entry.Signature.Return != expected {
		return diagnostic.At(program.Entry.Declaration.At, "E4002", "this command requires main returning %s", expected)
	}
	return nil
}

func (c Compiler) Eval(path string) (int64, error) {
	resolved, err := c.Limits.Resolve()
	if err != nil {
		return 0, err
	}
	program, err := c.analyze(path, resolved)
	if err != nil {
		return 0, err
	}
	if err := require(program, extension.Int); err != nil {
		return 0, err
	}
	result, err := lower.Execute(program, resolved, false)
	return result.Value.Integer, err
}

// Prepared contains validated circuit IR and any immutable terminal recipe.
type Prepared struct {
	Circuit  *ir.Program
	Plan     extension.CircuitPlan
	Origin   ir.Origin
	compiler Compiler
	limits   Limits
	entry    syntax.Pos
}

func (c Compiler) Lower(path string) (*ir.Program, error) {
	p, err := c.Prepare(path)
	if err != nil {
		return nil, err
	}
	return p.Circuit, nil
}

// Prepare type-checks and lowers the entry without simulation or parameter search.
func (c Compiler) Prepare(path string) (Prepared, error) {
	resolved, err := c.Limits.Resolve()
	if err != nil {
		return Prepared{}, err
	}
	program, err := c.analyze(path, resolved)
	if err != nil {
		return Prepared{}, err
	}
	if err := require(program, extension.MeasuredResult); err != nil {
		return Prepared{}, err
	}
	result, err := lower.Execute(program, resolved, false)
	if err != nil {
		return Prepared{}, err
	}
	c.Passes = slices.Clone(c.Passes)
	p := Prepared{Circuit: result.Circuit, Plan: result.Plan, Origin: result.Origin, compiler: c, limits: resolved, entry: program.Entry.Declaration.At}
	if err := p.validatePasses(); err != nil {
		return Prepared{}, err
	}
	return p, nil
}

func (p Prepared) validatePasses() error {
	circuit := p.Circuit
	layout, err := circuit.Validate()
	if err != nil {
		return diagnostic.At(p.entry, "E4003", "invalid lowered circuit: %v", err)
	}
	var order []ir.QubitID
	if p.Plan != nil {
		order = measurementOrder(circuit)
	}
	for _, pass := range p.compiler.Passes {
		if pass == nil {
			return diagnostic.At(p.entry, "E4003", "nil compiler pass")
		}
		if err := pass.Run(circuit); err != nil {
			return diagnostic.At(p.entry, "E4003", "pass %s: %v", pass.Name(), err)
		}
		next, err := circuit.Validate()
		if err != nil {
			return diagnostic.At(p.entry, "E4003", "pass %s emitted invalid circuit: %v", pass.Name(), err)
		}
		if next.Qubits > p.limits.MaxQubits || len(circuit.Operations)-next.Qubits-next.Bits > p.limits.MaxGates {
			return diagnostic.At(p.entry, "E4100", "pass %s exceeded the configured qubit or gate limit", pass.Name())
		}
		if p.Plan != nil && (next != layout || !slices.Equal(order, measurementOrder(circuit))) {
			return diagnostic.At(p.entry, "E4003", "pass %s changed the terminal plan's register layout or measurement order", pass.Name())
		}
	}
	return nil
}

func measurementOrder(circuit *ir.Program) []ir.QubitID {
	var order []ir.QubitID
	for _, op := range circuit.Operations {
		if m, ok := op.(ir.Measure); ok {
			order = append(order, m.Qubit)
		}
	}
	return order
}

func (c Compiler) Compile(path string) ([]byte, error) {
	prepared, err := c.Prepare(path)
	if err != nil {
		return nil, err
	}
	return prepared.Export()
}

// Export runs the configured backend on the prepared circuit.
func (p Prepared) Export() ([]byte, error) {
	backend := p.compiler.Backend
	if backend == nil {
		backend = openqasm.Backend{}
	}
	data, err := backend.Compile(p.Circuit)
	if err != nil {
		return nil, diagnostic.Wrap(p.entry, "E5000", err, "backend %s", backend.Name())
	}
	return data, nil
}

type Simulation = simulator.Result

func (c Compiler) Run(path string, shots int, seed int64) (Simulation, error) {
	prepared, err := c.Prepare(path)
	if err != nil {
		return Simulation{}, err
	}
	result, err := prepared.Execute(context.Background(), shots, seed)
	return result.Raw, err
}
