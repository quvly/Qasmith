package semantic

import (
	"maps"

	"github.com/quvly/Qasmith/internal/diagnostic"
	"github.com/quvly/Qasmith/internal/syntax"
	"github.com/quvly/Qasmith/pkg/extension"
)

type value struct {
	typ      Type
	resource int
	local    Local
}

type body struct {
	*analyzer
	f            *Function
	scope        *scope
	locals       map[string]value
	nextResource int
	nested       bool
	end          syntax.Pos
}

func (a *analyzer) body(f *Function) error {
	b := body{analyzer: a, f: f, scope: a.scopes[f.Module], locals: make(map[string]value), nextResource: len(f.Signature.Parameters), end: f.Declaration.Body.End}
	for i, param := range f.Declaration.Params {
		if err := b.bind(param.At, param.Name, value{typ: f.Signature.Parameters[i], resource: i}); err != nil {
			return err
		}
	}
	if f.Signature.Return == MeasuredResult && f != a.Entry {
		return diagnostic.At(f.Declaration.At, "E3100", "only the entry point may return MeasuredResult")
	}
	returned, err := b.block(f.Declaration.Body)
	if err != nil {
		return err
	}
	if !returned && f.Signature.Return != Void {
		return diagnostic.At(f.Declaration.At, "E3002", "missing return of type %s", f.Signature.Return)
	}
	return nil
}

func (b body) child() body {
	b.locals = maps.Clone(b.locals)
	b.nested = true
	return b
}

func (b *body) block(block syntax.BlockStatement) (bool, error) {
	b.end = block.End
	returned := false
	for _, stmt := range block.Statements {
		if returned {
			return false, diagnostic.At(stmt.Position(), "E3002", "unreachable statement after return")
		}
		var err error
		returned, err = b.statement(stmt)
		if err != nil {
			return false, err
		}
	}
	return returned, nil
}

func (b *body) statement(stmt syntax.Statement) (bool, error) {
	switch stmt := stmt.(type) {
	case *syntax.VariableDeclaration:
		direct, _ := stmt.Value.(*syntax.CallExpression)
		v, err := b.expr(stmt.Value, direct, "allocate")
		if err != nil {
			return false, err
		}
		if !b.compileTime(v.typ) && !(quantum(v.typ) && direct != nil && allocation(b.Targets[direct])) {
			return false, diagnostic.At(stmt.At, "E3100", "let requires a classical value or a new qubit or register allocation; pass borrowed elements directly as arguments")
		}
		return false, b.bind(stmt.NameAt, stmt.Name, v)
	case *syntax.ReturnStatement:
		v := value{typ: Void}
		if stmt.Value != nil {
			direct, _ := stmt.Value.(*syntax.CallExpression)
			var err error
			v, err = b.expr(stmt.Value, direct, "measure")
			if err != nil {
				return false, err
			}
			if v.typ == Void {
				return false, diagnostic.At(stmt.At, "E3002", "use a bare return in a Void function")
			}
		}
		if !assignable(v.typ, b.f.Signature.Return) {
			return false, diagnostic.At(stmt.At, "E3002", "expected return type %s, got %s", b.f.Signature.Return, v.typ)
		}
		return true, nil
	case *syntax.ExpressionStatement:
		v, err := b.expr(stmt.Value, nil, "")
		if err != nil {
			return false, err
		}
		if v.typ != Void {
			return false, diagnostic.At(stmt.At, "E3002", "expression statement must be a Void call")
		}
	case *syntax.ForStatement:
		b.Static = true
		for _, bound := range []syntax.Expression{stmt.Start, stmt.End} {
			if err := b.requireType(bound, Int, "loop bound"); err != nil {
				return false, err
			}
		}
		child := b.child()
		child.end = stmt.Body.End
		if err := child.bind(stmt.NameAt, stmt.Name, value{typ: Int, local: Local{Start: stmt.Body.At}}); err != nil {
			return false, err
		}
		_, err := child.block(stmt.Body)
		return false, err
	case *syntax.IfStatement:
		b.Static = true
		if err := b.requireType(stmt.Condition, Bool, "condition"); err != nil {
			return false, err
		}
		child := b.child()
		thenReturns, err := child.block(stmt.Then)
		if err != nil || stmt.Else == nil {
			return false, err
		}
		child = b.child()
		elseReturns, err := child.block(*stmt.Else)
		return thenReturns && elseReturns, err
	}
	return false, nil
}

func allocation(target *Function) bool {
	return target != nil && (target.Builtin == "qubit" || target.Builtin == "qreg")
}

func (b *body) bind(at syntax.Pos, name string, v value) error {
	_, constant := Constant(name)
	if _, exists := b.locals[name]; exists || b.scope.imported[name] || constant {
		return diagnostic.At(at, "E3001", "local name %q conflicts with an existing or reserved binding", name)
	}
	if b.Editor != nil {
		start := v.local.Start
		if start.Line == 0 {
			start = at
		}
		v.local = Local{Name: name, Type: v.typ, At: at, Start: start, End: b.end}
		b.Editor.Locals = append(b.Editor.Locals, v.local)
	}
	b.locals[name] = v
	return nil
}

func (b *body) requireType(expr syntax.Expression, wanted Type, role string) error {
	v, err := b.expr(expr, nil, "")
	if err != nil {
		return err
	}
	if v.typ != wanted {
		return diagnostic.At(expr.Position(), "E3002", "%s requires %s, got %s", role, wanted, v.typ)
	}
	return nil
}

func (b *body) expr(expr syntax.Expression, direct *syntax.CallExpression, allowed string) (value, error) {
	v, err := b.expression(expr, direct, allowed)
	if err == nil {
		b.Types[expr] = v.typ
	}
	return v, err
}

func (b *body) expression(expr syntax.Expression, direct *syntax.CallExpression, allowed string) (value, error) {
	switch e := expr.(type) {
	case *syntax.IntegerLiteral:
		return value{typ: Int}, nil
	case *syntax.FloatLiteral:
		return value{typ: Float}, nil
	case *syntax.BooleanLiteral:
		b.Static = true
		return value{typ: Bool}, nil
	case *syntax.Identifier:
		if _, ok := Constant(e.Name); ok {
			return value{typ: Float}, nil
		}
		v, found := b.locals[e.Name]
		if !found {
			return value{}, diagnostic.At(e.At, "E3003", "undefined variable %q", e.Name)
		}
		if b.Editor != nil {
			b.Editor.References[e] = v.local
		}
		return v, nil
	case *syntax.UnaryExpression:
		v, err := b.expr(e.Operand, nil, "")
		if err != nil {
			return value{}, err
		}
		if e.Op == "!" && v.typ != Bool {
			return value{}, diagnostic.At(e.At, "E3002", "operator ! requires Bool, got %s", v.typ)
		}
		if e.Op == "-" && !numeric(v.typ) {
			return value{}, diagnostic.At(e.At, "E3002", "unary minus requires a numeric operand, got %s", v.typ)
		}
		return v, nil
	case *syntax.BinaryExpression:
		return b.binary(e)
	case *syntax.ArrayLiteral:
		b.Static = true
		var element Type
		for _, item := range e.Elements {
			v, err := b.expr(item, nil, "")
			if err != nil {
				return value{}, err
			}
			if !v.typ.Classical() || element != "" && element != v.typ {
				return value{}, diagnostic.At(item.Position(), "E3002", "array elements must have one classical type and consistent dimensions")
			}
			element = v.typ
		}
		if element == "" {
			return value{}, diagnostic.At(e.At, "E3002", "empty array has no inferable element type")
		}
		t := extension.ArrayType(element, len(e.Elements))
		if !arrayFits(t, b.limits.MaxArrayElements) {
			return value{}, diagnostic.At(e.At, "E4100", "array element limit exceeded")
		}
		return value{typ: t}, nil
	case *syntax.IndexExpression:
		b.Static = true
		target, err := b.expr(e.Target, nil, "")
		if err != nil {
			return value{}, err
		}
		if err := b.requireType(e.Index, Int, "index"); err != nil {
			return value{}, err
		}
		n, reg := RegisterSize(target.typ)
		element := Qubit
		if !reg {
			var array bool
			element, n, array = target.typ.Array()
			if !array {
				return value{}, diagnostic.At(e.At, "E3002", "cannot index %s", target.typ)
			}
		}
		if index, literal := e.Index.(*syntax.IntegerLiteral); literal && (index.Value < 0 || n > 0 && index.Value >= int64(n)) {
			return value{}, diagnostic.At(index.At, "E4000", "index %d out of bounds for %s", index.Value, target.typ)
		}
		return value{typ: element, resource: -1}, nil
	case *syntax.CallExpression:
		if e != direct {
			allowed = ""
		}
		return b.call(e, allowed)
	default:
		return value{}, diagnostic.At(expr.Position(), "E3002", "unsupported expression")
	}
}

func (b *body) binary(e *syntax.BinaryExpression) (value, error) {
	left, err := b.expr(e.Left, nil, "")
	if err != nil {
		return value{}, err
	}
	rightBody := *b
	if e.Op == "&&" || e.Op == "||" {
		rightBody.nested = true
	}
	right, err := rightBody.expr(e.Right, nil, "")
	if err != nil {
		return value{}, err
	}

	a, z := left.typ, right.typ
	switch e.Op {
	case "&&", "||":
		b.Static = true
		if a == Bool && z == Bool {
			return value{typ: Bool}, nil
		}
		return value{}, diagnostic.At(e.At, "E3002", "operator %s requires Bool operands", e.Op)
	case "==", "!=":
		b.Static = true
		if a == Bool && z == Bool || numeric(a) && numeric(z) {
			return value{typ: Bool}, nil
		}
		return value{}, diagnostic.At(e.At, "E3002", "operator %s requires two numeric or two Bool operands", e.Op)
	case "<", "<=", ">", ">=":
		b.Static = true
		if numeric(a) && numeric(z) {
			return value{typ: Bool}, nil
		}
		return value{}, diagnostic.At(e.At, "E3002", "operator %s requires numeric operands", e.Op)
	case "%":
		if a == Int && z == Int {
			return value{typ: Int}, nil
		}
		return value{}, diagnostic.At(e.At, "E3002", "operator %% requires Int operands")
	default:
		if !numeric(a) || !numeric(z) {
			return value{}, diagnostic.At(e.At, "E3002", "operator %s requires numeric operands", e.Op)
		}
		if a == Float || z == Float {
			return value{typ: Float}, nil
		}
		return value{typ: Int}, nil
	}
}

func (b *body) call(e *syntax.CallExpression, allowed string) (value, error) {
	fail := func(code, format string, args ...any) (value, error) {
		return value{}, diagnostic.At(e.At, code, format, args...)
	}
	if _, found := b.locals[e.Name]; found {
		return fail("E3003", "local value %q is not callable", e.Name)
	}
	target := b.scope.names[e.Name]
	if target == nil {
		target = builtins[e.Name]
	}
	if target == nil {
		return fail("E3003", "undefined function %q", e.Name)
	}
	if b.Editor != nil {
		b.Targets[e] = target
	}
	result, params := target.Signature.Return, target.Signature.Parameters
	if target == b.Entry && result == MeasuredResult {
		return fail("E3100", "the quantum entry point cannot be called")
	}
	if b.compileTime(b.f.Signature.Return) && result == Void {
		return fail("E3100", "classical functions may call only pure numeric or other classical functions")
	}
	if target.Trusted != nil {
		b.Static = true
		if target.Trusted.Produce != nil && (allowed != "measure" || b.nested || b.f != b.Entry || b.f.Signature.Return != MeasuredResult) {
			return fail("E3100", "terminal circuit producer must be the final returned expression of the quantum entry point")
		}
	}
	switch target.Builtin {
	case "qubit", "qreg":
		if allowed != "allocate" || b.f != b.Entry || b.f.Signature.Return != MeasuredResult {
			return fail("E3100", "%s() requires a let binding in the quantum entry point", target.Builtin)
		}
		if target.Builtin == "qreg" {
			b.Static = true
			if len(e.Args) == 1 {
				if literal, ok := e.Args[0].(*syntax.IntegerLiteral); ok && literal.Value > 0 && literal.Value <= int64(b.limits.MaxQubits) {
					result = RegisterType(int(literal.Value))
				}
			}
		}
	case "measure":
		if allowed != "measure" || b.nested || b.f != b.Entry || b.f.Signature.Return != MeasuredResult {
			return fail("E3100", "measure() must be the final returned expression of the quantum entry point")
		}
		if len(e.Args) == 0 {
			return fail("E3004", "measure requires at least one qubit")
		}
		params = make([]Type, len(e.Args))
	case "len":
		b.Static = true
		params = []Type{""}
	}
	if len(e.Args) != len(params) {
		return fail("E3004", "%s expects %d arguments, got %d", e.Name, len(params), len(e.Args))
	}
	c := call{at: e.At, target: target, resources: make([]int, len(params)), conditional: b.nested}
	measured := make(map[int]bool)
	types := make([]Type, len(e.Args))
	for i, arg := range e.Args {
		v, err := b.expr(arg, nil, "")
		if err != nil {
			return value{}, err
		}
		types[i] = v.typ
		_, reg := RegisterSize(v.typ)
		switch target.Builtin {
		case "len":
			if _, _, array := v.typ.Array(); !array && !reg {
				return fail("E3004", "len expects an array or quantum register")
			}
		case "measure":
			if v.typ != Qubit && !(reg && len(e.Args) == 1) {
				return fail("E3004", "measure expects individual qubits or a single register")
			}
			if v.typ == Qubit && v.resource >= 0 {
				if measured[v.resource] {
					return fail("E3100", "measure requires distinct qubit operands")
				}
				measured[v.resource] = true
			}
		default:
			if target.Trusted != nil {
				param := target.Trusted.Parameters[i]
				if !param.Accepts(v.typ) {
					return value{}, diagnostic.At(arg.Position(), "E3004", "argument %d to %s expects %s, got %s", i+1, e.Name, param, v.typ)
				}
				if param.Rank == 2 {
					row, n, _ := v.typ.Array()
					_, width, _ := row.Array()
					if n > b.limits.MaxMatrixElements/width {
						return value{}, diagnostic.At(arg.Position(), "E4100", "matrix element limit exceeded (maximum %d)", b.limits.MaxMatrixElements)
					}
				}
			} else if !assignable(v.typ, params[i]) {
				return value{}, diagnostic.At(arg.Position(), "E3004", "argument %d to %s expects %s, got %s", i+1, e.Name, params[i], v.typ)
			}
		}
		c.resources[i] = v.resource
	}
	if b.Editor == nil && target.Trusted != nil && target.Trusted.CheckTypes != nil {
		if err := target.Trusted.CheckTypes(b.limits, types); err != nil {
			return fail("E3004", "%v", err)
		}
	}
	b.Targets[e] = target
	b.f.calls = append(b.f.calls, c)
	v := value{typ: result}
	if result == Qubit {
		v.resource = b.nextResource
		b.nextResource++
	}
	return v, nil
}
