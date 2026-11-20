# Compiler editor analysis API

Clients launching an external server should use [LSP](lsp.md). A trusted Go host can embed editor analysis through `pkg/qasmith`.

```go
var editor qasmith.EditorSession
snapshot, err := editor.Analyze(ctx, entryPath, map[string]string{
    entryPath: buffer,
})
```

Use absolute paths. `EditorPath` resolves canonical source identities, including new unsaved files with an existing parent directory. Overlay keys are canonicalized during analysis. Files with no matching overlay are read by the compiler resolver. Errors in the source appear in `snapshot.Diagnostics`; entry loading and context cancellation can be returned as Go errors.

`EditorAnalysis.Documents` contains source, compiler tokens, functions, visible binding scopes, resolved occurrences and import exports. `SourcePosition` uses one-based lines and UTF-8 byte columns. `EditorSession` serializes analysis calls and holds a bounded parse cache. Treat returned data as snapshots, and identify declarations by symbol ID within the corresponding source version.

The default server uses the bundled standard-library signatures and performs editor-safe static analysis. Editor analysis skips registered type-checking, value and circuit production callbacks. The normal compiler still applies the registered checks.
