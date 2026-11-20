# QAOA energy evaluation

The objective is the Ising expectation `Σ_x P(x) H_C(1−2x)`. It includes the model offset. `qaoa.EvaluateExact` uses the simulator's unmeasured statevector and squared amplitude magnitudes. It normalizes small floating-point drift and uses compensated summation. Exact mode is deterministic and uses no measurement randomness.

`qaoa.EvaluateSampled` builds a circuit with terminal measurements. It computes an estimate by weighting each observed energy by its measured frequency. It returns counts, the estimated expectation, and the lowest-energy observed assignment with its count and frequency. Equal observed energies select the first bitstring lexicographically. Use `qaoa.DecodeBitstring` to obtain the Boolean assignment.

`qaoa.NewLocalEvaluator` precomputes one table of basis energies for repeated evaluations. The local evaluator rejects models above 16 qubits before allocating the table. Sampling accepts 1 to 1,000,000 shots per evaluation and is reproducible for a fixed seed. The simulator observes cancellation between gates and in batches of shots. Exact evaluation and table construction check cancellation during traversal.

A sampled expectation carries sampling error. A best observed assignment is one particular outcome, and a classical minimum is available only when the bounded exhaustive solver completes. Float64 rounding applies to all reported energies.
