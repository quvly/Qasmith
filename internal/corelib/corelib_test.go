package corelib

import (
	"sort"
	"testing"
)

func TestReadCoreModule(t *testing.T) {
	modules := ListModules()
	if len(modules) == 0 {
		t.Fatal("no core modules found")
	}
	for _, name := range modules {
		src, err := Read(name)
		if err != nil {
			t.Errorf("Read(%q) failed: %v", name, err)
			continue
		}
		if src == "" {
			t.Errorf("Read(%q) returned empty source", name)
		}
	}
}

func TestExists(t *testing.T) {
	if !Exists("rotations") {
		t.Error("Expected rotations module to exist")
	}
	if !Exists("gates") {
		t.Error("Expected gates module to exist")
	}
	if Exists("nonexistent") {
		t.Error("Expected nonexistent module to not exist")
	}
}

func TestModuleName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"core/rotations", "rotations"},
		{"core/gates", "gates"},
		{"core/math", "math"},
		{"core/rotations.smith", "rotations"},
		{"core/", ""},
		{"core", ""},
		{"std/optimization/tsp", ""},
		{"other/module", ""},
		{"core/sub/module", ""},
	}
	for _, tc := range tests {
		got := ModuleName(tc.input)
		if got != tc.want {
			t.Errorf("ModuleName(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestIsCorePath(t *testing.T) {
	if !IsCorePath("<core>/rotations.smith") {
		t.Error("Expected <core>/rotations.smith to be a core path")
	}
	if IsCorePath("/path/to/file.smith") {
		t.Error("Expected /path/to/file.smith to not be a core path")
	}
	if IsCorePath("core/rotations") {
		t.Error("Expected core/rotations to not be a core path (it's an import specifier)")
	}
}

func TestVirtualPath(t *testing.T) {
	got := VirtualPath("rotations")
	want := "<core>/rotations.smith"
	if got != want {
		t.Errorf("VirtualPath(rotations) = %q, want %q", got, want)
	}
}

func TestListModules(t *testing.T) {
	modules := ListModules()
	if len(modules) == 0 {
		t.Fatal("no core modules found")
	}

	// Check expected modules are present
	expected := []string{"rotations", "gates", "math", "utils", "measurement"}
	for _, e := range expected {
		found := false
		for _, m := range modules {
			if m == e {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected core module %q not found in %v", e, modules)
		}
	}

	// Verify sorted
	sorted := make([]string, len(modules))
	copy(sorted, modules)
	sort.Strings(sorted)
	for i := range modules {
		if modules[i] != sorted[i] {
			t.Errorf("ListModules not sorted: %v", modules)
			break
		}
	}
}
