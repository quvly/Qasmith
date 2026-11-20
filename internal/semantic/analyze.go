package semantic

import (
	"fmt"
	"math"
	"strings"

	"github.com/quvly/Qasmith/internal/diagnostic"
	"github.com/quvly/Qasmith/internal/limits"
	"github.com/quvly/Qasmith/internal/modules"
	"github.com/quvly/Qasmith/internal/syntax"
	"github.com/quvly/Qasmith/pkg/extension"
	"github.com/quvly/Qasmith/pkg/ir"
)

type Type = extension.Type

const (
	Int            = extension.Int
	Float          = extension.Float
	Qubit          = extension.Qubit
	Void           = extension.Void
	MeasuredResult = extension.MeasuredResult
)

type Function struct {
	Declaration   *syntax.FunctionDeclaration
	Module        *syntax.Module
	Signature     extension.Signature
	Intrinsic     *extension.Intrinsic
	Trusted       *extension.Function
	Namespace     string
	quantumEffect bool
	Builtin       string
	Gate          ir.Gate
	calls         []call
}
type Program struct {
	Entry    *Function
	Targets  map[*syntax.CallExpression]*Function
	Types    map[syntax.Expression]Type
	Static   bool
	Registry *extension.Registry
	Editor   *EditorData
}
type scope struct {
	names    map[string]*Function
	imported map[string]bool
}
type analyzer struct {
	*Program
	scopes    map[*syntax.Module]*scope
	functions []*Function
	limits    limits.Limits
}
type call struct {
	at          syntax.Pos
	target      *Function
	resources   []int
	conditional bool
}

func builtin(name string, result Type, args ...Type) *Function {
	return &Function{Builtin: name, Signature: extension.Signature{Parameters: args, Return: result}}
}

var builtins = builtinFunctions()

func builtinFunctions() map[string]*Function {
	functions := map[string]*Function{
		"qubit":   builtin("qubit", Qubit),
		"qreg":    builtin("qreg", Register, Int),
		"len":     builtin("len", Int),
		"measure": builtin("measure", MeasuredResult),
	}
	for _, gate := range []ir.Gate{ir.X, ir.H, ir.CX, ir.RX, ir.RY, ir.RZ} {
		spec := gate.Spec()
		name := strings.ToLower(spec.Name)
		f := builtin(name, Void)
		f.Gate = gate
		for range spec.Parameters {
			f.Signature.Parameters = append(f.Signature.Parameters, Float)
		}
		for i := 0; i < spec.Qubits; i++ {
			if spec.DistinctQubits {
				for j := 0; j < i; j++ {
					f.Signature.Distinct = append(f.Signature.Distinct, [2]int{spec.Parameters + j, spec.Parameters + i})
				}
			}
			f.Signature.Parameters = append(f.Signature.Parameters, Qubit)
		}
		functions[name] = f
	}
	return functions
}

func Constant(name string) (float64, bool) {
	return math.Pi, name == "pi"
}

func reserved(name string) bool {
	_, constant := Constant(name)
	return constant || builtins[name] != nil && name != "len" && name != "qreg"
}

func numeric(t Type) bool { return t == Int || t == Float }

func Analyze(graph *modules.Graph, registry *extension.Registry, limits limits.Limits) (*Program, error) {
	return analyze(graph, registry, limits, nil)
}

func AnalyzeEditor(graph *modules.Graph, registry *extension.Registry, limits limits.Limits, editor *EditorData) (*Program, error) {
	editor.References = make(map[*syntax.Identifier]Local)
	editor.Scopes = make(map[string]map[string]*Function)
	editor.Imports = make(map[syntax.Pos]*Function)
	return analyze(graph, registry, limits, editor)
}

func analyze(graph *modules.Graph, registry *extension.Registry, limits limits.Limits, editor *EditorData) (*Program, error) {
	a := analyzer{Program: &Program{Targets: make(map[*syntax.CallExpression]*Function), Types: make(map[syntax.Expression]Type), Registry: registry, Editor: editor}, scopes: make(map[*syntax.Module]*scope), limits: limits}
	record := func(err error) error {
		if err == nil || editor == nil {
			return err
		}
		editor.Issues = append(editor.Issues, err)
		return nil
	}
	for _, module := range graph.Modules {
		if editor != nil && editor.Context.Err() != nil {
			return nil, editor.Context.Err()
		}
		if err := record(a.declare(module)); err != nil {
			return nil, err
		}
	}
	a.Entry = a.scopes[graph.Entry].names["main"]
	if a.Entry == nil && editor == nil {
		return nil, diagnostic.At(syntax.Pos{File: graph.Entry.Path, Line: 1, Column: 1}, "E3000", "entry file must declare main")
	}
	if a.Entry != nil && (len(a.Entry.Signature.Parameters) > 0 || a.Entry.Signature.Return != Int && a.Entry.Signature.Return != MeasuredResult) {
		if err := record(diagnostic.At(a.Entry.Declaration.At, "E3000", "main must take no parameters and return Int or MeasuredResult")); err != nil {
			return nil, err
		}
	}
	for _, module := range graph.Modules {
		if err := record(a.importSymbols(graph, module, registry)); err != nil {
			return nil, err
		}
		if editor != nil {
			editor.Scopes[module.Path] = a.scopes[module].names
		}
	}
	for _, f := range a.functions {
		if editor != nil && editor.Context.Err() != nil {
			return nil, editor.Context.Err()
		}
		if err := record(a.body(f)); err != nil {
			return nil, err
		}
	}
	if err := record(verifyCalls(a.functions)); err != nil {
		return nil, err
	}
	if a.Entry != nil {
		for _, call := range a.Entry.calls {
			if call.target.Trusted != nil && call.target.Trusted.Produce != nil && a.Entry.quantumEffect {
				if err := record(diagnostic.At(call.at, "E3100", "terminal circuit producer cannot be combined with independent quantum operations")); err != nil {
					return nil, err
				}
			}
		}
	}
	return a.Program, nil
}

func (a *analyzer) declare(module *syntax.Module) error {
	s := &scope{make(map[string]*Function), make(map[string]bool)}
	a.scopes[module] = s
	for _, decl := range module.Functions {
		if s.names[decl.Name] != nil || reserved(decl.Name) {
			return diagnostic.At(decl.At, "E3001", "duplicate or reserved function %q", decl.Name)
		}
		result, valid := a.annotation(Type(decl.Result))
		if !valid || quantum(result) {
			return diagnostic.At(decl.ResultAt, "E3002", "unsupported return type %q", decl.Result)
		}
		if !arrayFits(result, a.limits.MaxArrayElements) {
			return diagnostic.At(decl.ResultAt, "E4100", "array element limit exceeded")
		}
		f := &Function{Declaration: decl, Module: module, Signature: extension.Signature{Return: result}}
		for _, param := range decl.Params {
			typ, valid := a.annotation(Type(param.Type))
			if !valid || !a.compileTime(typ) && !quantum(typ) {
				return diagnostic.At(param.TypeAt, "E3002", "unsupported parameter type %q", param.Type)
			}
			if quantum(typ) && a.compileTime(result) {
				return diagnostic.At(param.At, "E3002", "classical functions cannot borrow quantum resources")
			}
			if !arrayFits(typ, a.limits.MaxArrayElements) {
				return diagnostic.At(param.TypeAt, "E4100", "array element limit exceeded")
			}
			f.Signature.Parameters = append(f.Signature.Parameters, typ)
		}
		s.names[decl.Name] = f
		a.functions = append(a.functions, f)
	}
	return nil
}

func (a *analyzer) importSymbols(graph *modules.Graph, module *syntax.Module, registry *extension.Registry) error {
	s := a.scopes[module]
	for i, decl := range module.Imports {
		for _, name := range decl.Names {
			if s.names[name.Alias] != nil || reserved(name.Alias) {
				return diagnostic.At(name.At, "E2003", "import name %q conflicts with an existing symbol", name.Alias)
			}
			var target *Function
			if imported := graph.Edges[module][i]; imported != nil {
				target = a.scopes[imported].names[name.Name]
				if target == nil || target.Module != imported {
					return diagnostic.At(name.At, "E2001", "module %s has no exported symbol %q", imported.Path, name.Name)
				}
				if !target.Declaration.Public {
					return diagnostic.At(name.At, "E2001", "symbol %q is private to %s", name.Name, imported.Path)
				}
			} else if trusted, ok := registry.LookupFunction(decl.Path, name.Name); ok {
				target = &Function{Trusted: &trusted, Namespace: decl.Path, Signature: extension.Signature{Return: trusted.Return}}
				for _, param := range trusted.Parameters {
					target.Signature.Parameters = append(target.Signature.Parameters, param.Type)
				}
			} else if intrinsic, ok := registry.Lookup(decl.Path, name.Name); ok {
				target = &Function{Intrinsic: &intrinsic, Signature: intrinsic.Signature, quantumEffect: intrinsic.Signature.Return == Void}
			} else if registry.HasModule(decl.Path) {
				return diagnostic.At(name.At, "E2001", "module %s has no exported symbol %q", decl.Path, name.Name)
			} else if a.Editor != nil && !strings.HasPrefix(decl.Path, "host:") {
				continue
			} else {
				return diagnostic.At(name.At, "E2001", "unregistered host intrinsic %s/%s", decl.Path, name.Name)
			}
			s.names[name.Alias], s.imported[name.Alias] = target, true
			if a.Editor != nil {
				a.Editor.Imports[name.At] = target
				a.Editor.Imports[name.AliasAt] = target
			}
		}
	}
	return nil
}

func verifyCalls(functions []*Function) error {
	state := make(map[*Function]int)
	var stack []*Function
	var visit func(*Function) error
	visit = func(f *Function) error {
		if state[f] == 2 {
			return nil
		}
		state[f] = 1
		stack = append(stack, f)
		defer func() { stack = stack[:len(stack)-1] }()
		pairs := make(map[[2]int]bool)
		for _, call := range f.calls {
			target := call.target
			if state[target] == 1 {
				var chain []string
				for _, frame := range stack {
					chain = append(chain, qualified(frame))
				}
				chain = append(chain, qualified(target))
				return diagnostic.At(call.at, "E3005", "unsupported recursion: %s", strings.Join(chain, " -> "))
			}
			if target.Declaration != nil {
				if err := visit(target); err != nil {
					return err
				}
			}
			if target.quantumEffect || target.Gate != "" || allocation(target) || target.Builtin == "measure" {
				f.quantumEffect = true
			}
			if call.conditional {
				continue
			}
			for _, pair := range target.Signature.Distinct {
				left, right := call.resources[pair[0]], call.resources[pair[1]]
				if left < 0 || right < 0 {
					continue
				}
				if left == right {
					return diagnostic.At(call.at, "E3100", "call to %s requires distinct qubit operands (parameters %d and %d)", qualified(target), pair[0]+1, pair[1]+1)
				}
				if left < len(f.Signature.Parameters) && right < len(f.Signature.Parameters) {
					key := [2]int{min(left, right), max(left, right)}
					if !pairs[key] {
						pairs[key] = true
						f.Signature.Distinct = append(f.Signature.Distinct, key)
					}
				}
			}
		}
		state[f] = 2
		return nil
	}
	for _, f := range functions {
		if err := visit(f); err != nil {
			return err
		}
	}
	return nil
}

func qualified(f *Function) string {
	switch {
	case f.Builtin != "":
		return f.Builtin
	case f.Trusted != nil:
		return f.Namespace + "/" + f.Trusted.Name
	case f.Intrinsic != nil:
		return fmt.Sprintf("host:%s/%s", f.Intrinsic.Namespace, f.Intrinsic.Name)
	default:
		return fmt.Sprintf("%s (%s)", f.Declaration.Name, f.Declaration.At)
	}
}

func (a *analyzer) compileTime(t Type) bool {
	return t.Classical() || a.Registry.LookupType(t) != nil
}

func (a *analyzer) annotation(t Type) (Type, bool) {
	if a.Registry.LookupType(t) != nil {
		return t, true
	}
	return annotation(t)
}
