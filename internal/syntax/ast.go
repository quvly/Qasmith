package syntax

import "github.com/quvly/Qasmith/internal/diagnostic"

type Pos = diagnostic.Position

type Program struct {
	Entry   *Module
	Modules []*Module
}

type Module struct {
	Path      string
	Imports   []ImportDeclaration
	Functions []*FunctionDeclaration
}

type ImportDeclaration struct {
	At    Pos
	Path  string
	Names []ImportSpecifier
}

type ImportSpecifier struct {
	At          Pos
	Name, Alias string
	AliasAt     Pos
}

type FunctionDeclaration struct {
	At           Pos
	Public       bool
	Name, Result string
	ResultAt     Pos
	NameAt       Pos
	Params       []Parameter
	Body         BlockStatement
}

type Parameter struct {
	At, TypeAt Pos
	Name, Type string
}

type BlockStatement struct {
	At, End    Pos
	Statements []Statement
}

type Node struct{ At Pos }

func (n Node) Position() Pos { return n.At }

type Statement interface {
	Position() Pos
	statement()
}
type VariableDeclaration struct {
	Node
	Name   string
	NameAt Pos
	Value  Expression
}
type ReturnStatement struct {
	Node
	Value Expression
}
type ExpressionStatement struct {
	Node
	Value Expression
}

func (*VariableDeclaration) statement() {}
func (*ReturnStatement) statement()     {}
func (*ExpressionStatement) statement() {}

type Expression interface {
	Position() Pos
	expression()
}
type CallExpression struct {
	Node
	Name string
	Args []Expression
}
type BinaryExpression struct {
	Node
	Op          string
	Left, Right Expression
}
type IntegerLiteral struct {
	Node
	Value int64
}
type Identifier struct {
	Node
	Name string
}

func (*CallExpression) expression()   {}
func (*BinaryExpression) expression() {}
func (*IntegerLiteral) expression()   {}
func (*Identifier) expression()       {}

type FloatLiteral struct {
	Node
	Value float64
}

type UnaryExpression struct {
	Node
	Op      string
	Operand Expression
}

func (*FloatLiteral) expression()    {}
func (*UnaryExpression) expression() {}

type BooleanLiteral struct {
	Node
	Value bool
}
type ArrayLiteral struct {
	Node
	Elements []Expression
}
type IndexExpression struct {
	Node
	Target, Index Expression
}
type ForStatement struct {
	Node
	Name       string
	NameAt     Pos
	Start, End Expression
	Body       BlockStatement
}
type IfStatement struct {
	Node
	Condition Expression
	Then      BlockStatement
	Else      *BlockStatement
}

func (*BooleanLiteral) expression()  {}
func (*ArrayLiteral) expression()    {}
func (*IndexExpression) expression() {}
func (*ForStatement) statement()     {}
func (*IfStatement) statement()      {}
