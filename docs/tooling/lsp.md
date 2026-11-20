# Qasmith language server

Run `qasmith lsp --stdio` from any editor with an LSP client. The server uses Content-Length framing and reserves stdout for protocol frames. Protocol transport errors go to stderr.

The server maintains versioned in-memory Smith buffers, including imported files. It passes their contents to the compiler editor API. The module resolver, lexer, parser, semantic analyzer and standard-library registry supply the language information. A bounded parse cache is keyed by path and source text. Diagnostics use a debounced analysis worker, and edits invalidate earlier results. Editor analysis skips circuit production and value callbacks. Simulation and parameter search are launched by explicit CLI commands.

| Capability | Behavior |
| --- | --- |
| Synchronization | Full-text open/change/save/close. Workspace folders and watched-file invalidation. |
| Diagnostics | Compiler diagnostic codes and UTF-16 ranges. Source version accompanies open-document results when the client advertises support. |
| Completion | Built-ins, visible lexical bindings, types, imported names, module exports and common keywords. |
| Hover | Resolved symbol signatures and registry documentation. |
| Definition and references | Source declarations and uses across the current graph and graphs of open documents. Registry-provided functions may lack source locations. |
| Signature help | Innermost call, with the current argument tracked as the cursor moves. |
| Document symbols | Functions and their source ranges from the compiler AST. |

An analysis can load up to 128 modules, each up to 1 MiB. The server accepts up to 64 open Smith documents, and JSON-RPC messages are limited to 8 MiB. The compiler's normal static limits still apply. Workspace folders do not add import search paths. Each entry uses the compiler's normal local manifest and path rules. References do not discover unopened, unrelated entry points. Recovery provides analysis for the source prefix up to the first parser or lexer error. A function can report its first semantic error before further errors become visible. The server currently uses UTF-16 full-text synchronization. [LSP 3.17 specification](https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/)
