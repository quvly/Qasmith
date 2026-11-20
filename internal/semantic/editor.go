package semantic

import (
	"context"

	"github.com/quvly/Qasmith/internal/syntax"
)

// Local records a compiler-resolved lexical binding and its visibility range.
type Local struct {
	Name           string
	Type           Type
	At, Start, End syntax.Pos
}

// EditorData contains partial semantic information even when checking fails.
type EditorData struct {
	Context    context.Context
	Issues     []error
	Locals     []Local
	References map[*syntax.Identifier]Local
	Scopes     map[string]map[string]*Function
	Imports    map[syntax.Pos]*Function
}

func BuiltinFunctions() map[string]*Function { return builtins }
