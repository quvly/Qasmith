package lsp

import (
	"errors"
	"slices"

	"github.com/quvly/Qasmith/pkg/qasmith"
)

type diagnostic struct {
	Range    Range  `json:"range"`
	Severity int    `json:"severity"`
	Code     string `json:"code"`
	Source   string `json:"source"`
	Message  string `json:"message"`
}

func (s *server) publish(result findings) {
	if result.revision != s.revision {
		return
	}
	grouped := make(map[string][]diagnostic)
	seen := make(map[string]map[diagnostic]bool)
	add := func(path, source string, err error) {
		d := diagnostic{Severity: 1, Code: "E2000", Source: "qasmith", Message: err.Error()}
		var detail *qasmith.Diagnostic
		if errors.As(err, &detail) {
			d.Range = diagnosticRange(source, detail.Position)
			d.Code, d.Message = detail.Code, detail.Message
		}
		if seen[path] == nil {
			seen[path] = make(map[diagnostic]bool)
		}
		if !seen[path][d] {
			seen[path][d] = true
			grouped[path] = append(grouped[path], d)
		}
	}
	for path := range s.docs {
		grouped[path] = []diagnostic{}
	}
	for path, a := range result.analyses {
		for _, err := range a.Diagnostics {
			target := path
			var detail *qasmith.Diagnostic
			if errors.As(err, &detail) && detail.Position.File != "" {
				target = detail.Position.File
			}
			add(target, sourceFor(a, target, s), err)
		}
	}
	for path, err := range result.failures {
		add(path, s.source(path), err)
	}
	for path := range s.previous {
		if _, present := grouped[path]; !present {
			s.notify("textDocument/publishDiagnostics", map[string]any{"uri": s.uri(path), "diagnostics": []any{}})
		}
	}
	clear(s.previous)
	for path, items := range grouped {
		slices.SortFunc(items, func(a, b diagnostic) int {
			if a.Range.Start.Line != b.Range.Start.Line {
				return a.Range.Start.Line - b.Range.Start.Line
			}
			return a.Range.Start.Character - b.Range.Start.Character
		})
		message := map[string]any{"uri": s.uri(path), "diagnostics": items}
		if doc, ok := s.docs[path]; ok && s.diagnosticVersions {
			message["version"] = doc.version
		}
		s.notify("textDocument/publishDiagnostics", message)
		s.previous[path] = true
	}
}
