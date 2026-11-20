# Optimization library architecture

Go applications can import `pkg/optimization/qubo`, `pkg/optimization/ising`, and `pkg/optimization/bruteforce` without initializing the compiler. The QUBO package constructs Ising models through the Ising package's public constructor. The exhaustive solver consumes QUBO models. Both model packages reuse internal sparse coefficient, summation, and JSON helpers. The optimization packages use the Go standard library and these mathematical components.

The working example builds a binary objective and an exactly-one penalty, checks a mapped spin assignment, round-trips both models through JSON, and uses exhaustive search.

```sh
go run ./examples/optimization/qubo
```

## Phase 2D and future language integration

A future compile-time value system could introduce immutable, typed library records for a variable count, offset, and ordered sparse terms. It would need schema-aware signature validation, bounded construction and copying, and typed field access or iteration. An adapter could bind those records to the public Go models and expose pure operations under a future `std/optimization` module. Phase 2D provides the [QAOA Go API and JSON-based CLI](qaoa.md). The Smith bindings described here are future work.

The QAOA circuit library consumes the Ising fields and couplings in canonical order together with explicit qubit and parameter mappings. It builds a QAOA cost layer from the existing rotation primitives and reuse the simulator or another backend through typed IR. The mathematical offset would stay in the model for objective evaluation, even when a circuit represents it only up to global phase. An exhaustive solution provides a small classical comparison baseline. The Go package includes the transverse-field mixer, circuit synthesis, and bounded parameter search.
