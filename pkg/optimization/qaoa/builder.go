// Package qaoa builds and optimizes generic Ising QAOA circuits.
package qaoa

import (
	"fmt"
	"math"
	"slices"

	"github.com/quvly/Qasmith/internal/limits"
	"github.com/quvly/Qasmith/internal/openqasm"
	"github.com/quvly/Qasmith/pkg/ir"
	"github.com/quvly/Qasmith/pkg/optimization/internal/energy"
	"github.com/quvly/Qasmith/pkg/optimization/ising"
	"github.com/quvly/Qasmith/pkg/optimization/qubo"
)

// Parameters holds one gamma and one beta per layer, in execution order.
type Parameters struct {
	Gammas []float64
	Betas  []float64
}

// Validate checks that both parameter lists contain layers finite values.
func (p Parameters) Validate(layers int) error {
	if layers < 1 || len(p.Gammas) != layers || len(p.Betas) != layers {
		return fmt.Errorf("expected a positive layer count and exactly %d gammas and betas (got %d and %d)", layers, len(p.Gammas), len(p.Betas))
	}
	for i := range p.Gammas {
		if !energy.Finite(p.Gammas[i]) || !energy.Finite(p.Betas[i]) {
			return fmt.Errorf("layer %d parameters must be finite", i)
		}
	}
	return nil
}

func (p Parameters) clone() Parameters {
	return Parameters{Gammas: slices.Clone(p.Gammas), Betas: slices.Clone(p.Betas)}
}

// Problem holds a validated Ising objective and the optional original QUBO.
// It can be reused across builds, evaluations, and optimization runs.
type Problem struct {
	model    ising.Model
	fields   []ising.Field
	pairs    []ising.Coupling
	original *qubo.Model
}

// New prepares an Ising problem.
func New(model ising.Model) (Problem, error) {
	if err := model.Validate(); err != nil {
		return Problem{}, err
	}
	return Problem{model: model, fields: model.Fields(), pairs: model.Couplings()}, nil
}

// FromQUBO converts once and retains the original for classical comparison.
func FromQUBO(model qubo.Model) (Problem, error) {
	converted, err := model.ToIsing()
	if err != nil {
		return Problem{}, err
	}
	p, err := New(converted)
	if err == nil {
		p.original = &model
	}
	return p, err
}

// Ising returns the immutable Hamiltonian, including its offset.
func (p Problem) Ising() ising.Model { return p.model }

// OriginalQUBO returns the immutable original model when prepared from QUBO.
func (p Problem) OriginalQUBO() (qubo.Model, bool) {
	if p.original == nil {
		return qubo.Model{}, false
	}
	return *p.original, true
}

// BuildOptions controls terminal measurements, circuit budgets, and optional passes.
// Zero budgets select the same defaults as the Smith compiler.
type BuildOptions struct {
	Measure   bool
	MaxQubits int
	MaxGates  int
	Passes    []ir.Pass
}

// Build constructs an unmeasured circuit with default limits.
func Build(model ising.Model, params Parameters) (*ir.Program, error) {
	problem, err := New(model)
	if err != nil {
		return nil, err
	}
	return problem.Build(params, BuildOptions{})
}

func (p Problem) gateBudget(layers int, opts BuildOptions) (int, int, error) {
	defaults := limits.Default()
	maxQubits, maxGates := opts.MaxQubits, opts.MaxGates
	if maxQubits == 0 {
		maxQubits = defaults.MaxQubits
	}
	if maxGates == 0 {
		maxGates = defaults.MaxGates
	}
	if maxQubits < 1 || maxGates < 1 {
		return 0, 0, fmt.Errorf("qubit and gate limits must be positive")
	}
	n := p.model.Variables()
	if n < 1 {
		return 0, 0, fmt.Errorf("invalid QAOA problem")
	}
	if n > maxQubits {
		return 0, 0, fmt.Errorf("allocated qubits limit exceeded (maximum %d)", maxQubits)
	}
	if layers < 1 {
		return 0, 0, fmt.Errorf("QAOA layer count must be positive")
	}
	if n > maxGates {
		return 0, 0, fmt.Errorf("generated gates limit exceeded (maximum %d)", maxGates)
	}
	perLayer := (maxGates - n) / layers
	if n > perLayer || len(p.fields) > perLayer-n || len(p.pairs) > (perLayer-n-len(p.fields))/3 {
		return 0, 0, fmt.Errorf("generated gates limit exceeded (maximum %d)", maxGates)
	}
	return n + layers*(n+len(p.fields)+3*len(p.pairs)), maxGates, nil
}

// Build generates validated IR for the uniform initial state followed by
// cost and X mixer layers. It optionally appends terminal measurements.
func (p Problem) Build(params Parameters, opts BuildOptions) (*ir.Program, error) {
	gates, budget, err := p.gateBudget(len(params.Gammas), opts)
	if err != nil {
		return nil, err
	}
	if err := params.Validate(len(params.Gammas)); err != nil {
		return nil, err
	}
	n := p.model.Variables()
	maxInt := int(^uint(0) >> 1)
	bits := 0
	if opts.Measure {
		bits = n
	}
	if gates > maxInt-n || bits > maxInt-n-gates {
		return nil, fmt.Errorf("circuit operation count overflow")
	}
	ops := make([]ir.Operation, 0, n+gates+bits)
	for i := range n {
		ops = append(ops, ir.AllocateQubit{Qubit: ir.QubitID(i)})
	}
	for i := range n {
		ops = append(ops, gate(ir.H, i))
	}
	for layer, gamma := range params.Gammas {
		ops, err = p.appendCost(ops, gamma)
		if err != nil {
			return nil, fmt.Errorf("cost layer %d: %w", layer, err)
		}
		mix := 2 * params.Betas[layer]
		if !energy.Finite(mix) {
			return nil, fmt.Errorf("mixer layer %d: non-finite RX angle", layer)
		}
		for i := range n {
			ops = append(ops, rotation(ir.RX, i, mix))
		}
	}
	for i := range bits {
		ops = append(ops, ir.Measure{Qubit: ir.QubitID(i), Bit: ir.BitID(i)})
	}
	circuit := &ir.Program{Operations: ops}
	if err := validateLayout(circuit, n, bits, budget); err != nil {
		return nil, err
	}
	for _, pass := range opts.Passes {
		if pass == nil {
			return nil, fmt.Errorf("nil QAOA circuit pass")
		}
		if err := pass.Run(circuit); err != nil {
			return nil, fmt.Errorf("pass %s: %w", pass.Name(), err)
		}
		if err := validateLayout(circuit, n, bits, budget); err != nil {
			return nil, fmt.Errorf("pass %s: %w", pass.Name(), err)
		}
	}
	return circuit, nil
}

func validateLayout(program *ir.Program, n, bits, budget int) error {
	layout, err := program.Validate()
	if err != nil {
		return err
	}
	if layout.Qubits != n || layout.Bits != bits {
		return fmt.Errorf("circuit changed the QAOA register layout")
	}
	if len(program.Operations)-n-bits > budget {
		return fmt.Errorf("generated gates limit exceeded (maximum %d)", budget)
	}
	for _, op := range program.Operations {
		if m, ok := op.(ir.Measure); ok && int(m.Bit) != int(m.Qubit) {
			return fmt.Errorf("measurement changed QAOA variable order")
		}
	}
	return nil
}

func (p Problem) appendCost(ops []ir.Operation, gamma float64) ([]ir.Operation, error) {
	for _, field := range p.fields {
		angle := costAngle(gamma, field.Coefficient)
		if !energy.Finite(angle) {
			return nil, fmt.Errorf("non-finite field angle at qubit %d", field.Index)
		}
		ops = append(ops, rotation(ir.RZ, field.Index, angle))
	}
	for _, pair := range p.pairs {
		angle := costAngle(gamma, pair.Coefficient)
		if !energy.Finite(angle) {
			return nil, fmt.Errorf("non-finite coupling angle at (%d, %d)", pair.I, pair.J)
		}
		ops = append(ops, gate(ir.CX, pair.I, pair.J), rotation(ir.RZ, pair.J, angle), gate(ir.CX, pair.I, pair.J))
	}
	return ops, nil
}

func costAngle(gamma, coefficient float64) float64 {
	// Scale mantissas first so neither doubling nor a premature underflow
	// changes the representable product.
	g, gexp := math.Frexp(gamma)
	c, cexp := math.Frexp(coefficient)
	return math.Ldexp(g*c, gexp+cexp+1)
}

func gate(kind ir.Gate, indices ...int) ir.ApplyGate {
	ids := make([]ir.QubitID, len(indices))
	for i, id := range indices {
		ids[i] = ir.QubitID(id)
	}
	return ir.ApplyGate{Gate: kind, Qubits: ids}
}

func rotation(kind ir.Gate, index int, angle float64) ir.ApplyGate {
	return ir.ApplyGate{Gate: kind, Qubits: []ir.QubitID{ir.QubitID(index)}, Parameters: []float64{angle}}
}

// Export builds a circuit and passes it to the selected backend. A nil backend
// selects the existing OpenQASM 3 backend.
func (p Problem) Export(params Parameters, opts BuildOptions, backend ir.Backend) ([]byte, error) {
	circuit, err := p.Build(params, opts)
	if err != nil {
		return nil, err
	}
	if backend == nil {
		backend = openqasm.Backend{}
	}
	return backend.Compile(circuit)
}
