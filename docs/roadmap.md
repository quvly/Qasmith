# Limitations and proposed milestones

Phase 2B provides fixed-size quantum registers and classical arrays, compile-time loops and conditions, static rotation angles, and terminal measurement. Quantum helpers borrow existing resources. The ideal local simulator supports up to 16 qubits. Compilation can produce larger circuits within its independent [resource limits](architecture.md#compilation-limits).

Arrays are immutable and their dimensions are fixed. Registers have concrete sizes in signatures. Source programs cannot use runtime parameter binding, measurement feedback, runtime control flow, mutation, recursion, closures, or generic types. Native runtime plugins, remote dependencies, and hardware-specific transpilation remain outside the current compiler.

Phase 2C adds standalone QUBO and Ising mathematics with a bounded classical reference solver. Phase 2D adds generic QAOA generation, expectation evaluation, and bounded hybrid parameter search through Go and the QUBO JSON CLI. Phase 3A adds the fixed-start TSP application through Go and CLI. Phase 3B adds trusted Smith optimization modules, opaque compile-time values, and execution plans. [Smith optimization guide](optimization/language-integration.md)

- Extend the standard library with compiler-distributed Smith code using the existing trusted module boundary.
- Establish ancilla ownership and uncomputation rules before introducing reversible integer arithmetic.
- Explore symbolic circuit parameters and additional targets while preserving typed IR.
- Add package metadata and dependency version selection before introducing a remote package registry.

Phase 3C adds user-defined QUBOs, generic penalties, and immutable composition through reusable Smith packages. Phase 4A adds the VS Code extension and standalone LSP. See the [editor tooling guide](tooling/vscode.md).

## Phase 4B recommendations

- Improve recovery across multiple independent parse and semantic errors in a single file.
- Add incremental synchronization and dependency-aware indexing before extending references beyond loaded module graphs.
- Add document links for import paths and evaluate safe quick fixes for compiler diagnostics.
- Establish platform-specific CLI distribution and VSIX release verification before publishing the extension.
