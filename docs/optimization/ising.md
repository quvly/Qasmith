# Ising Hamiltonians

Qasmith uses **`z = 1 − 2x`** everywhere in the optimization libraries. Binary zero maps to spin **+1**, and binary one maps to spin **−1**. Variable indices and assignment order are preserved.

```text
H(z) = offset + Σᵢ hᵢ zᵢ + Σᵢ<ⱼ Jᵢⱼ zᵢ zⱼ
zᵢ ∈ {−1, +1}
```

Use `ising.New(variables, offset, fields, couplings)` to construct a model directly. Supply canonical coupling endpoints `I < J`. Field indices and pairs must be unique. Terms may be supplied in any order. The model stores only nonzero coefficients and exposes copies through `Fields` and `Couplings`. Use `Field(i)` and `Coupling(i, j)` for individual coefficients.

Use `ising.Spins(bits)` to convert a Boolean assignment and `Model.Evaluate(spins)` to compute the energy. Evaluation rejects invalid spin values and length mismatches. It uses deterministic compensated Float64 summation and includes the offset.
