# QAOA circuit conventions

The circuit starts in the uniform state by applying H to every allocated qubit. For depth `p`, supply exactly `p` finite gammas and `p` finite betas. Layer zero executes first. Each layer applies its cost unitary and then its mixer.

The Ising Hamiltonian is `H_C = C + Σ hᵢ Zᵢ + Σ Jᵢⱼ Zᵢ Zⱼ`. To implement `exp(−i γ H_C)`, each nonzero field emits `RZ(2γhᵢ)`. Each nonzero coupling in canonical order emits CX, `RZ(2γJᵢⱼ)` on the target, and CX again. The constant offset contributes only global phase. The builder omits that phase from gates and the evaluators include the offset in every energy.

The mixer is `exp(−i β Σ Xᵢ)`, implemented with `RX(2β)` on every qubit in increasing order. Parameters are numeric radians at circuit generation time.

`problem.Build(params, qaoa.BuildOptions{})` returns unmeasured, validated IR. Set `Measure: true` for terminal measurements. `problem.Export(params, options, nil)` selects the existing OpenQASM 3 backend. The CLI emits measured circuits and defaults to a sibling `.qasm` path when `-o` is omitted.

Variable and qubit IDs match. Basis index bit `i` is qubit `i`. Displayed measurement strings start with variable zero, so `100` means `x[0]=1, x[1]=0, x[2]=0`, or basis index 1. The corresponding spins are `[-1,+1,+1]`.

Circuit defaults match Smith's 4,096-qubit and 100,000-gate budgets. Go build options and the CLI's `--max-qubits` and `--max-gates` control these budgets. The local simulator independently supports up to 16 qubits, so larger circuits can be exported within their build limits.
