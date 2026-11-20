# Trusted optimization modules

| Import | Export | Signature and behavior |
| --- | --- | --- |
| `std/optimization/tsp` | `tsp` | Takes a square rank-two numeric array and an `Int` start. Returns an `OptimizationProblem` with the existing fixed-start QUBO, penalty policy, original directed distances, and route decoder. |
| `std/optimization/qaoa` | `qaoa` | Takes an `OptimizationProblem` and two rank-one numeric arrays of equal nonzero length. Returns `MeasuredResult` by terminal production. Array length sets QAOA depth. |

The two exports above are trusted Go functions registered in a generic module registry. They can be aliased or imported by ordinary local Smith modules and packages. The [travel helper](../../examples/optimization/language/library/travel.smith) is ordinary Smith code using the host-backed TSP export. Standard module paths are resolved from registrations. Project dependencies cannot claim the reserved `std/` namespace.

Numeric arrays are homogeneous fixed-size Smith arrays with Int or Float leaves. The standard adapters convert Int entries to Float64 only when they are exactly representable. They report the relevant matrix or parameter index if conversion would round the value. Float literals already follow Smith's binary64 semantics. Matrix constraints include at least three cities, finite nonnegative entries, and a zero diagonal. The start uses original zero-based indices. See the [TSP guide](tsp.md).

The QAOA module depends on the generic optimization problem type. It also accepts a problem from a custom trusted QUBO factory without a decoder.

The [QUBO module](qubo-language.md) also builds generic objectives directly from Smith.
