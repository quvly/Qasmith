# Generic constraint penalties in Smith

`addExactlyOne(model, [0, 1, 2], 10.0)` adds `10 × (x₀ + x₁ + x₂ − 1)²`. The input list is nonempty, uses valid zero-based indices, and contains no repeats. A zero penalty is allowed and still validates the index list.

`addSquaredLinear(model, [0, 1, 2], [1.5, -2.0, 0.5], 1.5, 2.0)` adds `2 × (1.5x₀ − 2x₁ + 0.5x₂ − 1.5)²`. The index and weight arrays must have the same fixed length. Targets and weights may be negative or fractional. Penalties must be finite and nonnegative.

Both calls use the existing [QUBO penalty expansion](penalties.md), including its constant term. The [constraints example](../../examples/optimization/qubo-language/constraints.smith) applies both forms through imported helpers. Small penalties can trade constraint violations against other terms in the objective. QAOA sampling can produce assignments that violate a constraint even when the classical minimum satisfies it.
