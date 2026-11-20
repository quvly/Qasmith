package lower

import (
	"fmt"
	"slices"

	"github.com/quvly/Qasmith/internal/diagnostic"
	"github.com/quvly/Qasmith/internal/syntax"
	"github.com/quvly/Qasmith/pkg/extension"
	"github.com/quvly/Qasmith/pkg/ir"
)

type hostContext struct {
	origin     ir.Origin
	allowed    map[ir.QubitID]bool
	pending    []ir.Operation
	err        error
	active     bool
	gateBudget int
	maxGates   int
	at         syntax.Pos
}

func (c *hostContext) ApplyGate(gate ir.Gate, ids ...ir.QubitID) error {
	return c.ApplyParameterizedGate(gate, nil, ids...)
}

func (c *hostContext) ApplyParameterizedGate(gate ir.Gate, parameters []float64, ids ...ir.QubitID) error {
	if !c.active {
		return fmt.Errorf("lowering context expired")
	}
	if c.err != nil {
		return c.err
	}
	op := ir.ApplyGate{Gate: gate, Qubits: ids, Parameters: parameters, Origin: c.origin}
	if err := op.ValidateOperands(); err != nil {
		c.err = err
		return err
	}
	for _, id := range ids {
		if !c.allowed[id] {
			c.err = fmt.Errorf("gate references qubit q%d outside intrinsic arguments", id)
			return c.err
		}
	}
	if len(c.pending) >= c.gateBudget {
		c.err = limit(c.at, "generated gates", c.maxGates)
		return c.err
	}
	op.Qubits = slices.Clone(ids)
	op.Parameters = slices.Clone(parameters)
	c.pending = append(c.pending, op)
	return nil
}

func (f *frame) host(i *extension.Intrinsic, args []Value, origin ir.Origin, at syntax.Pos) (Value, error) {
	fail := func(err error) (Value, error) {
		if detail, ok := err.(*diagnostic.Error); ok && detail.Code == "E4100" {
			return Value{}, err
		}
		return Value{}, diagnostic.Wrap(at, "E4001", err, "host:%s/%s", i.Namespace, i.Name)
	}
	copyArgs := func() ([]Value, error) {
		copy := make([]Value, len(args))
		for index, arg := range args {
			v, err := f.copyHostValue(arg, arg.Type, at)
			if err != nil {
				return nil, err
			}
			copy[index] = v
		}
		return copy, nil
	}
	if i.Validate != nil {
		validateArgs, err := copyArgs()
		if err != nil {
			return Value{}, err
		}
		if err := i.Validate(validateArgs); err != nil {
			return fail(err)
		}
	}
	lowerArgs, err := copyArgs()
	if err != nil {
		return Value{}, err
	}
	context := &hostContext{origin: origin, allowed: make(map[ir.QubitID]bool), active: true,
		gateBudget: f.limits.MaxGates - f.gates, maxGates: f.limits.MaxGates, at: at}
	for _, arg := range args {
		if arg.Type == extension.Qubit {
			context.allowed[arg.Qubit] = true
		}
	}
	v, err := i.Lower(context, lowerArgs)
	context.active = false
	if context.err != nil {
		return fail(context.err)
	}
	if err != nil {
		return fail(err)
	}
	v, err = f.copyHostValue(v, i.Signature.Return, at)
	if err != nil {
		return fail(err)
	}
	f.gates += len(context.pending)
	f.circuit.Operations = append(f.circuit.Operations, context.pending...)
	return v, nil
}

// Copy arrays at the host boundary so callbacks cannot mutate Smith values.
func (f *frame) copyHostValue(v Value, want extension.Type, at syntax.Pos) (Value, error) {
	if v.Type != want {
		return Value{}, fmt.Errorf("expected result %s, got %s", want, v.Type)
	}
	if v.Type == extension.Float && !finite(v.Float) {
		return Value{}, fmt.Errorf("non-finite Float result")
	}
	if opaque := f.program.Registry.LookupType(want); opaque != nil {
		if !opaque.Accepts(v) {
			return Value{}, fmt.Errorf("invalid or foreign %s value", want)
		}
		return v, nil
	}
	element, n, array := want.Array()
	if !array {
		v.Array = nil
		return v, nil
	}
	if v.Array == nil || len(v.Array.Elements) != n {
		return Value{}, fmt.Errorf("expected %s array with %d elements", want, n)
	}
	if err := f.enter(at); err != nil {
		return Value{}, err
	}
	defer func() { f.executor.depth-- }()
	if err := f.reserveElements(at, n); err != nil {
		return Value{}, err
	}
	out := Value{Type: want, Array: &extension.ArrayValue{Elements: make([]Value, n)}}
	for index, child := range v.Array.Elements {
		copy, err := f.copyHostValue(child, element, at)
		if err != nil {
			return Value{}, err
		}
		out.Array.Elements[index] = copy
	}
	return out, nil
}
