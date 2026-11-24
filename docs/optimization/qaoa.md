# Generic QAOA

Qasmith v0.10.1 builds QAOA circuits from arbitrary supported Ising models or versioned QUBO models. Prepare a Go problem with `qaoa.New(isingModel)` or `qaoa.FromQUBO(quboModel)`. QUBO conversion happens once and retains the original model for classical comparison.

The CLI reads the existing `qasmith.qubo.v1` JSON format. These commands use the [three-variable example](../../examples/optimization/qaoa/problem.json).

```sh
qasmith qaoa compile examples/optimization/qaoa/problem.json --layers 2 --gammas 0.4,0.2 --betas 0.3,0.6 -o circuit.qasm
qasmith qaoa run examples/optimization/qaoa/problem.json --layers 1 --gammas 0.4 --betas 0.3 --shots 1000 --seed 42
qasmith qaoa optimize examples/optimization/qaoa/problem.json --layers 1 --max-evals 200 --restarts 3 --eval-mode exact --seed 42 -o optimized.qasm
```

QAOA alternates a diagonal cost unitary and a transverse-field mixer. Qasmith minimizes the expected cost, then reports candidates obtained from a separate measurement run. The search is heuristic and has no global optimality or speedup guarantee. [Original QAOA paper](https://arxiv.org/abs/1411.4028)

See the [circuit conventions](qaoa-circuits.md), [evaluation modes](qaoa-evaluation.md), [optimizer settings](qaoa-optimization.md), and [Go extension API](qaoa-extension-api.md).
