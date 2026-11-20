# Build QUBO objectives in Smith

These examples construct models with the standard QUBO functions and use the existing QAOA producer.

- [matrix.smith](matrix.smith) and [builder.smith](builder.smith) create the same unconstrained objective.
- [constraints.smith](constraints.smith) applies exactly-one and weighted squared penalties.
- [composition.smith](composition.smith) combines a model with a scaled objective.
- [imported.smith](imported.smith) passes a model through reusable Smith functions.
- [packaged.smith](packaged.smith) imports a parameterized model from a manifest-declared local dependency.

From the repository root:

```sh
go build ./cmd/qasmith
./qasmith check program.smith
./qasmith compile examples/optimization/qubo-language/matrix.smith -o matrix.qasm
./qasmith compile examples/optimization/qubo-language/builder.smith -o builder.qasm
cmp matrix.qasm builder.qasm
./qasmith run program.smith --shots 1000 --seed 42
./qasmith optimize program.smith --max-evals 200 --restarts 3 --eval-mode exact --shots 1000 --seed 42 --output-qasm optimized.qasm
```

The root program adds an exactly-one penalty to the reference matrix. The [run](expected/run-seed-42.txt) and [optimization](expected/optimize-seed-42.txt) captures show this configuration at seed 42. Counts are observed samples. The [tutorial](../../../docs/optimization/custom-models.md) covers writing your own model.
