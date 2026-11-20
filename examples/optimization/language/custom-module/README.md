# Custom trusted optimization module

From the repository root, run:

```sh
go run ./examples/optimization/language/custom-module
```

The Go host registers a typed one-variable QUBO factory. [factory.smith](factory.smith) imports and wraps that export, and [program.smith](program.smith) passes the opaque problem to the standard QAOA producer. The embedding command prints its generated OpenQASM and measured counts. The standalone Qasmith CLI includes the standard modules by default. The custom factory is available through this embedding command's explicit registration.

The [host extension guide](../../../../docs/optimization/host-optimization-extensions.md) explains registration, identity checks, and structured work budgets.
