package lower

import (
	"context"
	"fmt"
	"math"
	"math/big"

	"github.com/quvly/Qasmith/internal/diagnostic"
	"github.com/quvly/Qasmith/internal/limits"
	"github.com/quvly/Qasmith/internal/semantic"
	"github.com/quvly/Qasmith/internal/syntax"
	"github.com/quvly/Qasmith/pkg/extension"
	"github.com/quvly/Qasmith/pkg/ir"
)

type Value = extension.Value

// A zero Type marks a value unavailable during callback-free checking.
type executor struct {
	ctx                                       context.Context
	program                                   *semantic.Program
	circuit                                   *ir.Program
	nextQubit                                 ir.QubitID
	sites                                     []ir.Location
	limits                                    limits.Limits
	steps, depth, iterations, gates, elements int
	check                                     bool
	structuredWork                            int
	plan                                      extension.CircuitPlan
	origin                                    ir.Origin
}

type frame struct {
	*executor
	parent *frame
	locals map[string]Value
}

type Result struct {
	Circuit *ir.Program
	Value   Value
	Plan    extension.CircuitPlan
	Origin  ir.Origin
}

func Execute(program *semantic.Program, limits limits.Limits, check bool) (Result, error) {
	return execute(context.Background(), program, limits, check)
}

// ExecuteEditor performs callback-free static validation with cancellation.
func ExecuteEditor(ctx context.Context, program *semantic.Program, l limits.Limits) (Result, error) {
	return execute(ctx, program, l, true)
}

func execute(ctx context.Context, program *semantic.Program, limits limits.Limits, check bool) (Result, error) {
	e := &executor{ctx: ctx, program: program, circuit: &ir.Program{}, limits: limits, check: check}
	v, err := e.invoke(program.Entry, nil)
	if err != nil {
		return Result{}, err
	}
	return Result{Circuit: e.circuit, Value: v, Plan: e.plan, Origin: e.origin}, nil
}

func (e *executor) step(at syntax.Pos) error {
	if err := e.ctx.Err(); err != nil {
		return err
	}
	if e.steps >= e.limits.MaxSteps {
		return limit(at, "static evaluation steps", e.limits.MaxSteps)
	}
	e.steps++
	return nil
}

func (e *executor) enter(at syntax.Pos) error {
	if e.depth >= e.limits.MaxDepth {
		return limit(at, "static evaluation depth", e.limits.MaxDepth)
	}
	if err := e.step(at); err != nil {
		return err
	}
	e.depth++
	return nil
}

func limit(at syntax.Pos, resource string, maximum int) error {
	return diagnostic.At(at, "E4100", "%s limit exceeded (maximum %d)", resource, maximum)
}

func (e *executor) invoke(function *semantic.Function, args []Value) (Value, error) {
	if err := e.enter(function.Declaration.At); err != nil {
		return Value{}, err
	}
	defer func() { e.depth-- }()
	f := frame{executor: e, locals: make(map[string]Value, len(args))}
	for i, param := range function.Declaration.Params {
		f.locals[param.Name] = args[i]
	}
	result, returned, err := f.block(function.Declaration.Body)
	if err != nil {
		return Value{}, err
	}
	if returned {
		return convert(result, function.Signature.Return), nil
	}
	return Value{Type: extension.Void}, nil
}

func (f *frame) child() *frame {
	return &frame{executor: f.executor, parent: f, locals: make(map[string]Value)}
}

func (f *frame) block(block syntax.BlockStatement) (Value, bool, error) {
	if err := f.enter(block.At); err != nil {
		return Value{}, false, err
	}
	defer func() { f.executor.depth-- }()
	for _, stmt := range block.Statements {
		if err := f.step(stmt.Position()); err != nil {
			return Value{}, false, err
		}
		switch stmt := stmt.(type) {
		case *syntax.VariableDeclaration:
			value, err := f.expr(stmt.Value)
			if err != nil {
				return Value{}, false, err
			}
			f.locals[stmt.Name] = value
		case *syntax.ReturnStatement:
			if stmt.Value == nil {
				return Value{Type: extension.Void}, true, nil
			}
			value, err := f.expr(stmt.Value)
			return value, true, err
		case *syntax.ExpressionStatement:
			if _, err := f.expr(stmt.Value); err != nil {
				return Value{}, false, err
			}
		case *syntax.ForStatement:
			value, returned, err := f.loop(stmt)
			if err != nil || returned {
				return value, returned, err
			}
		case *syntax.IfStatement:
			value, returned, err := f.condition(stmt)
			if err != nil || returned {
				return value, returned, err
			}
		}
	}
	return Value{}, false, nil
}

func (f *frame) loop(stmt *syntax.ForStatement) (Value, bool, error) {
	first, err := f.expr(stmt.Start)
	if err != nil {
		return Value{}, false, err
	}
	last, err := f.expr(stmt.End)
	if err != nil {
		return Value{}, false, err
	}
	if first.Type == "" || last.Type == "" {
		return Value{}, mayReturn(stmt.Body), nil
	}
	if last.Integer < first.Integer {
		return Value{}, false, diagnostic.At(stmt.End.Position(), "E4000", "descending loop ranges are unsupported")
	}
	// Subtraction in unsigned space handles the whole signed Int domain.
	count := uint64(last.Integer) - uint64(first.Integer)
	if count > uint64(f.limits.MaxLoopIterations-f.iterations) {
		return Value{}, false, limit(stmt.At, "loop expansion", f.limits.MaxLoopIterations)
	}
	for i := first.Integer; i < last.Integer; i++ {
		if f.iterations == f.limits.MaxLoopIterations {
			return Value{}, false, limit(stmt.At, "loop expansion", f.limits.MaxLoopIterations)
		}
		f.iterations++
		child := f.child()
		child.locals[stmt.Name] = Value{Type: extension.Int, Integer: i}
		value, returned, err := child.block(stmt.Body)
		if returned || err != nil {
			return value, returned, err
		}
	}
	return Value{}, false, nil
}

func (f *frame) condition(stmt *syntax.IfStatement) (Value, bool, error) {
	value, err := f.expr(stmt.Condition)
	if err != nil {
		return Value{}, false, err
	}
	if value.Type == "" {
		return Value{}, mayReturn(stmt.Then) || stmt.Else != nil && mayReturn(*stmt.Else), nil
	}
	if value.Bool {
		return f.child().block(stmt.Then)
	}
	if stmt.Else != nil {
		return f.child().block(*stmt.Else)
	}
	return Value{}, false, nil
}

func mayReturn(block syntax.BlockStatement) bool {
	for _, stmt := range block.Statements {
		switch stmt := stmt.(type) {
		case *syntax.ReturnStatement:
			return true
		case *syntax.IfStatement:
			if mayReturn(stmt.Then) || stmt.Else != nil && mayReturn(*stmt.Else) {
				return true
			}
		case *syntax.ForStatement:
			if mayReturn(stmt.Body) {
				return true
			}
		}
	}
	return false
}

func (f *frame) expr(expr syntax.Expression) (Value, error) {
	if err := f.enter(expr.Position()); err != nil {
		return Value{}, err
	}
	defer func() { f.executor.depth-- }()
	switch expr := expr.(type) {
	case *syntax.IntegerLiteral:
		return Value{Type: extension.Int, Integer: expr.Value}, nil
	case *syntax.FloatLiteral:
		return Value{Type: extension.Float, Float: expr.Value}, nil
	case *syntax.BooleanLiteral:
		return Value{Type: extension.Bool, Bool: expr.Value}, nil
	case *syntax.Identifier:
		if number, ok := semantic.Constant(expr.Name); ok {
			return Value{Type: extension.Float, Float: number}, nil
		}
		for scope := f; scope != nil; scope = scope.parent {
			if value, found := scope.locals[expr.Name]; found {
				return value, nil
			}
		}
	case *syntax.UnaryExpression:
		value, err := f.expr(expr.Operand)
		if err != nil || value.Type == "" {
			return Value{}, err
		}
		switch value.Type {
		case extension.Bool:
			value.Bool = !value.Bool
		case extension.Float:
			value.Float = -value.Float
		case extension.Int:
			if value.Integer == math.MinInt64 {
				return Value{}, diagnostic.At(expr.At, "E4000", "integer overflow")
			}
			value.Integer = -value.Integer
		}
		return value, nil
	case *syntax.BinaryExpression:
		return f.binary(expr)
	case *syntax.ArrayLiteral:
		return f.array(expr)
	case *syntax.IndexExpression:
		return f.index(expr)
	case *syntax.CallExpression:
		return f.call(expr)
	}
	return Value{}, diagnostic.At(expr.Position(), "E4000", "unsupported expression %T", expr)
}

func (f *frame) binary(expr *syntax.BinaryExpression) (Value, error) {
	left, err := f.expr(expr.Left)
	if err != nil || left.Type == "" {
		return Value{}, err
	}
	if expr.Op == "&&" && !left.Bool || expr.Op == "||" && left.Bool {
		return left, nil
	}
	right, err := f.expr(expr.Right)
	if err != nil || right.Type == "" {
		return Value{}, err
	}
	switch expr.Op {
	case "&&", "||":
		return right, nil
	case "==", "!=", "<", "<=", ">", ">=":
		var result bool
		switch {
		case left.Type == extension.Bool:
			result = (left.Bool == right.Bool) == (expr.Op == "==")
		case left.Type == extension.Float || right.Type == extension.Float:
			result = compare(expr.Op, convert(left, extension.Float).Float, convert(right, extension.Float).Float)
		default:
			result = compare(expr.Op, left.Integer, right.Integer)
		}
		return Value{Type: extension.Bool, Bool: result}, nil
	}
	if left.Type == extension.Float || right.Type == extension.Float {
		return floatArithmetic(expr, convert(left, extension.Float).Float, convert(right, extension.Float).Float)
	}
	number, err := arithmetic(expr.Op, left.Integer, right.Integer)
	if err != nil {
		return Value{}, diagnostic.At(expr.At, "E4000", "%v", err)
	}
	return Value{Type: extension.Int, Integer: number}, nil
}

func compare[T int64 | float64](op string, a, b T) bool {
	switch op {
	case "==":
		return a == b
	case "!=":
		return a != b
	case "<":
		return a < b
	case "<=":
		return a <= b
	case ">":
		return a > b
	default:
		return a >= b
	}
}

func (f *frame) array(expr *syntax.ArrayLiteral) (Value, error) {
	if err := f.reserveElements(expr.At, len(expr.Elements)); err != nil {
		return Value{}, err
	}
	value := Value{Type: f.program.Types[expr], Array: &extension.ArrayValue{Elements: make([]Value, len(expr.Elements))}}
	for i, item := range expr.Elements {
		v, err := f.expr(item)
		if err != nil {
			return Value{}, err
		}
		value.Array.Elements[i] = v
	}
	return value, nil
}

func (f *frame) reserveElements(at syntax.Pos, n int) error {
	if n > f.limits.MaxArrayElements-f.elements {
		return limit(at, "compile-time array elements", f.limits.MaxArrayElements)
	}
	f.elements += n
	return nil
}

func (f *frame) index(expr *syntax.IndexExpression) (Value, error) {
	target, err := f.expr(expr.Target)
	if err != nil {
		return Value{}, err
	}
	index, err := f.expr(expr.Index)
	if err != nil || index.Type == "" {
		return Value{}, err
	}
	typ := target.Type
	if typ == "" {
		typ = f.program.Types[expr.Target]
	}
	n, reg := semantic.RegisterSize(typ)
	if !reg {
		_, n, _ = typ.Array()
	}
	if index.Integer < 0 || n > 0 && index.Integer >= int64(n) {
		return Value{}, diagnostic.At(expr.Index.Position(), "E4000", "index %d out of bounds for %s", index.Integer, typ)
	}
	if target.Type == "" {
		return Value{}, nil
	}
	if reg {
		return Value{Type: extension.Qubit, Qubit: target.Qubit + ir.QubitID(index.Integer)}, nil
	}
	return target.Array.Elements[index.Integer], nil
}

func convert(v Value, target extension.Type) Value {
	if target == extension.Float && v.Type == extension.Int {
		return Value{Type: extension.Float, Float: float64(v.Integer)}
	}
	return v
}

func finite(number float64) bool {
	return !math.IsNaN(number) && !math.IsInf(number, 0)
}

func floatArithmetic(expr *syntax.BinaryExpression, left, right float64) (Value, error) {
	var result float64
	switch expr.Op {
	case "+":
		result = left + right
	case "-":
		result = left - right
	case "*":
		result = left * right
	case "/":
		if right == 0 {
			return Value{}, diagnostic.At(expr.At, "E4000", "division by zero")
		}
		result = left / right
	}
	if !finite(result) {
		return Value{}, diagnostic.At(expr.At, "E4000", "non-finite Float result")
	}
	return Value{Type: extension.Float, Float: result}, nil
}

func arithmetic(op string, left, right int64) (int64, error) {
	a, b := big.NewInt(left), big.NewInt(right)
	switch op {
	case "+":
		a.Add(a, b)
	case "-":
		a.Sub(a, b)
	case "*":
		a.Mul(a, b)
	case "/", "%":
		if right == 0 {
			return 0, fmt.Errorf("division by zero")
		}
		if op == "/" {
			a.Quo(a, b)
		} else {
			a.Rem(a, b)
		}
	}
	if !a.IsInt64() {
		return 0, fmt.Errorf("integer overflow")
	}
	return a.Int64(), nil
}
