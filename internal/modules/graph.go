package modules

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/quvly/Qasmith/internal/corelib"
	"github.com/quvly/Qasmith/internal/diagnostic"
	"github.com/quvly/Qasmith/internal/syntax"
)

type Graph struct {
	syntax.Program
	Root    string
	Edges   map[*syntax.Module][]*syntax.Module
	Trusted map[*syntax.Module]map[int]string
	Issues  []error
	Sources map[string]string
}

func Load(entry string) (*Graph, error) { return LoadWithRegistry(entry, nil) }

func LoadWithRegistry(entry string, registered func(string) bool) (*Graph, error) {
	return loadGraph(entry, registered, nil)
}

type EditorSources struct {
	Context context.Context
	Overlay map[string]string
	Parse   func(string, string) (*syntax.Module, error)
}

func LoadEditor(entry string, registered func(string) bool, editor *EditorSources) (*Graph, error) {
	return loadGraph(entry, registered, editor)
}

func loadGraph(entry string, registered func(string) bool, editor *EditorSources) (*Graph, error) {
	path, err := sourcePath(entry, editor)
	if err != nil {
		return nil, diagnostic.At(filePos(entry), "E2000", "open entry: %v", err)
	}
	root, deps, err := project(filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	loader := loader{
		graph:      &Graph{Root: root, Edges: make(map[*syntax.Module][]*syntax.Module), Trusted: make(map[*syntax.Module]map[int]string), Sources: make(map[string]string)},
		editor:     editor,
		registered: registered,
		spellings:  make(map[string][]string),
		deps:       deps, cache: make(map[string]*syntax.Module), roots: make(map[string]string),
	}
	main, err := loader.load(path, loader.packageRoot(root, path), filePos(path))
	if err != nil {
		return nil, err
	}
	loader.graph.Entry = main
	return loader.graph, nil
}

type loader struct {
	graph      *Graph
	deps       map[string]string
	cache      map[string]*syntax.Module
	roots      map[string]string
	stack      []string
	spellings  map[string][]string
	registered func(string) bool
	editor     *EditorSources
}

func filePos(path string) syntax.Pos { return syntax.Pos{File: path, Line: 1, Column: 1} }

func canonical(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(abs)
}

// EditorPath canonicalizes a file path, including a newly created file whose
// parent already exists. The resolver still requires disk or overlay contents.
func EditorPath(path string) (string, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	real, err := filepath.EvalSymlinks(path)
	if !os.IsNotExist(err) {
		return real, err
	}
	if _, linkErr := os.Lstat(path); !os.IsNotExist(linkErr) {
		return "", err
	}
	parent, parentErr := filepath.EvalSymlinks(filepath.Dir(path))
	if parentErr != nil {
		return "", parentErr
	}
	return filepath.Join(parent, filepath.Base(path)), nil
}

func sourcePath(path string, editor *EditorSources) (string, error) {
	if editor == nil {
		return canonical(path)
	}
	actual, err := EditorPath(path)
	if err == nil {
		if _, ok := editor.Overlay[actual]; ok {
			return actual, nil
		}
	}
	return canonical(path)
}

func within(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return true
	}
	for ancestor := path; len(ancestor) >= len(root); ancestor = filepath.Dir(ancestor) {
		if strings.EqualFold(ancestor, root) {
			a, aErr := os.Lstat(ancestor)
			b, bErr := os.Stat(root)
			return aErr == nil && bErr == nil && os.SameFile(a, b)
		}
		if filepath.Dir(ancestor) == ancestor {
			break
		}
	}
	return false
}

func (l *loader) packageRoot(root, path string) string {
	for _, dep := range l.deps {
		if within(root, dep) && within(dep, path) {
			return dep
		}
	}
	return root
}

func (l *loader) load(path, root string, at syntax.Pos) (*syntax.Module, error) {
	folded := strings.ToLower(path)
	for _, existing := range l.spellings[folded] {
		if existing == path {
			break
		}
		a, aErr := os.Stat(existing)
		b, bErr := os.Stat(path)
		if aErr == nil && bErr == nil && os.SameFile(a, b) {
			path = existing
			break
		}
	}
	if i := slices.Index(l.stack, path); i >= 0 {
		cycle := append(slices.Clone(l.stack[i:]), path)
		return nil, diagnostic.At(at, "E2002", "circular module dependency: %s", strings.Join(cycle, " imports "))
	}
	if m := l.cache[path]; m != nil {
		if l.roots[path] != root {
			return nil, diagnostic.At(at, "E2003", "module %s has conflicting package roots", path)
		}
		return m, nil
	}
	var source string
	var overlay bool
	if corelib.IsCorePath(path) {
		name := strings.TrimSuffix(strings.TrimPrefix(path, corelib.VirtualRoot), ".smith")
		src, err := corelib.Read(name)
		if err != nil {
			return nil, diagnostic.At(at, "E2000", "load core module %q: %v", name, err)
		}
		source = src
		overlay = true
	} else if l.editor != nil {
		if err := l.editor.Context.Err(); err != nil {
			return nil, err
		}
		if len(l.graph.Sources) >= 128 {
			return nil, diagnostic.At(at, "E2000", "editor module limit exceeded")
		}
		source, overlay = l.editor.Overlay[path]
	}
	if !overlay {
		info, err := os.Stat(path)
		if err != nil || !info.Mode().IsRegular() || filepath.Ext(path) != ".smith" {
			return nil, diagnostic.At(at, "E2000", "expected a regular .smith file at %s", path)
		}
		if l.editor != nil && info.Size() > 1<<20 {
			return nil, diagnostic.At(at, "E2000", "editor source exceeds 1 MiB")
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, diagnostic.At(at, "E2000", "read module: %v", err)
		}
		source = string(data)
	}
	if l.editor != nil && !corelib.IsCorePath(path) && len(source) > 1<<20 {
		return nil, diagnostic.At(at, "E2000", "editor source exceeds 1 MiB")
	}
	parse := syntax.Parse
	if l.editor != nil && !corelib.IsCorePath(path) {
		parse = l.editor.Parse
		l.graph.Sources[path] = source
	}
	m, err := parse(path, source)
	if err != nil {
		if l.editor == nil {
			return nil, err
		}
		l.graph.Issues = append(l.graph.Issues, err)
		if m == nil {
			m = &syntax.Module{Path: path}
		}
	}

	l.spellings[folded] = append(l.spellings[folded], path)
	l.stack = append(l.stack, path)
	defer func() { l.stack = l.stack[:len(l.stack)-1] }()
	edges := make([]*syntax.Module, len(m.Imports))
	for i, decl := range m.Imports {
		if l.registered != nil && l.registered(decl.Path) {
			if l.graph.Trusted[m] == nil {
				l.graph.Trusted[m] = make(map[int]string)
			}
			l.graph.Trusted[m][i] = decl.Path
			continue
		}
		if strings.HasPrefix(decl.Path, "std/") && l.registered != nil {
			if err := l.issue(diagnostic.At(decl.At, "E2000", "unregistered standard-library module %q", decl.Path)); err != nil {
				return nil, err
			}
			continue
		}
		if strings.HasPrefix(decl.Path, "host:") {
			continue
		}
		target, targetRoot, err := l.resolve(path, root, decl.Path)
		if err != nil {
			if err := l.issue(diagnostic.At(decl.At, "E2000", "resolve %q: %v", decl.Path, err)); err != nil {
				return nil, err
			}
			continue
		}
		edges[i], err = l.load(target, targetRoot, decl.At)
		if err := l.issue(err); err != nil {
			return nil, err
		}
	}
	l.cache[path], l.roots[path] = m, root
	l.graph.Edges[m] = edges
	l.graph.Modules = append(l.graph.Modules, m)
	return m, nil
}

func (l *loader) issue(err error) error {
	if err == nil || l.editor == nil {
		return err
	}
	if l.editor.Context.Err() != nil {
		return err
	}
	l.graph.Issues = append(l.graph.Issues, err)
	return nil
}

func (l *loader) resolve(importer, root, spec string) (string, string, error) {
	if strings.ContainsAny(spec, "\\\x00") || filepath.IsAbs(spec) {
		return "", "", fmt.Errorf("invalid module path")
	}

	// Core library: import { ... } from "core/module"
	if strings.HasPrefix(spec, "core/") || spec == "core" {
		name := corelib.ModuleName(spec)
		if name == "" {
			return "", "", fmt.Errorf("invalid core module path %q (expected core/<module>)", spec)
		}
		if !corelib.Exists(name) {
			return "", "", fmt.Errorf("core module %q not found; available: %s", spec, strings.Join(corelib.ListModules(), ", "))
		}
		return corelib.VirtualPath(name), corelib.VirtualRoot, nil
	}

	// Relative imports from within a core module resolve inside the core root
	if corelib.IsCorePath(importer) && (strings.HasPrefix(spec, "./") || strings.HasPrefix(spec, "../")) {
		candidate := filepath.Join(filepath.Dir(importer), filepath.FromSlash(spec))
		if filepath.Ext(candidate) == "" {
			candidate += ".smith"
		}
		if !corelib.IsCorePath(candidate) {
			return "", "", fmt.Errorf("path escapes core root")
		}
		name := strings.TrimSuffix(strings.TrimPrefix(candidate, corelib.VirtualRoot), ".smith")
		if !corelib.Exists(name) {
			return "", "", fmt.Errorf("core module %q not found", candidate)
		}
		return candidate, corelib.VirtualRoot, nil
	}

	var candidate string
	if strings.HasPrefix(spec, "./") || strings.HasPrefix(spec, "../") {
		candidate = filepath.Join(filepath.Dir(importer), filepath.FromSlash(spec))
		root = l.packageRoot(root, candidate)
	} else {
		pkg, sub, ok := strings.Cut(spec, "/")
		if pkg == "std" {
			return "", "", fmt.Errorf("std namespace is reserved")
		}
		dep, declared := l.deps[pkg]
		if !declared {
			return "", "", fmt.Errorf("undeclared package %q", pkg)
		}
		if !ok || sub == "" || strings.HasPrefix(sub, "/") {
			return "", "", fmt.Errorf("expected package/module")
		}
		root = dep
		if filepath.Ext(sub) == "" {
			sub += ".smith"
		}
		candidate = filepath.Join(root, filepath.FromSlash(sub))
	}
	if !within(root, candidate) {
		return "", "", fmt.Errorf("path escapes root %s", root)
	}
	target, err := sourcePath(candidate, l.editor)
	if err != nil {
		return "", "", err
	}
	if !within(root, target) {
		return "", "", fmt.Errorf("path escapes root %s", root)
	}
	return target, l.packageRoot(root, target), nil
}

// Walk visits source modules in import order, marking shared dependencies.
func (g *Graph) Walk(visit func(path, prefix string, repeated bool)) {
	seen := make(map[*syntax.Module]bool)
	trustedSeen := make(map[string]bool)
	var walk func(*syntax.Module, string, string)
	walk = func(m *syntax.Module, label, prefix string) {
		path, err := filepath.Rel(g.Root, m.Path)
		if err != nil {
			path = m.Path
		}
		visit(filepath.ToSlash(path), label, seen[m])
		if seen[m] {
			return
		}
		seen[m] = true
		var children []int
		for i, child := range g.Edges[m] {
			if child != nil || g.Trusted[m][i] != "" {
				children = append(children, i)
			}
		}
		for i, index := range children {
			branch, indent := "├── ", "│   "
			if i == len(children)-1 {
				branch, indent = "└── ", "    "
			}
			if child := g.Edges[m][index]; child != nil {
				walk(child, prefix+branch, prefix+indent)
			} else {
				name := g.Trusted[m][index]
				visit(name, prefix+branch, trustedSeen[name])
				trustedSeen[name] = true
			}
		}
	}
	walk(g.Entry, "", "")
}
