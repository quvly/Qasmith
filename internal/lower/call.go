package lower

import (
	"fmt"

	"github.com/quvly/Qasmith/internal/diagnostic"
	"github.com/quvly/Qasmith/internal/semantic"
	"github.com/quvly/Qasmith/internal/syntax"
	"github.com/quvly/Qasmith/pkg/extension"
	"github.com/quvly/Qasmith/pkg/ir"
)

func (f *frame) call(expr *syntax.CallExpression) (Value, error) {
	target := f.program.Targets[expr]
	args := make([]Value, len(expr.Args))
	for i, arg := range expr.Args {
		value, err := f.expr(arg)
		if err != nil {
			return Value{}, err
		}
		if i < len(target.Signature.Parameters) {
			param := target.Signature.Parameters[i]
			if n, reg := semantic.RegisterSize(param); reg && value.Type != "" && value.Type != param {
				return Value{}, diagnostic.At(arg.Position(), "E3004", "argument %d to %s expects %s, got %s (expected %d qubits)", i+1, expr.Name, param, value.Type, n)
			}
			value = convert(value, param)
		}
		args[i] = value
	}
	for _, pair := range target.Signature.Distinct {
		a, b := args[pair[0]], args[pair[1]]
		if a.Type != "" && b.Type != "" && a.Qubit == b.Qubit {
			return Value{}, diagnostic.At(expr.At, "E3100", "call to %s requires distinct qubit operands (parameters %d and %d)", expr.Name, pair[0]+1, pair[1]+1)
		}
	}

	if target.Builtin == "len" {
		typ := args[0].Type
		if typ == "" {
			typ = f.program.Types[expr.Args[0]]
		}
		n, reg := semantic.RegisterSize(typ)
		if !reg {
			_, n, _ = typ.Array()
		}
		if n == 0 {
			return Value{}, nil
		}
		return Value{Type: extension.Int, Integer: int64(n)}, nil
	}
	if f.check && (target.Intrinsic != nil || target.Trusted != nil) {
		return Value{}, nil
	}
	if target.Declaration != nil {
		f.sites = append(f.sites, ir.Location(expr.At))
		defer func() { f.executor.sites = f.sites[:len(f.sites)-1] }()
		value, err := f.invoke(target, args)
		if err != nil {
			return Value{}, fmt.Errorf("%w\n  called from %s", err, expr.At)
		}
		return value, nil
	}

	origin := ir.Origin{Location: ir.Location(expr.At)}
	for i := len(f.sites) - 1; i >= 0; i-- {
		origin.CallSites = append(origin.CallSites, f.sites[i])
	}
	switch target.Builtin {
	case "qubit":
		return f.allocate(1, false, origin, expr.At)
	case "qreg":
		if args[0].Type == "" {
			return Value{}, nil
		}
		return f.allocate(args[0].Integer, true, origin, expr.Args[0].Position())
	case "measure":
		return f.measure(args, origin, expr.At)
	}
	if target.Gate != "" {
		if f.gates == f.limits.MaxGates {
			return Value{}, limit(expr.At, "generated gates", f.limits.MaxGates)
		}
		f.gates++
		if f.check {
			return Value{Type: extension.Void}, nil
		}
		op := ir.ApplyGate{Gate: target.Gate, Origin: origin}
		for _, arg := range args {
			if arg.Type == extension.Qubit {
				op.Qubits = append(op.Qubits, arg.Qubit)
			} else {
				op.Parameters = append(op.Parameters, arg.Float)
			}
		}
		f.emit(op)
		return Value{Type: extension.Void}, nil
	}
	if target.Trusted != nil {
		return f.trusted(target, args, origin, expr.At)
	}
	return f.host(target.Intrinsic, args, origin, expr.At)
}

func (f *frame) allocate(n int64, register bool, origin ir.Origin, at syntax.Pos) (Value, error) {
	if n <= 0 {
		return Value{}, diagnostic.At(at, "E4000", "register size must be a positive Int")
	}
	if n > int64(f.limits.MaxQubits-int(f.nextQubit)) {
		return Value{}, limit(at, "allocated qubits", f.limits.MaxQubits)
	}
	base := f.nextQubit
	for range n {
		f.emit(ir.AllocateQubit{Qubit: f.nextQubit, Origin: origin})
		f.nextQubit++
	}
	typ := extension.Qubit
	if register {
		typ = semantic.RegisterType(int(n))
	}
	return Value{Type: typ, Qubit: base}, nil
}

func (f *frame) measure(args []Value, origin ir.Origin, at syntax.Pos) (Value, error) {
	seen := make(map[ir.QubitID]bool)
	bit := ir.BitID(0)
	for _, arg := range args {
		if arg.Type == "" {
			continue
		}
		n, reg := semantic.RegisterSize(arg.Type)
		if !reg {
			n = 1
		}
		for i := range n {
			id := arg.Qubit + ir.QubitID(i)
			if seen[id] {
				return Value{}, diagnostic.At(at, "E3100", "measure requires distinct qubit operands")
			}
			seen[id] = true
			f.emit(ir.Measure{Qubit: id, Bit: bit, Origin: origin})
			bit++
		}
	}
	return Value{Type: extension.MeasuredResult}, nil
}

func (e *executor) emit(op ir.Operation) {
	if !e.check {
		e.circuit.Operations = append(e.circuit.Operations, op)
	}
}
