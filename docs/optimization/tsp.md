# Fixed-start travelling salesman problems

The public `pkg/optimization/tsp` package accepts complete directed or symmetric distance matrices with at least three cities. All entries must be finite and nonnegative, the diagonal must be zero, and the start must be a valid zero-based city index. Triangle inequality is unnecessary. JSON inputs use `qasmith.tsp.v1` with `start`, `distances`, and an optional `name`.

## Encoding and penalty

Call `tsp.New(distances, start)` or decode a JSON document into `tsp.Problem`. `problem.Encoding()` provides `Index(city, position)` and `Variable(index)`. Variable order is position-major. Within each position, non-start cities appear in ascending original index order. With four cities and start 0, variable indices 0, 1, 2 represent cities 1, 2, 3 at position zero. The next three represent position one.

`tsp.ToQUBO(problem, options)` adds start-to-first and last-to-start distances as linear terms. Directed transitions from one position to the next appear once as pair interactions. Exactly-one penalties apply to each position and each non-start city through the generic QUBO library. The diagonal is zero, including when two positions select the same city in an invalid assignment.

The automatic penalty is the greater of 1 and twice the upper bound `cities × maximum edge distance`. The encoder checks that it strictly exceeds that bound and that the penalty offset is representable. Unsafe automatic values produce an error suggesting rescaling or an explicit override. In the mathematical objective, an invalid assignment incurs at least one unit of weighted squared violation and has nonnegative travel cost, so the automatic choice separates it from every feasible tour. Float64 coefficient accumulation and evaluation still have finite precision. Custom positive finite penalties can make the lowest-energy QUBO assignment infeasible.

## Execution and interpretation

Use `tsp.Prepare` to create and retain the QUBO, encoding, and a generic QAOA problem. `app.QAOA().Build` and `Export` use the standard typed IR and backend. `app.Run` returns decoded sampled outcomes and can also compute the exact expectation. `app.Optimize` delegates parameter search to the existing QAOA engine. Its search objective is expected energy including the constraint penalties.

Display strings start at variable zero. `problem.DecodeBitstring` and `Decode` validate both one-hot constraints. A feasible assignment returns a closed route and original distance. An infeasible assignment returns its violations and no route. `RouteDistance` independently validates and evaluates an explicit tour, using the original directed matrix.

Report metrics separate sampled expected QUBO energy, the best observed QUBO assignment, the best feasible route, and the optional independent classical optimum. A sampled bitstring maps to at most one tour. Counts already aggregate repeat observations of that tour. An unobserved valid route is never reported as a sampled candidate. Optimal sample rate uses all shots and requires a completed reference search.

## Limits

Fixed-start encoding uses `(n-1)²` variables. The default encoding cap is 4,096 variables, with a separate option for callers who need a different budget. The generic circuit builder applies its own gate and qubit limits. The local simulator supports at most 16 qubits, so four and five cities can be simulated, while six cities require 25 qubits and can still be exported within circuit limits.

The independent tour solver enumerates `(n-1)!` permutations. It defaults to at most nine cities, with an explicit ceiling of eleven. It reports the number of tied optimum tours and chooses the lexicographically first route. The generic binary exhaustive solver has its separate variable limits. QAOA search is bounded and heuristic.

See the [four-city example](../../examples/optimization/tsp/README.md) for all six CLI commands and [the Phase 3B proposal](tsp-language.md) for future Smith integration.
