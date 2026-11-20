# Qasmith Phase 4A compliance report

Qasmith 0.9.0 adds the standalone Go language server and a VS Code extension. The VSIX contains the editor integration and requires a separately installed CLI. The local extension identifier is `qasmith-local.qasmith-vscode`.

| Requirement | Status | Evidence |
| --- | --- | --- |
| Smith file recognition, highlighting, comments, pairing, indentation, folding and snippets | PASS | [Extension manifest](../editors/vscode/package.json), [TextMate grammar test](../editors/vscode/test/grammar.unit.ts), [language configuration](../editors/vscode/language-configuration.json), [snippets](../editors/vscode/snippets/qasmith.json), [extension-host suite](../editors/vscode/test/host.ts). |
| Standalone Go LSP and JSON-RPC lifecycle | PASS | [CLI entry](../internal/cli/cli.go), [framing](../internal/lsp/protocol.go), [protocol tests](../internal/lsp/server_test.go) cover initialization, malformed JSON, request cancellation and graceful shutdown. |
| Unsaved buffers and project awareness | PASS | [Compiler module loader](../internal/modules/graph.go) and [protocol tests](../internal/lsp/server_test.go) cover versioned edits, unsaved dependencies, aliases, missing files, close/save, canonical paths, multi-root sessions, local manifests and import cycles. |
| Live diagnostics and UTF-16 ranges | PASS | [Diagnostic publisher](../internal/lsp/diagnostics.go) and [position conversion](../internal/lsp/positions.go). Protocol tests check register indexing, incomplete programs, stale buffers, Unicode and CRLF. The extension-host suite sees an undefined-gate diagnostic appear after an unsaved edit and clear after the fix. |
| Completion, hover, definition, references, signature help and symbols | PASS | [LSP features](../internal/lsp/features.go) and [protocol tests](../internal/lsp/server_test.go) exercise built-ins, local names, aliased cross-module navigation, public versus private exports, standard-library exports, signatures, references and the outline. |
| Compiler ownership and callback isolation | PASS | [Compiler editor API](../pkg/qasmith/editor.go) reuses the compiler lexer, parser, analyzer and resolver. [Callback test](../pkg/qasmith/editor_test.go) verifies the editor skips registered callbacks and the normal compiler check still applies its registered type checks. |
| Check, compile, run, optimize, preview, restart and output | PASS | [Extension commands](../editors/vscode/src/commands.ts), [process tests](../editors/vscode/test/compiler.unit.ts) and [extension-host suite](../editors/vscode/test/host.ts). The host test runs these commands with an actual local compiler and filenames containing spaces. Preview text matches the emitted file. The standalone OpenQASM 3 parser 1.0.1 also parsed the generated QUBO example as OpenQASM 3.0 with 20 statements. |
| Executable discovery, Workspace Trust and cancellation | PASS | [Launcher](../editors/vscode/src/languageClient.ts) and [process tests](../editors/vscode/test/compiler.unit.ts) cover a missing executable, safe process arguments and cancellation. The [extension-host suite](../editors/vscode/test/host.ts) checks recovery from a missing executable and tests Restricted Mode with a configured executable. |
| Compiler regressions | PASS | `gofmt -w .`, `go test ./...`, `go vet ./...` and `go build ./cmd/qasmith` succeeded. Existing integration tests cover compilation, optimization and simulation. |
| Extension build, lint, tests and VSIX | PASS | `npm ci`, `npm run compile`, `npm run lint`, `npm test`, `npm run test:integration` and official VSIX packaging succeeded. VS Code installed the package into an isolated profile and listed `qasmith-local.qasmith-vscode@0.9.0`. [Local VSIX](../editors/vscode/qasmith-vscode-0.9.0.vsix). |
| User and developer documentation | PASS | [VS Code guide](tooling/vscode.md), [LSP overview](tooling/lsp.md), [development](tooling/development.md), [troubleshooting](tooling/troubleshooting.md), [editor API](tooling/language-server-api.md) and [Phase 4B recommendations](roadmap.md#phase-4b-recommendations). |

## Performance and compatibility

The 40-module benchmark reported 2.78 ms per warm-cache analysis on an Apple M1, with about 1.12 MB allocated per operation. This measures the compiler editor analysis without the network, GUI, or diagnostic debounce. See [benchmark code](../internal/lsp/server_test.go).

The VS Code manifest targets 1.90 and newer. The desktop integration suite was run on 1.147.0 on macOS arm64. The standalone server uses UTF-16 and full-text document updates. References are limited to resolved graphs. Live analysis skips registered callbacks, so some structured-value, model-construction and circuit-production errors surface through CLI commands. Incomplete source recovery stops at the first lexical or syntax error. Distribution is local via VSIX.

VSIX SHA-256 is `0ee2a7c427d2691eea7f52196b90c47a1ac497ae847064ea4f3d7337de857059`.
