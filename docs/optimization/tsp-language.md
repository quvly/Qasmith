# Proposed Phase 3B Smith integration

Phase 3A exposes TSP through Go and a JSON-based CLI. Smith already supports immutable numeric matrices for distances, integers for the start and layer count, Float arrays for gamma and beta, and fixed-size borrowed registers. These suffice for input representation and parameterized gate helpers.

A reusable library would also need immutable structured types for the validated distance problem, sparse QUBO terms, Ising fields and couplings, and route-encoding metadata. A typed extension API could accept a static matrix and return a compiler-managed value with a declared schema and bounded dimensions. A subsequent extension could consume that value and the two parameter arrays. It would validate both array lengths against the positive layer count and derive the required circuit width from the encoding.

The existing host gate API can apply typed gates to borrowed qubits. A future resource extension could either require the Smith caller to allocate a checked register or expose a compiler-mediated allocator for entry points. An allocator would need to account for qubit and gate budgets, issue stable logical IDs, preserve ownership, and forbid allocation after measurement. Callbacks would emit validated typed operations and retain source origins. No Go callback would obtain raw mutable access to the compiler's circuit or resource state.

Runtime optimization would remain in the execution layer. It would repeatedly build and evaluate circuits with explicit parameter values, budgets, and cancellation, then return counts and route records. Compile-time checking would validate shapes and resource use without executing the search.

A future `std/optimization` package could expose ordinary reusable functions such as `tsp` and `qaoa` through this API. The public Go models, encoder, and decoder would supply the mathematical implementation. Smith support for those imports and structured values is proposed, and Phase 3A adds no optimization-specific syntax or IR.
