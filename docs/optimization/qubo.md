# Quadratic binary objectives

A QUBO is a function on binary variables with zero-based indices.

```text
E(x) = offset + Σᵢ aᵢ xᵢ + Σᵢ<ⱼ bᵢⱼ xᵢ xⱼ
xᵢ ∈ {0, 1}
```

Use `qubo.New(variables)` to create a mutable builder. `AddLinear`, `AddQuadratic`, `SetOffset`, and `AddOffset` return errors for invalid inputs or coefficient overflow. `AddQuadratic(i, j, c)` accepts either order. Equal indices become a linear term because `xᵢ² = xᵢ`. Repeated contributions add. Zero-valued terms occupy no sparse storage.

```go
model, err := qubo.FromMatrix([][]float64{
    {-2, 4},
    {0, 3},
}, 1.0)
if err != nil {
    return err
}
energy, err := model.Evaluate([]bool{true, false})
```

The model in this example evaluates to −1. For a matrix, the convention is `E(x) = xᵀQx + offset`. The diagonal sets the linear coefficients. An unordered pair contributes `Q[i][j] + Q[j][i]` exactly once. Either triangle can carry the whole coefficient. A symmetric matrix with equal entries in both triangles contributes twice the individual entry.

`Build` snapshots the builder. The finalized `Model` supports value copies and concurrent reads. Term slices returned by `LinearTerms` and `QuadraticTerms` are independent copies. `Linear` and `Quadratic` return zero for absent terms. Use a positive variable count, including for constant-only models.

`Evaluate` requires exactly as many Boolean values as the model has variables. Evaluation uses compensated Float64 summation in the order offset, increasing linear index, then lexicographic pair. It reports non-finite intermediate or final energies. For example, overflow can occur in a partial sum even if later terms would mathematically cancel it.
