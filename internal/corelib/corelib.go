// Package corelib provides access to the embedded Qasmith core standard library.
package corelib

import (
	"io/fs"
	"strings"

	qasmithStdlib "github.com/quvly/Qasmith/stdlib"
)

// VirtualRoot is the synthetic path prefix for core library modules.
// Core modules use paths like "<core>/rotations.smith" so they never
// collide with filesystem paths.
const VirtualRoot = "<core>/"

// Read returns the source code of a core module.
// The name should be the module name without the .smith extension,
// e.g., "rotations" or "gates".
func Read(name string) (string, error) {
	data, err := qasmithStdlib.CoreFS.ReadFile("core/" + name + ".smith")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Exists reports whether a core module with the given name exists.
func Exists(name string) bool {
	_, err := qasmithStdlib.CoreFS.ReadFile("core/" + name + ".smith")
	return err == nil
}

// IsCorePath reports whether path refers to a core library module.
func IsCorePath(path string) bool {
	return strings.HasPrefix(path, VirtualRoot)
}

// VirtualPath returns the virtual filesystem path for a core module name.
func VirtualPath(name string) string {
	return VirtualRoot + name + ".smith"
}

// ModuleName extracts the module name from a core/ import specifier.
// For example, "core/rotations" returns "rotations".
// Returns empty string if the specifier doesn't match core/*.
func ModuleName(importPath string) string {
	if !strings.HasPrefix(importPath, "core/") {
		return ""
	}
	name := strings.TrimPrefix(importPath, "core/")
	// Reject nested paths or empty names
	if strings.Contains(name, "/") || name == "" {
		return ""
	}
	// Strip .smith extension if present
	name = strings.TrimSuffix(name, ".smith")
	return name
}

// ListModules returns the names of all available core modules.
func ListModules() []string {
	entries, err := fs.ReadDir(qasmithStdlib.CoreFS, "core")
	if err != nil {
		return nil
	}
	var modules []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".smith") {
			modules = append(modules, strings.TrimSuffix(entry.Name(), ".smith"))
		}
	}
	return modules
}
