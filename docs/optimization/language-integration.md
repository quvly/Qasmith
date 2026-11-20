# Optimization in Smith

A Smith program can construct an optimization problem with a trusted library and return a measured circuit from a terminal producer. The root [tsp.smith](../../tsp.smith) uses the four-city TSP example.

```typescript
import { tsp } from "std/optimization/tsp";
import { qaoa } from "std/optimization/qaoa";

fn main() -> MeasuredResult {
    let distances = [[0, 2, 3], [2, 0, 1], [3, 1, 0]];
    let problem = tsp(distances, 0);
    return qaoa(problem, [0.4], [0.3]);
}
```

`OptimizationProblem` is a registered opaque compile-time type. It holds an immutable QUBO-backed model, variable ordering, and optional decoder. Smith can bind it, pass it to a typed parameter, and return it from a pure helper in another module. Its fields are accessible to trusted Go code through the type descriptor's typed API. Smith has no implicit scalar or qubit conversion, indexing, mutation, or array literals of opaque values.

A terminal producer is permitted only as the top-level final returned expression of `main() -> MeasuredResult`. Classical declarations and pure helpers may precede it. Allocations, borrowed gates, measurements, and calls with other quantum effects cannot be combined with it, including through helpers.

```sh
go build ./cmd/qasmith
./qasmith check tsp.smith
./qasmith compile tsp.smith -o circuit.qasm
./qasmith run tsp.smith --shots 1000 --seed 42
./qasmith optimize tsp.smith --max-evals 200 --restarts 3 --shots 1000 --seed 42 --output-qasm optimized.qasm
```

`check` validates imports, signatures, static shapes, and effects. It defers host value callbacks and circuit production. Thus a negative distance or invalid start can pass shape checking and then produce a source-aware error during compilation. See the [examples](../../examples/optimization/language/README.md).
