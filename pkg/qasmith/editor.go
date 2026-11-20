package qasmith

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/quvly/Qasmith/internal/lower"
	"github.com/quvly/Qasmith/internal/modules"
	"github.com/quvly/Qasmith/internal/semantic"
	"github.com/quvly/Qasmith/internal/stdlib"
	"github.com/quvly/Qasmith/internal/syntax"
	"github.com/quvly/Qasmith/pkg/extension"
)

// SourcePosition uses one-based line and UTF-8 byte column coordinates.
type SourcePosition = syntax.Pos
type SourceRange struct{ Start, End SourcePosition }
type EditorToken = syntax.Lexeme

type EditorParameter struct{ Name, Type string }
type EditorSymbol struct {
	ID, Name, Type, Documentation string
	Parameters                    []EditorParameter
	Function, Public, Variadic    bool
	Definition, Range             SourceRange
}
type EditorOccurrence struct {
	Symbol      EditorSymbol
	Range       SourceRange
	Declaration bool
}
type EditorBinding struct {
	Symbol EditorSymbol
	Scope  SourceRange
}
type EditorImport struct {
	Path    string
	At      SourcePosition
	Exports []EditorSymbol
}
type EditorDocument struct {
	Source      string
	Tokens      []EditorToken
	Functions   []EditorSymbol
	Bindings    []EditorBinding
	Occurrences []EditorOccurrence
	Imports     []EditorImport
}
type EditorAnalysis struct {
	Entry       string
	Documents   map[string]*EditorDocument
	Diagnostics []error
	Types       []string
}
type cachedParse struct {
	text   string
	module *syntax.Module
	err    error
	tokens []syntax.Lexeme
}

// EditorSession caches compiler parses across successive overlay analyses.
// Analysis calls are serialized; snapshots can be read independently.
type EditorSession struct {
	Compiler Compiler
	mu       sync.Mutex
	cache    map[string]cachedParse
}

// EditorPath resolves source identity, including an unsaved new file.
func EditorPath(path string) (string, error) { return modules.EditorPath(path) }

func (s *EditorSession) parse(path, text string) cachedParse {
	if c, ok := s.cache[path]; ok && c.text == text {
		return c
	}
	m, err := syntax.ParseEditor(path, text)
	c := cachedParse{text: text, module: m, err: err, tokens: syntax.Lexemes(path, text)}
	if len(s.cache) >= 128 {
		clear(s.cache)
	}
	if s.cache == nil {
		s.cache = make(map[string]cachedParse)
	}
	s.cache[path] = c
	return c
}

// Analyze runs the real compiler against source overlays. The editor path
// retains partial symbols and never invokes value or circuit production callbacks.
func (s *EditorSession) Analyze(ctx context.Context, entry string, overlays map[string]string) (EditorAnalysis, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return EditorAnalysis{}, err
	}
	limits, err := s.Compiler.Limits.Resolve()
	if err != nil {
		return EditorAnalysis{}, err
	}
	registry := s.Compiler.Registry.Clone()
	if err := stdlib.Register(registry); err != nil {
		return EditorAnalysis{}, err
	}
	normalized := make(map[string]string, len(overlays))
	for path, source := range overlays {
		if actual, err := EditorPath(path); err == nil {
			normalized[actual] = source
		}
	}
	graph, err := modules.LoadEditor(entry, registry.HasModule, &modules.EditorSources{Context: ctx, Overlay: normalized, Parse: func(path, text string) (*syntax.Module, error) {
		c := s.parse(path, text)
		return c.module, c.err
	}})
	if err != nil {
		return EditorAnalysis{}, err
	}
	data := &semantic.EditorData{Context: ctx}
	program, err := semantic.AnalyzeEditor(graph, registry, limits, data)
	if err != nil {
		return EditorAnalysis{}, err
	}
	out := EditorAnalysis{Entry: graph.Entry.Path, Documents: make(map[string]*EditorDocument),
		Types:       []string{"Int", "Float", "Bool", "Void", "Qubit", "QReg", "MeasuredResult"},
		Diagnostics: append(slices.Clone(graph.Issues), data.Issues...)}
	for _, typ := range registry.TypeNames() {
		out.Types = append(out.Types, string(typ))
	}
	if program.Entry != nil && len(out.Diagnostics) == 0 && program.Static {
		if _, err := lower.ExecuteEditor(ctx, program, limits); err != nil {
			out.Diagnostics = append(out.Diagnostics, err)
		}
	}
	if err := ctx.Err(); err != nil {
		return EditorAnalysis{}, err
	}
	for _, m := range graph.Modules {
		text := graph.Sources[m.Path]
		doc := &EditorDocument{Source: text, Tokens: slices.Clone(s.parse(m.Path, text).tokens)}
		out.Documents[m.Path] = doc
		for _, f := range m.Functions {
			resolved := data.Scopes[m.Path][f.Name]
			if resolved == nil || resolved.Declaration != f {
				doc.Functions = append(doc.Functions, functionSymbol(f.Name, &semantic.Function{Declaration: f, Signature: extension.Signature{Return: extension.Type(f.Result)}}))
				continue
			}
			symbol := functionSymbol(f.Name, resolved)
			doc.Functions = append(doc.Functions, symbol)
			doc.Occurrences = append(doc.Occurrences, EditorOccurrence{symbol, symbol.Definition, true})
		}
		for name, f := range semantic.BuiltinFunctions() {
			if data.Scopes[m.Path][name] == nil {
				doc.Bindings = append(doc.Bindings, EditorBinding{Symbol: functionSymbol(name, f)})
			}
		}
		for name, f := range data.Scopes[m.Path] {
			doc.Bindings = append(doc.Bindings, EditorBinding{Symbol: functionSymbol(name, f)})
		}
		for i, decl := range m.Imports {
			im := EditorImport{Path: decl.Path, At: decl.At}
			if dep := graph.Edges[m][i]; dep != nil {
				for _, f := range dep.Functions {
					if resolved := data.Scopes[dep.Path][f.Name]; f.Public && resolved != nil && resolved.Declaration == f {
						im.Exports = append(im.Exports, functionSymbol(f.Name, resolved))
					}
				}
			} else {
				for name, fn := range registry.ModuleExports(decl.Path) {
					im.Exports = append(im.Exports, functionSymbol(name, &semantic.Function{Trusted: &fn, Namespace: decl.Path, Signature: extension.Signature{Return: fn.Return}}))
				}
			}
			doc.Imports = append(doc.Imports, im)
			for _, name := range decl.Names {
				for _, part := range []struct {
					name string
					at   SourcePosition
				}{{name.Name, name.At}, {name.Alias, name.AliasAt}} {
					if f := data.Imports[part.at]; f != nil {
						doc.Occurrences = append(doc.Occurrences, EditorOccurrence{functionSymbol(part.name, f), nameRange(part.at, part.name), false})
					}
					if name.At == name.AliasAt {
						break
					}
				}
			}
		}
	}
	for _, local := range data.Locals {
		doc := out.Documents[local.At.File]
		symbol := localSymbol(local)
		doc.Bindings = append(doc.Bindings, EditorBinding{symbol, SourceRange{local.Start, local.End}})
		doc.Occurrences = append(doc.Occurrences, EditorOccurrence{symbol, symbol.Definition, true})
	}
	for ref, local := range data.References {
		out.Documents[ref.At.File].Occurrences = append(out.Documents[ref.At.File].Occurrences, EditorOccurrence{localSymbol(local), nameRange(ref.At, ref.Name), false})
	}
	for call, f := range program.Targets {
		doc := out.Documents[call.At.File]
		doc.Occurrences = append(doc.Occurrences, EditorOccurrence{functionSymbol(call.Name, f), nameRange(call.At, call.Name), false})
	}
	return out, nil
}

func nameRange(at SourcePosition, name string) SourceRange {
	end := at
	end.Column += len(name)
	return SourceRange{at, end}
}
func localSymbol(local semantic.Local) EditorSymbol {
	return EditorSymbol{ID: "local/" + local.At.String(), Name: local.Name, Type: string(local.Type), Definition: nameRange(local.At, local.Name)}
}
func functionSymbol(name string, f *semantic.Function) EditorSymbol {
	s := EditorSymbol{Name: name, Function: true, Type: string(f.Signature.Return)}
	names := []string(nil)
	if decl := f.Declaration; decl != nil {
		s.ID = "fn/" + decl.NameAt.String()
		s.Public = decl.Public
		s.Definition = nameRange(decl.NameAt, decl.Name)
		end := decl.Body.End
		end.Column++
		s.Range = SourceRange{decl.At, end}
		for _, p := range decl.Params {
			names = append(names, p.Name)
		}
	}
	if f.Builtin != "" {
		s.ID = "builtin/" + f.Builtin
		meta := builtinMetadata[f.Builtin]
		names, s.Documentation = meta.names, meta.doc
		s.Variadic = f.Builtin == "measure"
	}
	if f.Trusted != nil {
		s.ID = f.Namespace + "/" + f.Trusted.Name
		s.Documentation = f.Trusted.Documentation
		for i, p := range f.Trusted.Parameters {
			s.Parameters = append(s.Parameters, EditorParameter{parameterName(f.Trusted.ParameterNames, i), p.String()})
		}
		return s
	}
	if f.Intrinsic != nil {
		s.ID = "host:" + f.Intrinsic.Namespace + "/" + f.Intrinsic.Name
	}
	for i, p := range f.Signature.Parameters {
		s.Parameters = append(s.Parameters, EditorParameter{parameterName(names, i), string(p)})
	}
	if f.Builtin == "measure" {
		s.Parameters = []EditorParameter{{"qubits", "Qubit… or QReg"}}
	}
	if f.Builtin == "len" {
		s.Parameters = []EditorParameter{{"value", "array or QReg"}}
	}
	return s
}
func parameterName(names []string, i int) string {
	if i < len(names) && names[i] != "" {
		return names[i]
	}
	return fmt.Sprintf("arg%d", i+1)
}

var builtinMetadata = map[string]struct {
	names []string
	doc   string
}{
	"qubit":   {nil, "Allocates one qubit in a quantum entry point."},
	"qreg":    {[]string{"size"}, "Allocates a fixed-size quantum register in a quantum entry point."},
	"len":     {nil, "Returns a fixed array or register length."},
	"h":       {[]string{"q"}, "Applies a Hadamard gate."},
	"x":       {[]string{"q"}, "Applies a Pauli X gate."},
	"cx":      {[]string{"control", "target"}, "Applies a controlled X gate to distinct qubits."},
	"rx":      {[]string{"theta", "q"}, "Rotates around the X axis, in radians."},
	"ry":      {[]string{"theta", "q"}, "Rotates around the Y axis, in radians."},
	"rz":      {[]string{"theta", "q"}, "Rotates around the Z axis, in radians."},
	"measure": {nil, "Measures qubits in argument order in the final return."},
}

func (s EditorSymbol) Signature() string {
	if !s.Function {
		return s.Name + ": " + s.Type
	}
	parts := make([]string, len(s.Parameters))
	for i, p := range s.Parameters {
		parts[i] = p.Name + ": " + p.Type
	}
	return s.Name + "(" + strings.Join(parts, ", ") + ") -> " + s.Type
}
