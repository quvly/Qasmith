# Compiler architecture

The source loader finds the entry manifest and resolves imports as parsing discovers them. The resulting graph has one AST per canonical source module. Semantic analysis declares signatures, binds imports to exported symbols, validates every function body, and associates call expressions with their targets. A call-graph traversal finds recursion and propagates distinct-qubit constraints up the graph.

Lowering evaluates static scalars and immutable arrays. It binds arguments in per-call frames, evaluates indexing and conditions, and expands each loop in order. Register allocation reserves consecutive logical IDs. Helper parameters carry the caller’s register base and size. Lowering inlines quantum calls using those same IDs. Arithmetic failures point to the operator and report active source call sites. `check` type-checks the full dependency graph and evaluates reachable compile-time constructs with host callbacks disabled. Values that require a callback, and checks that depend on them, are resolved when compiling or evaluating. The IR can be validated, passed through optional optimizations, emitted by a backend, or simulated directly.

## Circuit IR

The public `pkg/ir` package represents a circuit as an ordered list of these value operations:

| Operation | Operands | Semantics |
| --- | --- | --- |
| `AllocateQubit` | `QubitID` | Create a logical qubit initialized to zero. IDs begin at zero and increase in allocation order. |
| `ApplyGate` | `Gate`, ordered qubit IDs, optional numeric parameters | Apply X or H to one live qubit, CX to a distinct control and target in that order, or RX, RY, or RZ with one finite Float64 angle in radians. |
| `Measure` | `QubitID`, `BitID` | Measure a live qubit in the computational basis. Consecutive result IDs start at zero. |

Measurements form a terminal block. A qubit may be measured once. `Program.Validate` checks the sequence and returns register sizes. Optional origin metadata identifies the operation's source and active call sites from innermost to outermost.

Source registers, loops, arrays, and branches are resolved before IR validation. Backends and passes continue to work with the same flat logical qubit IDs and operations.

## OpenQASM 3 and simulation

The default backend declares `qubit[N] q` and `bit[M] c`, includes `stdgates.inc`, and emits gates and measurements in circuit order. It formats angles as decimal literals that round-trip to their original Float64 values. See the official [gate syntax](https://openqasm.com/versions/3.0/language/gates.html) and [measurement syntax](https://openqasm.com/versions/3.0/language/insts.html). The simulator uses the same half-angle convention and preserves complex phase. [OpenQASM standard gates](https://openqasm.com/language/standard_library.html)

The simulator evolves a complex128 statevector once per circuit. Logical qubit ID `i` corresponds to bit `i` of a basis index. Because all measurements are terminal, sampling the final basis distribution and selecting measured qubits yields the same joint distribution as terminal projective measurements. The printed string starts with measurement bit `c[0]`, corresponding to the first argument of `measure`. Sampling is deterministic for a fixed circuit, shot count, and seed. The simulator supports up to 16 qubits.

## Compilation limits

Set `qasmith.Compiler.Limits` in Go or use the corresponding CLI flags. A zero field or flag selects the default. Limits apply across the reachable compilation and produce a source-aware diagnostic when exceeded.

| Limit | Default | CLI flag |
| --- | ---: | --- |
| Total loop iterations | 10,000 | `--max-loop-iterations` |
| Generated gates | 100,000 | `--max-gates` |
| Allocated qubits | 4,096 | `--max-qubits` |
| Compile-time array slots | 32,768 | `--max-array-elements` |
| Static evaluation depth | 256 | `--max-depth` |
| Static evaluation steps | 1,000,000 | `--max-steps` |

Nested expansions share the loop budget. Array slots count across array construction and host boundary copies, including the outer slots of a matrix. Static evaluation steps count evaluated expressions, statements, calls, and blocks. A failed compilation produces no partial circuit. The local simulator keeps its independent 16-qubit limit.


## Terminal circuit producers

Semantic analysis tracks ordinary quantum effects through function calls. A trusted terminal producer returns a typed recipe for a complete circuit. Lowering validates fresh IR and records the source origin. Prepared circuits may be exported or sampled, and supported variational recipes may be optimized separately. See the [execution plan lifecycle](optimization/optimization-plans.md).

Generic Smith model transformations retain immutable QUBO snapshots and charge construction work before allocation. Conversion to Ising occurs when the QAOA plan is created. See [model immutability](optimization/model-immutability.md).
