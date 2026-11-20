# Add a trusted optimization model builder

The [working custom module](../../examples/optimization/language/custom-module/main.go) embeds the public compiler. It creates a one-variable QUBO from a typed input and lets the Smith standard QAOA function build the circuit.

1. Create an `extension.Registry` and register `plan.OptimizationProblem().Descriptor()`. This is the identity the standard QAOA export accepts.
2. Register a module such as `host:custom-optimization` with `RegisterModule`. Export an `extension.Function` returning the opaque type name. Use exact scalar or fixed-array parameter types, or a numeric array rank constraint.
3. Implement the `Evaluate` callback. Inspect dimensions and limits and call `ctx.Spend` before the work. Construct a QUBO with its public builder, then use `plan.FromQUBO` and `kind.Pack`. Optionally supply a typed result decoder.
4. Set `Compiler.Registry` to your registry. Call `Check`, `Prepare`, or `Compile`. A compiler uses a registration snapshot and adds its standard modules. Your registration object remains reusable.

```typescript
import { createProblem } from "host:custom-optimization";
import { qaoa } from "std/optimization/qaoa";

fn main() -> MeasuredResult {
    let problem = createProblem(1.0);
    return qaoa(problem, [0.4], [0.3]);
}
```

Code can define additional opaque types with `extension.NewOpaqueType` or `NewCheckedOpaqueType`. A descriptor's snapshot callback validates and isolates mutable data on pack and unpack. Values from different descriptors with the same name are rejected at the boundary. Registered structured results can apply additional budget checks.

A terminal extension supplies `Produce` and returns a `CircuitPlan` with a `Build` method. It has no access to the compiler's IR storage. The compiler validates measured output, budget limits, and source origins. The ordinary `extension.Intrinsic` interface remains available for borrowed gates. Type-only `CheckTypes` may run during `check`. Value validation, evaluation, and production callbacks run only during lowering.

Host callbacks execute with ordinary Go process privileges. They must be explicitly registered by the embedding application and use their budget context synchronously. Source files can resolve those exports but cannot load native binaries or obtain filesystem, shell, or network APIs through the language.

Use keyed literals for `extension.Value`, `qasmith.Limits`, and `qaoa.OptimizeOptions`. These structs gained fields. Callback signatures for existing intrinsic registrations and the general compiler methods are preserved.
