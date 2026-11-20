# Standard library

The compiler distributes the trusted host-backed modules `std/optimization/tsp`, `std/optimization/qaoa`, and `std/optimization/qubo`. The registry resolves their explicit exports through normal Smith imports. See the [standard optimization modules](../docs/optimization/standard-library.md).

The compiler also provides the primitive functions `qubit`, `qreg`, `len`, `x`, `h`, `cx`, `rx`, `ry`, `rz`, and `measure`, along with the constant `pi`. The [package guide](../docs/packages.md) shows how to wrap primitives in reusable Smith functions.
