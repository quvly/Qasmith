# Three-variable QAOA example

Run from the repository root with the embedded generic QUBO and seed 42.

```sh
go run ./examples/optimization/qaoa -initial-output initial.qasm -o optimized.qasm
```

The example converts the QUBO, generates a depth-one circuit, and compares exact and sampled expected energies. It searches for lower expected energy, runs the best parameters with a separate measurement seed, and decodes the best observed bitstring. It also computes the bounded classical reference and the fraction of final samples at that energy. Both OpenQASM files use the existing Qasmith backend.

See the [QAOA guide](../../../docs/optimization/qaoa.md) for the corresponding CLI commands.
