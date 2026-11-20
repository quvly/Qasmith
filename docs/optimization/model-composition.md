# Model composition

`scale(model, factor)` applies the factor to the constant offset and every term. Zero clears the coefficients and offset while preserving the variable count. Negative factors are valid.

`combine(left, right)` adds coefficients and offsets by variable index. The two models must have the same variable count and share the caller's indexing convention. No variable remapping is performed.

The [composition example](../../examples/optimization/qubo-language/composition.smith) combines a two-variable objective with a scaled matrix model. Separate variables hold the unscaled models for reuse.

The generic functions reject any input with an attached domain decoder. For example, a TSP route report describes a particular matrix and constraint encoding, so modifying its QUBO through a generic operation could make the report misleading. Domain-specific transformation policies belong to the library that owns the decoder.
