# Constraint penalties

Use `Builder.AddSquaredPenalty(weights, target, lambda)` to add the cost `lambda × (Σᵢ wᵢ xᵢ − target)²`. Supply indexed `qubo.Weight` entries with finite values and distinct indices. The target must be finite and the strength must be finite and nonnegative.

```text
offset increment = lambda × target²
linear increment at i = lambda × (wᵢ² − 2 × target × wᵢ)
quadratic increment at i,j = 2 × lambda × wᵢ × wⱼ
```

`AddExactlyOne([]int{0,1,2}, strength)` uses the same expansion with unit weights and target one. The penalty is zero when exactly one variable is selected and positive for other assignments when strength is positive. To make violations unattractive in a combined model, choose a strength with respect to its other objective terms.

Use `Model.Scale(factor)` to obtain a new scaled model and `Builder.AddModel(model)` to combine models of the same variable count. Scaling and composition include their constant offsets. Failed penalty additions and failed model composition leave the builder unchanged.

## Classical reference solver

`bruteforce.Solve(model)` exhaustively checks assignments up to 20 variables by default. `Solver{MaxVariables: n}.Solve(ctx, model)` lets the caller lower or raise the limit up to 24 and cancel a running search. Equal Float64 energies retain the lexicographically first assignment with zero before one. Search time is exponential in the number of variables.
