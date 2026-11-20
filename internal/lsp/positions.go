package lsp

import (
	"net/url"
	"path/filepath"
	"runtime"
	"strings"
	"unicode/utf8"

	"github.com/quvly/Qasmith/pkg/qasmith"
)

type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}
type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

func fileURI(path string) string {
	slash := filepath.ToSlash(path)
	if host, tail, ok := strings.Cut(strings.TrimPrefix(slash, "//"), "/"); strings.HasPrefix(slash, "//") && ok {
		return (&url.URL{Scheme: "file", Host: host, Path: "/" + tail}).String()
	}
	if !strings.HasPrefix(slash, "/") {
		slash = "/" + slash
	}
	return (&url.URL{Scheme: "file", Path: slash}).String()
}

func pathFromURI(uri string) (string, bool) {
	u, err := url.Parse(uri)
	if err != nil || u.Scheme != "file" || u.RawQuery != "" || u.Fragment != "" {
		return "", false
	}
	path := u.Path
	if runtime.GOOS == "windows" {
		if u.Host != "" && u.Host != "localhost" {
			path = "//" + u.Host + path
		} else if len(path) > 2 && path[0] == '/' && path[2] == ':' {
			path = path[1:]
		}
	} else if u.Host != "" && u.Host != "localhost" {
		return "", false
	}
	path = filepath.FromSlash(path)
	if !filepath.IsAbs(path) {
		return "", false
	}
	actual, err := qasmith.EditorPath(path)
	if err != nil {
		actual = filepath.Clean(path)
	}
	return actual, true
}

func lineText(source string, line int) string {
	if line < 0 {
		return ""
	}
	for range line {
		i := strings.IndexByte(source, '\n')
		if i < 0 {
			return ""
		}
		source = source[i+1:]
	}
	if i := strings.IndexByte(source, '\n'); i >= 0 {
		source = source[:i]
	}
	return strings.TrimSuffix(source, "\r")
}

func toPosition(source string, p qasmith.SourcePosition) Position {
	line := max(0, p.Line-1)
	row := lineText(source, line)
	byteColumn := min(len(row), max(0, p.Column-1))
	units := 0
	for i, r := range row {
		if i >= byteColumn {
			break
		}
		units++
		if r > 0xffff {
			units++
		}
	}
	return Position{line, units}
}

func fromPosition(source, path string, p Position) qasmith.SourcePosition {
	row := lineText(source, p.Line)
	units := 0
	offset := len(row)
	for i, r := range row {
		width := 1
		if r > 0xffff {
			width++
		}
		if units+width > p.Character {
			offset = i
			break
		}
		units += width
	}
	return qasmith.SourcePosition{File: path, Line: p.Line + 1, Column: offset + 1}
}

func toRange(source string, r qasmith.SourceRange) Range {
	return Range{toPosition(source, r.Start), toPosition(source, r.End)}
}

func before(a, b qasmith.SourcePosition) bool {
	return a.Line < b.Line || a.Line == b.Line && a.Column < b.Column
}

func diagnosticRange(source string, p qasmith.SourcePosition) Range {
	row := lineText(source, p.Line-1)
	i := min(len(row), max(0, p.Column-1))
	end := p
	if i < len(row) {
		r, n := utf8.DecodeRuneInString(row[i:])
		end.Column += n
		if r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			for _, r := range row[i+n:] {
				if !(r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') {
					break
				}
				end.Column++
			}
		}
	}
	return toRange(source, qasmith.SourceRange{Start: p, End: end})
}
