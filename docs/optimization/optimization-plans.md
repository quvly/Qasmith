# Optimization execution plans

`Compiler.Prepare` returns validated IR, the producer's immutable circuit recipe when present, and its source origin. The QAOA recipe in `pkg/optimization/plan` retains the QUBO-derived problem, parameter lists, layer count, variable order, decoder, and build budgets. Calling `Parameters()` returns copies. A prepared recipe can build a fresh circuit without simulation or search.

- `compile` evaluates the source model, builds the recipe's measured circuit at the source parameters, applies configured passes, and emits OpenQASM with the configured backend.
- `run` samples the prepared circuit, evaluates original QUBO energies, and invokes the optional decoder.
- `optimize` extracts the recipe and calls the existing generic QAOA optimizer with the source parameters and layer count. It builds a revised circuit and may sample that circuit with a separate deterministic seed. Use `--no-sample` to skip final measurement execution and `--output-qasm` to save the optimized circuit. The Smith file is unchanged.

Exact and sampled search objectives are selected with `--eval-mode exact` or `--eval-mode sampled`. Both minimize the energy expectation. Results distinguish search expectation, final sample estimate, and best observed assignment. A compiled quantum program without a supported recipe can run but cannot use the general optimization command.

## Resource limits

Existing array, static step, gate, and qubit limits apply throughout. These additional defaults can be overridden in `qasmith.Compiler.Limits` or with the corresponding CLI flags.

| Resource | Default | CLI |
| --- | ---: | --- |
| Matrix elements | 4,096 | `--max-matrix-elements` |
| Model variables | 4,096 | `--max-model-variables` |
| Coefficients including offset | 100,000 | `--max-model-coefficients` |
| Charged structured evaluation work | 250,000 | `--max-structured-work` |
| Circuit layers | 64 | `--max-circuit-layers` |

Trusted callbacks charge their work against both structured and static step budgets. The supplied TSP integration checks model sizes before building the QUBO. The compiler revalidates returned IR and prevents passes from changing a terminal recipe's register size or measurement order. The [optimizer's limits](qaoa-optimization.md) and the 16-qubit local simulator limit are independent of circuit export.
