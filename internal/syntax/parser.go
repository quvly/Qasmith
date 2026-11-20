package syntax

import (
	"errors"
	"strconv"

	"github.com/quvly/Qasmith/internal/diagnostic"
)

type parser struct {
	tokens          []token
	index           int
	depth           int
	expressionStart int
	blockDepth      int
	err             error
	editor          bool
}

func Parse(file, source string) (*Module, error) {
	m, err := parse(file, source, false)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// ParseEditor retains completed declarations and statements before a syntax error.
func ParseEditor(file, source string) (*Module, error) { return parse(file, source, true) }

func parse(file, source string, editor bool) (*Module, error) {
	tokens, err := lex(file, source)
	if err != nil {
		var detail *diagnostic.Error
		if !editor || !errors.As(err, &detail) {
			return nil, err
		}
		tokens = append(tokens, token{kind: "end of file", at: detail.Position})
	}
	p := parser{tokens: tokens, editor: editor}
	m := &Module{Path: file}
	for p.current().kind != "end of file" && p.err == nil {
		if p.current().kind == "import" {
			d := p.importDecl()
			if d.Path != "" {
				m.Imports = append(m.Imports, d)
			}
		} else {
			f := p.function()
			if f.Name != "" && f.Result != "" && f.Body.At.Line != 0 {
				m.Functions = append(m.Functions, f)
			}
		}
	}
	if err != nil {
		return m, err
	}
	if p.err != nil {
		return m, p.err
	}
	return m, nil
}

func (p *parser) current() token { return p.tokens[p.index] }
func (p *parser) take() token {
	t := p.current()
	if t.kind != "end of file" {
		p.index++
	}
	return t
}
func (p *parser) match(kind string) bool {
	if p.current().kind != kind {
		return false
	}
	p.take()
	return true
}
func (p *parser) fail(t token, format string, args ...any) {
	if p.err == nil {
		p.err = diagnostic.At(t.at, "E1002", format, args...)
	}
}
func (p *parser) expect(kind string) token {
	t := p.take()
	if t.kind != kind {
		p.fail(t, "expected %s, found %s", kind, t.kind)
	}
	return t
}

func (p *parser) importDecl() ImportDeclaration {
	at := p.expect("import").at
	p.expect("{")
	var names []ImportSpecifier
	for p.err == nil && !(p.editor && p.current().kind == "}") {
		name := p.take()
		alias := name

		switch name.kind {
		case "identifier":
			if p.match("as") {
				alias = p.expect("identifier")
			}
		case "for", "in", "if", "else", "true", "false":
			// Previously valid host names can still be imported with an alias.
			p.expect("as")
			alias = p.expect("identifier")
		default:
			p.fail(name, "expected identifier, found %s", name.kind)
		}
		names = append(names, ImportSpecifier{name.at, name.value, alias.value, alias.at})
		if !p.match(",") || p.current().kind == "}" {
			break
		}
	}
	p.expect("}")
	p.expect("from")
	path := p.expect("string")
	p.expect(";")
	return ImportDeclaration{at, path.value, names}
}

func (p *parser) function() *FunctionDeclaration {
	pub := p.match("pub")
	at := p.expect("fn").at
	name := p.expect("identifier")
	p.expect("(")
	var params []Parameter
	for p.current().kind != ")" && p.err == nil {
		param := p.expect("identifier")
		p.expect(":")
		typ := p.typeAnnotation()
		params = append(params, Parameter{param.at, typ.at, param.value, typ.value})
		if !p.match(",") {
			break
		}
	}
	p.expect(")")
	p.expect("->")
	result := p.typeAnnotation()
	body := p.block()
	return &FunctionDeclaration{at, pub, name.value, result.value, result.at, name.at, params, body}
}

func (p *parser) typeAnnotation() token {
	typ := p.expect("identifier")
	if typ.value == "QReg" {
		p.expect("<")
		size := p.expect("integer")
		p.expect(">")
		typ.value += "<" + size.value + ">"
	}
	for dimensions := 0; p.err == nil && p.match("["); dimensions++ {
		if dimensions == 256 {
			p.fail(p.current(), "array type nesting exceeds 256")
			break
		}
		size := p.expect("integer")
		p.expect("]")
		typ.value += "[" + size.value + "]"
	}
	return typ
}

func (p *parser) block() BlockStatement {
	body := BlockStatement{At: p.expect("{").at}
	p.blockDepth++
	defer func() { p.blockDepth-- }()
	if p.blockDepth > 256 {
		p.fail(p.current(), "block nesting exceeds 256")
		return body
	}
	for p.current().kind != "}" && p.err == nil {
		stmt := p.statement()
		if p.err == nil || !p.editor {
			body.Statements = append(body.Statements, stmt)
		}
	}
	body.End = p.expect("}").at
	return body
}

func (p *parser) statement() Statement {
	start := Node{p.current().at}
	var stmt Statement
	switch {
	case p.match("for"):
		name := p.expect("identifier")
		p.expect("in")
		startExpr := p.expression(1)
		p.expect("..")
		endExpr := p.expression(1)
		return &ForStatement{start, name.value, name.at, startExpr, endExpr, p.block()}
	case p.match("if"):
		condition := p.expression(1)
		then := p.block()
		var otherwise *BlockStatement
		if p.match("else") {
			block := p.block()
			otherwise = &block
		}
		return &IfStatement{start, condition, then, otherwise}
	case p.match("let"):
		name := p.expect("identifier")
		p.expect("=")
		stmt = &VariableDeclaration{start, name.value, name.at, p.expression(1)}
	case p.match("return"):
		var value Expression
		if p.current().kind != ";" {
			value = p.expression(1)
		}
		stmt = &ReturnStatement{start, value}
	default:
		stmt = &ExpressionStatement{start, p.expression(1)}
	}
	p.expect(";")
	return stmt
}

func precedence(op string) int {
	switch op {
	case "||":
		return 1
	case "&&":
		return 2
	case "==", "!=":
		return 3
	case "<", "<=", ">", ">=":
		return 4
	case "+", "-":
		return 5
	case "*", "/", "%":
		return 6
	default:
		return 0
	}
}

func (p *parser) expression(min int) Expression {
	if p.depth == 0 {
		p.expressionStart = p.index
	}
	p.depth++
	defer func() { p.depth-- }()
	if p.index-p.expressionStart > 4096 {
		p.fail(p.current(), "expression exceeds 4096 tokens")
		return nil
	}
	if p.depth > 512 {
		p.fail(p.current(), "expression nesting exceeds 512")
		return nil
	}
	left := p.primary()
	for p.err == nil && p.match("[") {
		index := p.expression(1)
		p.expect("]")
		left = &IndexExpression{Node{indexPosition(index, p.current().at)}, left, index}
	}
	for p.err == nil && precedence(p.current().kind) >= min {
		if p.index-p.expressionStart > 4096 {
			p.fail(p.current(), "expression exceeds 4096 tokens")
			break
		}
		op := p.take()
		right := p.expression(precedence(op.kind) + 1)
		left = &BinaryExpression{Node{op.at}, op.kind, left, right}
	}
	return left
}

func (p *parser) primary() Expression {
	t := p.take()
	node := Node{t.at}
	switch t.kind {
	case "!":
		return &UnaryExpression{node, "!", p.expression(7)}
	case "true", "false":
		return &BooleanLiteral{node, t.kind == "true"}
	case "[":
		array := &ArrayLiteral{Node: node}
		for p.current().kind != "]" && p.err == nil {
			array.Elements = append(array.Elements, p.expression(1))
			if !p.match(",") {
				break
			}
		}
		p.expect("]")
		return array
	case "-":
		// A signed literal preserves the full int64 range, including MinInt64.
		if p.current().kind == "integer" {
			return p.integer(t, "-"+p.take().value)
		}
		return &UnaryExpression{node, "-", p.expression(7)}
	case "integer":
		return p.integer(t, t.value)
	case "float":
		number, err := strconv.ParseFloat(t.value, 64)
		if err != nil {
			p.fail(t, "invalid Float literal %q: %v", t.value, err)
		}
		return &FloatLiteral{node, number}
	case "identifier":
		if !p.match("(") {
			return &Identifier{node, t.value}
		}
		call := &CallExpression{Node: node, Name: t.value}
		for p.current().kind != ")" && p.err == nil {
			call.Args = append(call.Args, p.expression(1))
			if !p.match(",") {
				break
			}
		}
		p.expect(")")
		return call
	case "(":
		expr := p.expression(1)
		p.expect(")")
		return expr
	default:
		p.fail(t, "expected expression, found %s", t.kind)
		return nil
	}
}

func (p *parser) integer(t token, value string) Expression {
	number, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		p.fail(t, "integer literal outside the signed 64-bit range")
	}
	return &IntegerLiteral{Node{t.at}, number}
}

func indexPosition(expr Expression, fallback Pos) Pos {
	if expr == nil {
		return fallback
	}
	return expr.Position()
}
