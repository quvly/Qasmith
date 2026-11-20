package lower

import (
	"fmt"
	"slices"

	"github.com/quvly/Qasmith/internal/diagnostic"
	"github.com/quvly/Qasmith/internal/limits"
	"github.com/quvly/Qasmith/internal/semantic"
	"github.com/quvly/Qasmith/internal/syntax"
	"github.com/quvly/Qasmith/pkg/extension"
	"github.com/quvly/Qasmith/pkg/ir"
)

type staticContext struct {
	executor *executor
	budgets  limits.Limits
	at       syntax.Pos
	err      error
}

func (c *staticContext) Limits() limits.Limits { return c.budgets }
func (c *staticContext) Spend(units int) error {
	if c.executor == nil {
		return fmt.Errorf("static context expired")
	}
	if c.err != nil {
		return c.err
	}
	if units < 0 {
		c.err = fmt.Errorf("invalid negative structured work charge")
		return c.err
	}
	e := c.executor
	if units > e.limits.MaxStructuredWork-e.structuredWork {
		c.err = limit(c.at, "structured evaluation work", e.limits.MaxStructuredWork)
	} else if units > e.limits.MaxSteps-e.steps {
		c.err = limit(c.at, "static evaluation steps", e.limits.MaxSteps)
	} else {
		e.structuredWork += units
		e.steps += units
	}
	return c.err
}

func (f *frame) trusted(target *semantic.Function, args []Value, origin ir.Origin, at syntax.Pos) (Value, error) {
	fn := target.Trusted
	fail := func(err error) (Value, error) {
		if detail, ok := err.(*diagnostic.Error); ok {
			return Value{}, detail
		}
		return Value{}, diagnostic.Wrap(at, "E4001", err, "%s/%s", target.Namespace, fn.Name)
	}
	copyArgs := func() ([]Value, error) {
		values := make([]Value, len(args))
		for i, arg := range args {
			if !fn.Parameters[i].Accepts(arg.Type) {
				return nil, fmt.Errorf("argument %d violates trusted signature", i+1)
			}
			value, err := f.copyHostValue(arg, arg.Type, at)
			if err != nil {
				return nil, err
			}
			values[i] = value
		}
		return values, nil
	}
	ctx := &staticContext{executor: f.executor, budgets: f.limits, at: at}
	defer func() { ctx.executor = nil }()
	if fn.Validate != nil {
		input, err := copyArgs()
		if err != nil {
			return fail(err)
		}
		if err := fn.Validate(ctx, input); err != nil {
			return fail(err)
		}
		if ctx.err != nil {
			return fail(ctx.err)
		}
	}
	input, err := copyArgs()
	if err != nil {
		return fail(err)
	}
	if fn.Produce != nil {
		if f.plan != nil || len(f.circuit.Operations) != 0 {
			return fail(fmt.Errorf("terminal circuit producer requires an empty circuit"))
		}
		plan, err := fn.Produce(ctx, input)
		if err != nil {
			return fail(err)
		}
		if ctx.err != nil {
			return fail(ctx.err)
		}
		if plan == nil {
			return fail(fmt.Errorf("terminal producer returned no circuit plan"))
		}
		circuit, err := plan.Build()
		if err != nil {
			return fail(err)
		}
		circuit, err = PrepareProduced(circuit, origin, f.limits)
		if err != nil {
			return fail(err)
		}
		f.plan, f.origin, f.circuit = plan, origin, circuit
		return Value{Type: extension.MeasuredResult}, nil
	}
	v, err := fn.Evaluate(ctx, input)
	if err != nil {
		return fail(err)
	}
	if ctx.err != nil {
		return fail(ctx.err)
	}
	v, err = f.copyHostValue(v, fn.Return, at)
	if err != nil {
		return fail(err)
	}
	if desc := f.program.Registry.LookupType(fn.Return); desc != nil {
		if err := desc.ValidateResult(ctx, v); err != nil {
			return fail(err)
		}
		if ctx.err != nil {
			return fail(ctx.err)
		}
	}
	return v, nil
}

// PrepareProduced validates a terminal circuit and attaches the call location.
func PrepareProduced(p *ir.Program, origin ir.Origin, l limits.Limits) (*ir.Program, error) {
	layout, err := p.Validate()
	if err != nil {
		return nil, err
	}
	if layout.Qubits > l.MaxQubits || len(p.Operations)-layout.Qubits-layout.Bits > l.MaxGates {
		return nil, fmt.Errorf("terminal circuit exceeded the configured qubit or gate limit")
	}
	if layout.Bits == 0 {
		return nil, fmt.Errorf("terminal circuit must include measured qubits")
	}
	out := &ir.Program{Operations: make([]ir.Operation, 0, len(p.Operations))}
	for _, op := range p.Operations {
		switch op := op.(type) {
		case ir.AllocateQubit:
			op.Origin = producedOrigin(op.Origin, origin)
			out.Operations = append(out.Operations, op)
		case ir.ApplyGate:
			op.Origin = producedOrigin(op.Origin, origin)
			op.Qubits, op.Parameters = slices.Clone(op.Qubits), slices.Clone(op.Parameters)
			out.Operations = append(out.Operations, op)
		case ir.Measure:
			op.Origin = producedOrigin(op.Origin, origin)
			out.Operations = append(out.Operations, op)
		}
	}
	return out, nil
}

func producedOrigin(old, call ir.Origin) ir.Origin {
	if old.File == "" {
		call.CallSites = slices.Clone(call.CallSites)
		return call
	}
	old.CallSites = append(slices.Clone(old.CallSites), call.Location)
	old.CallSites = append(old.CallSites, call.CallSites...)
	return old
}
