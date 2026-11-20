package modules

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
)

func link(t *testing.T, target, path string) {
	t.Helper()
	if err := os.Symlink(target, path); err != nil {
		if runtime.GOOS == "windows" && errors.Is(err, syscall.Errno(1314)) {
			t.Skip("Windows runner does not grant symbolic-link privileges")
		}
		t.Fatal(err)
	}
}

func put(t *testing.T, root, name, source string) string {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCanonicalGraphCache(t *testing.T) {
	dir := t.TempDir()
	entry := put(t, dir, "main.smith", `import { a } from "./a.smith"; import { b } from "./sub/b.smith";`)
	put(t, dir, "a.smith", `import { c } from "./alias.smith"; pub fn a() -> Int { return c(); }`)
	put(t, dir, "sub/b.smith", `import { c } from "../shared.smith"; pub fn b() -> Int { return c(); }`)
	put(t, dir, "shared.smith", `pub fn c() -> Int { return 1; }`)
	link(t, filepath.Join(dir, "shared.smith"), filepath.Join(dir, "alias.smith"))
	g, err := Load(entry)
	if err != nil {
		t.Fatal(err)
	}
	a, b := g.Edges[g.Entry][0], g.Edges[g.Entry][1]
	if len(g.Modules) != 4 || g.Edges[a][0] != g.Edges[b][0] {
		t.Fatal("shared dependency was loaded twice")
	}
	var visits []string
	g.Walk(func(path, prefix string, repeat bool) {
		if repeat {
			path += " (shared)"
		}
		visits = append(visits, prefix+path)
	})
	want := "main.smith\n├── a.smith\n│   └── shared.smith\n└── sub/b.smith\n    └── shared.smith (shared)"
	if got := strings.Join(visits, "\n"); got != want {
		t.Fatalf("graph:\n%s", got)
	}
}

func TestBoundaries(t *testing.T) {
	outside := put(t, t.TempDir(), "outside.smith", `pub fn f() -> Int { return 1; }`)
	for _, tc := range []struct{ name, source, dep, want string }{
		{"absolute", `import { f } from "` + filepath.ToSlash(outside) + `";`, "", "invalid module path"},
		{"project traversal", `import { f } from "../outside.smith";`, "", "escapes root"},
		{"project symlink", `import { f } from "./escape.smith";`, "", "escapes root"},
		{"package traversal", `import { f } from "lib/../safe";`, "", "escapes root"},
		{"package relative traversal", `import { f } from "lib/main";`, `import { f } from "../safe.smith";`, "escapes root"},
		{"package symlink", `import { f } from "lib/escape";`, "", "escapes root"},
		{"relative package symlink", `import { f } from "./lib/escape.smith";`, "", "escapes root"},
		{"undeclared", `import { f } from "other/unit";`, "", "undeclared package"},
		{"reserved", `import { f } from "std/unit";`, "", "namespace is reserved"},
		{"missing", `import { f } from "./missing.smith";`, "", "missing.smith"},
		{"cycle", `import { f } from "lib/main";`, `import { f } from "./main.smith";`, "circular module dependency"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			put(t, root, "qasmith.json", `{"name":"app","version":"0.1.0","dependencies":{"lib":{"path":"./lib"}}}`)
			put(t, root, "lib/main.smith", tc.dep)
			safe := put(t, root, "safe.smith", `pub fn f() -> Int { return 1; }`)
			if tc.name == "project symlink" {
				link(t, outside, filepath.Join(root, "escape.smith"))
			}
			if tc.name == "package symlink" || tc.name == "relative package symlink" {
				link(t, safe, filepath.Join(root, "lib/escape.smith"))
			}
			_, err := Load(put(t, root, "main.smith", tc.source))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("wanted %q, got %v", tc.want, err)
			}
		})
	}
}

func TestManifestValidation(t *testing.T) {
	for _, body := range []string{
		`{"name":"app","name":"again","version":"0.1.0"}`,
		`{"name":"app","version":"invalid"}`,
		`{"name":"app","version":"0.1.0","unknown":1}`,
		`{"name":"app","version":"0.1.0"} {}`,
		`{"name":"app","version":"0.1.0","dependencies":{"std":{"path":"lib"}}}`,
		`{"name":"app","version":"0.1.0","dependencies":{"app":{"path":"lib"}}}`,
		`{"name":"app","version":"0.1.0","dependencies":{"lib":{"path":"missing"}}}`,
		`{"name":"app","version":"0.1.0","dependencies":{"lib":{"path":"module.smith"}}}`,
		`{"name":"app","version":"0.1.0","dependencies":{"lib":{"path":"lib"},"alias":{"path":"lib"}}}`,
	} {
		root := t.TempDir()
		put(t, root, "qasmith.json", body)
		put(t, root, "lib/dummy.smith", "")
		_, err := Load(put(t, root, "module.smith", ""))
		if err == nil || !strings.Contains(err.Error(), "E2100") {
			t.Errorf("expected invalid manifest for %s, got %v", body, err)
		}
	}
}

func TestExternalDependency(t *testing.T) {
	external := t.TempDir()
	put(t, external, "bell.smith", "pub fn f() -> Int { return 1; }")
	root := t.TempDir()
	put(t, root, "qasmith.json", `{"name":"app","version":"1.0.0","dependencies":{"external":{"path":"`+filepath.ToSlash(external)+`"}}}`)
	g, err := Load(put(t, root, "main.smith", `import { f } from "external/bell.smith";`))
	if err != nil || len(g.Modules) != 2 {
		t.Fatalf("resolve external dependency: %v", err)
	}
}
