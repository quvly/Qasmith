package lsp

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/quvly/Qasmith/pkg/qasmith"
)

func (s *server) feature(ctx context.Context, method string, params documentParams) (any, error) {
	path, ok := pathFromURI(params.TextDocument.URI)
	if !ok {
		return nil, fmt.Errorf("expected a file URI")
	}
	analysis, ok := s.cache.analyses[path]
	if !ok || s.cache.revision != s.revision {
		var err error
		analysis, err = s.editor.Analyze(ctx, path, overlays(s.docs))
		if err != nil {
			return nil, err
		}
	}
	doc := analysis.Documents[path]
	if doc == nil {
		doc = analysis.Documents[analysis.Entry]
	}
	if doc == nil {
		return nil, fmt.Errorf("source is not available")
	}
	at := fromPosition(doc.Source, path, params.Position)
	switch method {
	case "textDocument/completion":
		return completion(analysis, doc, at), nil
	case "textDocument/documentSymbol":
		result := make([]any, 0, len(doc.Functions))
		for _, symbol := range doc.Functions {
			r := toRange(doc.Source, symbol.Range)
			selection := toRange(doc.Source, symbol.Definition)
			if !s.hierarchicalSymbols {
				result = append(result, map[string]any{"name": symbol.Name, "kind": 12, "location": Location{s.uri(path), selection}})
				continue
			}
			if r.Start.Line == selection.Start.Line && r.Start.Character > selection.Start.Character {
				r.Start = selection.Start
			}
			if before(symbol.Range.End, symbol.Definition.End) {
				r.End = selection.End
			}
			result = append(result, map[string]any{"name": symbol.Name, "detail": symbol.Signature(), "kind": 12, "range": r, "selectionRange": selection})
		}
		return result, nil
	case "textDocument/signatureHelp":
		return signatureHelp(doc, at), nil
	}
	symbol, selected, ok := symbolAt(doc, at)
	if !ok {
		return nil, nil
	}
	switch method {
	case "textDocument/hover":
		text := symbol.Signature()
		if symbol.Documentation != "" {
			text += "\n\n" + symbol.Documentation
		}
		return map[string]any{"contents": map[string]string{"kind": "plaintext", "value": text}, "range": toRange(doc.Source, selected)}, nil
	case "textDocument/definition":
		r := symbol.Definition
		if r.Start.File == "" {
			return nil, nil
		}
		text := sourceFor(analysis, r.Start.File, s)
		return Location{s.uri(r.Start.File), toRange(text, r)}, nil
	case "textDocument/references":
		return s.references(ctx, analysis, symbol.ID, params.Context.IncludeDeclaration)
	}
	return nil, nil
}

func sourceFor(analysis qasmith.EditorAnalysis, path string, s *server) string {
	if doc := analysis.Documents[path]; doc != nil {
		return doc.Source
	}
	return s.source(path)
}

func symbolAt(doc *qasmith.EditorDocument, at qasmith.SourcePosition) (qasmith.EditorSymbol, qasmith.SourceRange, bool) {
	for _, ref := range doc.Occurrences {
		if !before(at, ref.Range.Start) && before(at, ref.Range.End) {
			return ref.Symbol, ref.Range, true
		}
	}
	return qasmith.EditorSymbol{}, qasmith.SourceRange{}, false
}

func (s *server) references(ctx context.Context, initial qasmith.EditorAnalysis, id string, include bool) ([]Location, error) {
	results := []Location{}
	seen := make(map[Location]bool)
	add := func(a qasmith.EditorAnalysis) {
		for path, doc := range a.Documents {
			for _, ref := range doc.Occurrences {
				if ref.Symbol.ID != id || ref.Declaration && !include {
					continue
				}
				loc := Location{s.uri(path), toRange(doc.Source, ref.Range)}
				if !seen[loc] {
					seen[loc] = true
					results = append(results, loc)
				}
			}
		}
	}
	add(initial)
	for path := range s.docs {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if path == initial.Entry {
			continue
		}
		a, ok := s.cache.analyses[path]
		if !ok {
			var err error
			a, err = s.editor.Analyze(ctx, path, overlays(s.docs))
			if err != nil {
				continue
			}
		}
		add(a)
	}
	slices.SortFunc(results, func(a, b Location) int {
		if a.URI != b.URI {
			return strings.Compare(a.URI, b.URI)
		}
		if a.Range.Start.Line != b.Range.Start.Line {
			return a.Range.Start.Line - b.Range.Start.Line
		}
		return a.Range.Start.Character - b.Range.Start.Character
	})
	return results, nil
}

type completionItem struct {
	Label         string `json:"label"`
	Detail        string `json:"detail,omitempty"`
	Documentation string `json:"documentation,omitempty"`
	Kind          int    `json:"kind"`
}

func completion(a qasmith.EditorAnalysis, doc *qasmith.EditorDocument, at qasmith.SourcePosition) []completionItem {
	var result []completionItem
	addSymbol := func(s qasmith.EditorSymbol) {
		kind := 6
		if s.Function {
			kind = 3
		}
		result = append(result, completionItem{s.Name, s.Signature(), s.Documentation, kind})
	}
	if im := importContext(doc, at); im != nil {
		for _, s := range im.Exports {
			addSymbol(s)
		}
	} else {
		token := previousToken(doc, at)
		switch {
		case token == ":" || token == "->":
			for _, typ := range a.Types {
				result = append(result, completionItem{Label: typ, Kind: 7})
			}
		default:
			f := enclosing(doc, at)
			if f == nil {
				for _, word := range []string{"import", "pub", "fn"} {
					result = append(result, completionItem{Label: word, Kind: 14})
				}
				break
			}
			for _, b := range visibleBindings(doc, at) {
				if f.Type != "MeasuredResult" && f.Type != "Void" && (b.Type == "Void" || b.Type == "Qubit" || b.Type == "MeasuredResult" || b.Type == "QReg") {
					continue
				}
				addSymbol(b)
			}
			if token == "{" || token == ";" || token == "}" {
				for _, word := range []string{"if", "for", "let", "return"} {
					result = append(result, completionItem{Label: word, Kind: 14})
				}
			}
		}
	}
	slices.SortFunc(result, func(a, b completionItem) int { return strings.Compare(a.Label, b.Label) })
	if result == nil {
		return []completionItem{}
	}
	return result
}

func importContext(doc *qasmith.EditorDocument, at qasmith.SourcePosition) *qasmith.EditorImport {
	for i := range doc.Imports {
		im := &doc.Imports[i]
		if before(at, im.At) {
			continue
		}
		open := false
		for _, t := range doc.Tokens {
			if before(t.At, im.At) {
				continue
			}
			if t.Kind == "from" || t.Kind == ";" {
				break
			}
			if t.Kind == "{" {
				open = before(t.At, at)
			}
			if t.Kind == "}" && open && !before(t.At, at) {
				return im
			}
			if t.Kind == "}" {
				break
			}
		}
	}
	return nil
}

func previousToken(doc *qasmith.EditorDocument, at qasmith.SourcePosition) string {
	var previous string
	for _, t := range doc.Tokens {
		if !before(t.At, at) {
			break
		}
		if t.Kind == "identifier" && t.At.Line == at.Line && t.At.Column+len(t.Value) >= at.Column {
			break
		}
		if t.Kind != "end of file" {
			previous = t.Kind
		}
	}
	return previous
}

func enclosing(doc *qasmith.EditorDocument, at qasmith.SourcePosition) *qasmith.EditorSymbol {
	for i := range doc.Functions {
		f := &doc.Functions[i]
		if !before(at, f.Range.Start) && !before(f.Range.End, at) {
			return f
		}
	}
	return nil
}

func visibleBindings(doc *qasmith.EditorDocument, at qasmith.SourcePosition) map[string]qasmith.EditorSymbol {
	result := make(map[string]qasmith.EditorSymbol)
	for _, binding := range doc.Bindings {
		r := binding.Scope
		if r.Start.Line == 0 || before(r.Start, at) && !before(r.End, at) {
			result[binding.Symbol.Name] = binding.Symbol
		}
	}
	return result
}

func signatureHelp(doc *qasmith.EditorDocument, at qasmith.SourcePosition) any {
	type frame struct {
		kind, name string
		args       int
	}
	var stack []frame
	var previous, prior qasmith.EditorToken
	for _, tok := range doc.Tokens {
		if !before(tok.At, at) {
			break
		}
		switch tok.Kind {
		case "(", "[", "{":
			name := ""
			if tok.Kind == "(" && previous.Kind == "identifier" && prior.Kind != "fn" {
				name = previous.Value
			}
			stack = append(stack, frame{kind: tok.Kind, name: name})
		case ")", "]", "}":
			if len(stack) > 0 && strings.Index("([{", stack[len(stack)-1].kind) == strings.Index(")]}", tok.Kind) {
				stack = stack[:len(stack)-1]
			}
		case ",":
			if len(stack) > 0 {
				stack[len(stack)-1].args++
			}
		}
		prior, previous = previous, tok
	}
	bindings := visibleBindings(doc, at)
	for i := len(stack) - 1; i >= 0; i-- {
		f := stack[i]
		if f.name == "" {
			continue
		}
		s := bindings[f.name]
		if !s.Function {
			continue
		}
		parameters := make([]map[string]string, 0, len(s.Parameters))
		for _, p := range s.Parameters {
			parameters = append(parameters, map[string]string{"label": p.Name + ": " + p.Type})
		}
		active := min(f.args, max(0, len(parameters)-1))
		return map[string]any{"signatures": []any{map[string]any{"label": s.Signature(), "documentation": s.Documentation, "parameters": parameters}},
			"activeSignature": 0, "activeParameter": active}
	}
	return nil
}
