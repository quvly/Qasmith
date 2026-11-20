# Hybrid parameter optimization

`qaoa.Optimize(ctx, problem, options)` minimizes the evaluator's expected energy. The default search polls both signs of each gamma and beta coordinate, starting with a `π/4` step. It halves the step after a full sweep without improvement. Additional starts are drawn from a seeded generator, with gammas in `[-π, π)` and betas in `[-π/2, π/2)`. Starts share the evaluation and iteration budgets and are swept in round-robin order. These intervals initialize restarts and are not hard parameter bounds or assumed periods.

| Setting | Default | CLI |
| --- | ---: | --- |
| Layers | 1 | `--layers` |
| Initial gamma and beta per layer | 0.4 and 0.3 | `--gammas`, `--betas` |
| Objective evaluations | 200 | `--max-evals` |
| Coordinate sweeps | 100 | `--max-iters` |
| Step tolerance | 0.0001 | `--tolerance` |
| Seed | 0 | `--seed` |
| Additional starts | 0 | `--restarts` |
| Objective mode | exact | `--eval-mode exact\|sampled` |
| Shots per sampled evaluation and final run | 1,000 | `--shots` |

Search evaluation and sweep budgets are each capped at 100,000, and additional starts at 64. Convergence means every requested start reached the step tolerance. Reaching a budget returns the best evaluated parameters and reports the stopping reason. The returned counts come from a separate deterministic final seed. During sampled search, using a fixed seed at each evaluation keeps comparisons reproducible.

For Go callers, set `Reference: true` to enable bounded exhaustive comparison. The CLI enables it by default and accepts `--reference=false`. The default reference limit is 20 variables and can be lowered or raised up to 24. QUBO inputs use the original model for reference energies. Direct Ising problems use the same exhaustive solver with a spin-mapped binary objective. The optimal sample fraction counts all final assignments whose reference energy equals the reference minimum, including ties. The result explicitly indicates when no reference was calculated.
