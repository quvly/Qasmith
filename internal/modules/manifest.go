package modules

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/quvly/Qasmith/internal/diagnostic"
)

type dependency struct {
	Path string `json:"path"`
}
type manifest struct {
	Name         string                `json:"name"`
	Version      string                `json:"version"`
	Dependencies map[string]dependency `json:"dependencies"`
}

var packageName = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*$`)
var version = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[A-Za-z0-9-]+(\.[A-Za-z0-9-]+)*)?(\+[A-Za-z0-9-]+(\.[A-Za-z0-9-]+)*)?$`)

func project(start string) (string, map[string]string, error) {
	for dir := start; ; dir = filepath.Dir(dir) {
		path := filepath.Join(dir, "qasmith.json")
		info, err := os.Lstat(path)
		if os.IsNotExist(err) {
			if filepath.Dir(dir) == dir {
				return start, nil, nil
			}
			continue
		}
		if err != nil {
			return "", nil, diagnostic.At(filePos(path), "E2100", "%v", err)
		}
		if !info.Mode().IsRegular() {
			return "", nil, diagnostic.At(filePos(path), "E2100", "manifest must be a regular file")
		}
		deps, err := readManifest(dir, path)
		return dir, deps, err
	}
}

func readManifest(dir, path string) (map[string]string, error) {
	fail := func(format string, args ...any) (map[string]string, error) {
		return nil, diagnostic.At(filePos(path), "E2100", format, args...)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fail("%v", err)
	}
	if err := uniqueKeys(json.NewDecoder(bytes.NewReader(data))); err != nil {
		return fail("%v", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var m manifest
	if err := decoder.Decode(&m); err != nil {
		return fail("%v", err)
	}
	if _, err := decoder.Token(); err != io.EOF {
		return fail("unexpected data after manifest")
	}
	if !packageName.MatchString(m.Name) || !version.MatchString(m.Version) {
		return fail("invalid project name or version")
	}
	var names []string
	for name := range m.Dependencies {
		names = append(names, name)
	}
	slices.Sort(names)
	roots := make(map[string]string)
	for _, name := range names {
		dep := m.Dependencies[name]
		if !packageName.MatchString(name) || name == "std" || strings.EqualFold(name, m.Name) || strings.TrimSpace(dep.Path) == "" {
			return fail("invalid or reserved dependency %q", name)
		}
		candidate := dep.Path
		if !filepath.IsAbs(candidate) {
			candidate = filepath.Join(dir, candidate)
		}
		root, err := canonical(candidate)
		if err != nil {
			return fail("dependency %q: %v", name, err)
		}
		info, err := os.Stat(root)
		if err != nil || !info.IsDir() {
			return fail("dependency %q must refer to a directory", name)
		}
		if within(root, dir) {
			return fail("dependency %q overlaps the project root", name)
		}
		for _, other := range names {
			otherRoot, found := roots[other]
			if found && (strings.EqualFold(name, other) || within(root, otherRoot) || within(otherRoot, root)) {
				return fail("conflicting dependencies %q and %q", other, name)
			}
		}
		roots[name] = root
	}
	return roots, nil
}

func uniqueKeys(d *json.Decoder) error {
	return readJSONValue(d, 0)
}

func readJSONValue(d *json.Decoder, depth int) error {
	if depth > 32 {
		return fmt.Errorf("manifest nesting exceeds 32")
	}
	t, err := d.Token()
	if err != nil {
		return err
	}
	if t != json.Delim('{') && t != json.Delim('[') {
		return nil
	}
	keys := make(map[string]bool)
	for d.More() {
		if t == json.Delim('{') {
			key, err := d.Token()
			if err != nil {
				return err
			}
			name := strings.ToLower(key.(string))
			if keys[name] {
				return fmt.Errorf("duplicate manifest key %q", key)
			}
			keys[name] = true
		}
		if err := readJSONValue(d, depth+1); err != nil {
			return err
		}
	}
	_, err = d.Token()
	return err
}
