# Phase 4A compiler API notes

The existing compiler separates the source loader, syntax tree, semantic checking, lowering, and execution. `Compiler.Check` and `Compiler.Dependencies` provide current entry points for source validation and resolved dependencies. `qasmith.Diagnostic` exposes its stable category code and source position, and wraps underlying failures. IR provenance retains the defining location and call sites. Compile-time host data checks and callback execution remain separate from semantic checks.

For Phase 4A, expose a read-only analysis snapshot with per-file module identities, import and alias targets, public and private symbols, function signatures, and inferred expression types. Provide a source-overlay loader for unsaved buffers and document encoding and offset conversions for editor ranges. Preserve related call-site locations separately from formatted diagnostic text. Add error recovery and multiple diagnostics if needed for incomplete buffers. Completion and navigation should consume the real compiler's semantic data and trusted registry exports.

These are proposals for the editor phase. Phase 3C ships the compiler, library, and existing diagnostic APIs.
