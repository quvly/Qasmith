package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/quvly/Qasmith/pkg/qasmith"
)

type testClient struct {
	t        *testing.T
	in       *io.PipeWriter
	messages chan map[string]json.RawMessage
	done     chan error
	id       int
}

func client(t *testing.T) *testClient {
	input, write := io.Pipe()
	output, send := io.Pipe()
	c := &testClient{t: t, in: write, messages: make(chan map[string]json.RawMessage, 64), done: make(chan error, 1)}
	go func() {
		c.done <- Serve(input, send, io.Discard)
		send.Close()
		input.Close()
	}()
	go func() {
		defer close(c.messages)
		buf := bufio.NewReader(output)
		for {
			raw, err := readFrame(buf)
			if err != nil {
				return
			}
			var reply map[string]json.RawMessage
			if json.Unmarshal(raw, &reply) != nil {
				return
			}
			c.messages <- reply
		}
	}()
	t.Cleanup(func() { write.Close(); output.Close() })
	return c
}

func (c *testClient) send(method string, id int, params any) {
	c.t.Helper()
	m := map[string]any{"jsonrpc": "2.0", "method": method, "params": params}
	if id != 0 {
		m["id"] = id
	}
	if err := writeFrame(c.in, m); err != nil {
		c.t.Fatal(err)
	}
}

func (c *testClient) until(match func(map[string]json.RawMessage) bool) map[string]json.RawMessage {
	c.t.Helper()
	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()
	for {
		select {
		case m, ok := <-c.messages:
			if !ok {
				c.t.Fatal("server closed")
			}
			if match(m) {
				return m
			}
		case <-timeout.C:
			c.t.Fatal("timed out waiting for message")
		}
	}
}

func (c *testClient) request(method string, params any) map[string]json.RawMessage {
	c.id++
	c.send(method, c.id, params)
	return c.until(func(m map[string]json.RawMessage) bool { return string(m["id"]) == fmt.Sprint(c.id) })
}

func need(t *testing.T, payload json.RawMessage, parts ...string) {
	t.Helper()
	for _, part := range parts {
		if !strings.Contains(string(payload), part) {
			t.Fatalf("expected %q in %s", part, payload)
		}
	}
}

func fixture(t testing.TB, root, name, source string) string {
	t.Helper()
	file := filepath.Join(root, name)
	if err := os.WriteFile(file, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	return file
}

func TestProtocolEditor(t *testing.T) {
	root, second := t.TempDir(), t.TempDir()
	root, _ = qasmith.EditorPath(root)
	second, _ = qasmith.EditorPath(second)
	helper := fixture(t, root, "helper file.smith", "pub fn helper(n: Int) -> Int { return n; }\nfn private() -> Int { return 4; }\n")
	entry := filepath.Join(root, "unsaved entry.smith")
	other := filepath.Join(second, "other.smith")
	uri := fileURI(entry)
	source := "import { helper as count } from \"./helper file.smith\";\nimport { qaoa } from \"std/optimization/qaoa\";\nfn main() -> MeasuredResult { let q = qubit(); let n = count(2); h(q); return measure(q); }\n"
	at := func(fragment string, offset int) Position {
		index := strings.LastIndex(source, fragment) + offset
		return Position{strings.Count(source[:index], "\n"), index - strings.LastIndex(source[:index], "\n") - 1}
	}
	c := client(t)
	need(t, c.request("textDocument/hover", nil)["error"], "-32002")
	need(t, c.request("initialize", map[string]any{
		"workspaceFolders":      []workspaceFolder{{fileURI(root)}, {fileURI(second)}},
		"initializationOptions": map[string]any{"diagnostics": map[string]int{"debounceMs": 30}},
		"capabilities": map[string]any{"textDocument": map[string]any{
			"publishDiagnostics": map[string]bool{"versionSupport": true},
			"documentSymbol":     map[string]bool{"hierarchicalDocumentSymbolSupport": true},
		}},
	})["result"], "utf-16", "signatureHelpProvider", "workspaceFolders")
	c.send("initialized", 0, nil)
	fmt.Fprint(c.in, "Content-Length: 1\r\n\r\n{")
	need(t, c.until(func(m map[string]json.RawMessage) bool { return string(m["id"]) == "null" })["error"], "-32700")
	c.send("textDocument/didOpen", 0, map[string]any{"textDocument": textDocument{uri, source, 1}})
	c.send("textDocument/didOpen", 0, map[string]any{"textDocument": textDocument{fileURI(other), "fn main() -> Int { return 1; }", 1}})
	feature := func(method string, position Position) json.RawMessage {
		p := map[string]any{"textDocument": map[string]string{"uri": uri}, "position": position, "context": map[string]bool{"includeDeclaration": true}}
		result := c.request("textDocument/"+method, p)
		if len(result["error"]) > 0 {
			t.Fatalf("%s: %s", method, result["error"])
		}
		return result["result"]
	}
	need(t, feature("completion", at("return", 0)), "\"measure\"", "\"count\"", "\"qaoa\"", "\"q\"")
	exports := feature("completion", at(" as count", 0))
	need(t, exports, "\"helper\"")
	if strings.Contains(string(exports), "private") {
		t.Fatalf("private export: %s", exports)
	}
	need(t, feature("definition", at("count(2)", 1)), fileURI(helper))
	need(t, feature("references", at("count(2)", 1)), fileURI(helper), uri)
	need(t, feature("hover", at("count(2)", 1)), "count(n: Int)", "Int")
	need(t, feature("signatureHelp", at("count(2)", 6)), "n: Int", "activeParameter")
	need(t, feature("documentSymbol", Position{}), "\"main\"", "selectionRange")
	c.send("$/cancelRequest", 0, map[string]int{"id": c.id + 1})
	need(t, c.request("textDocument/hover", map[string]any{"textDocument": map[string]string{"uri": uri}})["error"], "-32800")

	diagnostics := func(target string, version int) json.RawMessage {
		return c.until(func(m map[string]json.RawMessage) bool {
			if string(m["method"]) != "\"textDocument/publishDiagnostics\"" {
				return false
			}
			var p struct {
				URI     string
				Version int
			}
			json.Unmarshal(m["params"], &p)
			return p.URI == target && p.Version == version
		})["params"]
	}
	change := func(target string, version int, text string) {
		c.send("textDocument/didChange", 0, map[string]any{"textDocument": textDocument{URI: target, Version: version}, "contentChanges": []map[string]string{{"text": text}}})
	}
	helperURI := fileURI(helper)
	c.send("textDocument/didOpen", 0, map[string]any{"textDocument": textDocument{helperURI, "pub fn renamed(n: Int) -> Int { return n; }", 1}})
	need(t, diagnostics(uri, 1), "helper")
	change(helperURI, 3, "pub fn helper(n: Int) -> Int { return n; }")
	change(helperURI, 2, "pub fn stale(n: Int) -> Int { return n; }")
	need(t, diagnostics(helperURI, 3), "\"diagnostics\":[]")
	c.send("textDocument/didSave", 0, map[string]any{"textDocument": textDocument{URI: helperURI}})
	c.send("textDocument/didClose", 0, map[string]any{"textDocument": textDocument{URI: helperURI}})
	need(t, diagnostics(helperURI, 0), "\"diagnostics\":[]")
	broken := "fn main() -> Int { /* Ž🧪 */ return unknown; }\r\n"
	change(uri, 2, broken)
	need(t, diagnostics(uri, 2), "unknown", `"character":36`)
	change(uri, 3, "fn helper() -> Int { return 1; }\nfn main() -> Int { let n =")
	need(t, diagnostics(uri, 3), "E1002")
	need(t, feature("documentSymbol", Position{}), "main", "helper")
	c.send("textDocument/didClose", 0, map[string]any{"textDocument": textDocument{URI: uri}})
	need(t, diagnostics(uri, 0), "\"diagnostics\":[]")
	need(t, c.request("textDocument/definition", map[string]any{"textDocument": map[string]string{"uri": uri}})["error"], "no such file")
	if got := c.request("shutdown", nil)["result"]; string(got) != "null" {
		t.Fatalf("shutdown: %s", got)
	}
	c.send("exit", 0, nil)
	select {
	case err := <-c.done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("server did not exit")
	}
}

func TestUTF16RoundTrip(t *testing.T) {
	source := "// Ž🧪\r\n/* Ж🧪 */ fn"
	at := qasmith.SourcePosition{Line: 2, Column: 15}
	if p := toPosition(source, at); p != (Position{1, 11}) || fromPosition(source, "", p) != at {
		t.Fatalf("position=%+v roundtrip=%+v", p, fromPosition(source, "", p))
	}
}

func BenchmarkEditorGraph40Modules(b *testing.B) {
	root := b.TempDir()
	for i := 0; i < 40; i++ {
		source := fmt.Sprintf("pub fn step%d() -> Int { return 1; }", i)
		if i < 39 {
			source = fmt.Sprintf("import { step%d } from \"./m%d.smith\";\npub fn step%d() -> Int { return step%d(); }", i+1, i+1, i, i+1)
		}
		fixture(b, root, fmt.Sprintf("m%d.smith", i), source)
	}
	editor := &qasmith.EditorSession{}
	entry := filepath.Join(root, "m0.smith")
	b.ResetTimer()
	for range b.N {
		a, err := editor.Analyze(context.Background(), entry, nil)
		if err != nil || len(a.Diagnostics) != 0 || len(a.Documents) != 40 {
			b.Fatalf("analysis: %v %v %d", err, a.Diagnostics, len(a.Documents))
		}
	}
}

func TestEditorPackageGraphAndMissingImport(t *testing.T) {
	root, pkg := t.TempDir(), t.TempDir()
	fixture(t, root, "qasmith.json", fmt.Sprintf(`{"name":"app","version":"1.0.0","dependencies":{"shared":{"path":%q}}}`, pkg))
	entry := fixture(t, root, "entry.smith", "import { publicValue as value } from \"shared/math\";\nfn main() -> Int { return value(); }")
	lib := fixture(t, pkg, "math.smith", "import { inner } from \"./inner.smith\";\npub fn publicValue() -> Int { return inner(); }")
	inner := fixture(t, pkg, "inner.smith", "pub fn inner() -> Int { return 42; }")
	canonical, err := qasmith.EditorPath(lib)
	if err != nil {
		t.Fatal(err)
	}
	editor := &qasmith.EditorSession{}
	analyze := func(overlays map[string]string) qasmith.EditorAnalysis {
		t.Helper()
		snapshot, err := editor.Analyze(context.Background(), entry, overlays)
		if err != nil {
			t.Fatal(err)
		}
		return snapshot
	}
	a := analyze(nil)
	if len(a.Diagnostics) != 0 || len(a.Documents) != 3 || a.Documents[canonical] == nil {
		t.Fatalf("manifest graph: %d sources, %v", len(a.Documents), a.Diagnostics)
	}
	broken := analyze(map[string]string{inner: "import { publicValue } from \"./math.smith\";\npub fn inner() -> Int { return publicValue(); }"})
	if !strings.Contains(fmt.Sprint(broken.Diagnostics), "circular module dependency") {
		t.Fatalf("cycle: %v", broken.Diagnostics)
	}
	if err := os.Remove(inner); err != nil {
		t.Fatal(err)
	}
	broken = analyze(nil)
	if !strings.Contains(fmt.Sprint(broken.Diagnostics), "inner.smith") {
		t.Fatalf("removed import: %v", broken.Diagnostics)
	}
}

func TestStandardLibraryImportAndRegisterDiagnostics(t *testing.T) {
	root := t.TempDir()
	entry := fixture(t, root, "entry.smith", "import {  } from \"std/optimization/qubo\";\nfn main() -> MeasuredResult { let q = qreg(4); h(q[8]); return measure(q); }")
	editor := &qasmith.EditorSession{}
	a, err := editor.Analyze(context.Background(), entry, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(fmt.Sprint(a.Diagnostics), "index 8") {
		t.Fatalf("register diagnostic: %v", a.Diagnostics)
	}
	items := completion(a, a.Documents[a.Entry], qasmith.SourcePosition{File: a.Entry, Line: 1, Column: 10})
	data, _ := json.Marshal(items)
	need(t, data, "qubo", "fromMatrix", "addLinear", "addQuadratic", "addExactlyOne", "addSquaredLinear", "scale", "combine")
	reg := a.Documents[a.Entry]
	value, _, ok := symbolAt(reg, qasmith.SourcePosition{File: a.Entry, Line: 2, Column: 35})
	if !ok || value.Type != "QReg<4>" {
		t.Fatalf("register hover: %+v", value)
	}
}
