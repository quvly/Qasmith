# Quantum Optimization in Qasmith

Qasmith includes built-in support for quantum optimization through QUBO models, Ising formulations, and the QAOA algorithm. This section guides you through building and solving optimization problems with quantum circuits.

## Overview

Quantum optimization maps a problem onto an energy function and finds the lowest-energy state using a quantum circuit. Qasmith supports the full workflow:

1. **Model** — express your problem as a QUBO or Ising model
2. **Compile** — generate a QAOA circuit from the model
3. **Optimize** — tune circuit parameters to minimize energy
4. **Decode** — interpret the measurement result as a solution

## Optimization approaches

### QUBO (Quadratic Unconstrained Binary Optimization)

Express a problem as a quadratic function of binary variables:

$$\min_{x \in \{0,1\}^n} x^T Q x$$

- [QUBO Language Guide](qubo-language.md) — use QUBO in Smith code
- [QUBO Go API](qubo.md) — use QUBO from Go
- [Custom Models](custom-models.md) — build your own QUBO models

### Ising Model

Map a problem to spin variables:

$$\min_{s \in \{-1,+1\}^n} \sum_i h_i s_i + \sum_{i<j} J_{ij} s_i s_j$$

- [Ising Model Guide](ising.md) — formulation and conversion

### QAOA (Quantum Approximate Optimization Algorithm)

Run a variational quantum circuit to approximate the optimal solution:

- [QAOA Guide](qaoa.md) — full QAOA workflow
- [QAOA Circuits](qaoa-circuits.md) — circuit structure
- [QAOA Evaluation](qaoa-evaluation.md) — measure and interpret results

### Travelling Salesman Problem (TSP)

A complete application solving the TSP with quantum optimization:

- [TSP Guide](tsp.md) — API, limits, and route interpretation

## Standalone libraries

The Go packages under `pkg/optimization` build and evaluate QUBO and Ising models, convert between energy conventions, add penalties, serialize models, and solve small QUBOs by bounded exhaustive search.

```bash
go run ./examples/optimization/qubo
```

See the [QUBO guide](qubo.md) and [optimization architecture](architecture.md).

## Smith-language integration

Build optimization models directly in Smith code:

```typescript
import "std/qubo" as qubo;

fn main() -> MeasuredResult {
    let model = qubo::new(2);
    qubo::linear(model, 0, -1.0);
    qubo::linear(model, 1, -1.0);
    qubo::quadratic(model, 0, 1, 2.0);
    return qubo::solve(model);
}
```

## Next steps

- Start with [Building Your First Quantum Optimization Model](custom-models.md)
- Browse the [QUBO examples](../../examples/optimization/qubo-language/README.md)
- Run the [QAOA example](../../examples/optimization/qaoa/README.md)
