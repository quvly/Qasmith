# QUBO to Ising conversion

Use `qubo.Model.ToIsing()` to substitute `xᵢ = (1 − zᵢ)/2`. For QUBO offset `c`, linear coefficients `a`, and canonical quadratic coefficients `b`, the conversion is:

```text
Jᵢⱼ = bᵢⱼ/4
hᵢ = −aᵢ/2 − Σⱼ≠ᵢ b[min(i,j), max(i,j)]/4
offset = c + Σᵢ aᵢ/2 + Σᵢ<ⱼ bᵢⱼ/4
```

For `E = 1 − 2x₀ + 3x₁ + 4x₀x₁`, the result is `H = 2.5 − 2.5z₁ + z₀z₁`. The four assignments `00`, `10`, `01`, and `11` have energies 1, −1, 4, and 6 under both models.

Conversion accumulates contributions as exact rationals before rounding each output coefficient to binary64. This postpones overflow checks until a complete coefficient has been computed. A coefficient outside Float64 range produces an error. Subnormal terms can round to zero, and cancellation among very large rounded coefficients can limit the accuracy of a small energy. The offset remains part of the serialized and evaluated model.

Exhaustive tests compare QUBO and Ising energies to a direct matrix reference using `1e-12 + 1e-12 × max(|actual|, |expected|)` on the small test models. Applications with significantly different coefficient scales should choose a tolerance for their numerical conditioning.
