# Optimization model JSON

Use Go's `encoding/json` with `qubo.Model` or `ising.Model`. Both schemas include a version identifier, positive variable count, constant offset, and dense index-ordered single-variable coefficients. Pair coefficients use sparse canonical entries sorted by `(i, j)`.

```json
{
  "schema": "qasmith.qubo.v1",
  "variables": 2,
  "offset": 1,
  "linear": [-2, 3],
  "quadratic": [{"i": 0, "j": 1, "coefficient": 4}]
}
```

```json
{
  "schema": "qasmith.ising.v1",
  "variables": 2,
  "offset": 2.5,
  "fields": [0, -2.5],
  "couplings": [{"i": 0, "j": 1, "coefficient": 1}]
}
```

The encoder preserves sufficient digits to reconstruct finite Float64 coefficients and always orders canonical pairs consistently. The decoder validates the full document before replacing the receiver. It rejects missing, unknown, duplicate, and null members, invalid pair indices or order, repeated pairs, non-finite numbers, and a dense vector whose length differs from the declared variable count. An out-of-order list of individually canonical pairs is accepted and normalized.

A zero coefficient and an absent term represent the same mathematical value. JSON size is linear in the variable count plus the number of interactions due to the dense coefficient vector, even though the in-memory representation is sparse.
