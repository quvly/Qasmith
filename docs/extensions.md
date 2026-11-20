# Extending the Qasmith Compiler in Go

Use `.smith` packages to distribute functions written in Qasmith. Use the Go API when an embedding application needs to register a trusted intrinsic, run an IR pass, or select a backend. Go extensions are compiled into the embedding program and registered by that program. The standalone CLI exposes built-in operations.

## Register a host intrinsic

The following Go function can be part of a custom host. It exposes `flipTwice` with a single borrowed qubit argument.

```go
import (
    "github.com/quvly/Qasmith/pkg/extension"
    "github.com/quvly/Qasmith/pkg/ir"
    "github.com/quvly/Qasmith/pkg/qasmith"
)

func compilerWithGates() (qasmith.Compiler, error) {
    registry := &extension.Registry{}
    err := registry.Register(extension.Intrinsic{
        Namespace: "example",
        Name:      "flipTwice",
        Signature: extension.Signature{
            Parameters: []extension.Type{extension.Qubit},
            Return:     extension.Void,
        },
        Lower: func(ctx extension.LoweringContext, args []extension.Value) (extension.Value, error) {
            for range 2 {
                if err := ctx.ApplyGate(ir.X, args[0].Qubit); err != nil {
                    return extension.Value{}, err
                }
            }
            return extension.Value{Type: extension.Void}, nil
        },
    })
    return qasmith.Compiler{Registry: registry}, err
}
```

Source programs explicitly import the registered name:

```typescript
import { flipTwice } from "host:example";

fn main() -> MeasuredResult {
    let q = qubit();
    flipTwice(q);
    return measure(q);
}
```

A signature can use classical and `Qubit` parameters and return a classical type or `Void`. A classical intrinsic must be pure and has no qubit parameters. Declare distinct qubit operand pairs in `Signature.Distinct` for static alias checking through helper calls. An optional `Validate` callback checks concrete argument values during lowering. `Check` validates signatures and distinct operands without running either callback.

Call `ApplyGate` synchronously during lowering. It accepts supported typed gates and borrowed qubits from that call. The host API has no allocation, measurement, or raw QASM emission operation. Callback code has ordinary Go privileges and is part of the trusted executable. Source modules and manifest files cannot load native extensions.

## Replace the backend or add passes

Implement `ir.Backend` with `Name` and `Compile`, then assign it to `qasmith.Compiler.Backend`. A nil backend selects OpenQASM.

Implement `ir.Pass` with `Name` and `Run`, then add it to `qasmith.Compiler.Passes`. Passes run in slice order and their output is validated. For example, set `Passes: []ir.Pass{qasmith.CancelAdjacentX{}}` to cancel consecutive X gates on the same qubit. Omit passes to retain the original instruction stream. Use `Compiler.Lower` to inspect the typed circuit directly.

## Parameterized gates

For rotations, assert `ctx` to `extension.ParameterizedLoweringContext` and call `ApplyParameterizedGate(ir.RZ, []float64{angle}, q)`. It checks gate arity, finite angles, and borrowed qubits. Existing callbacks still receive `extension.LoweringContext` and can keep calling `ApplyGate` for fixed gates.

Use keyed Go literals for `extension.Value` and `ir.ApplyGate`. These structs gained the `Float` and `Parameters` fields respectively. Fixed gates omit `Parameters`.

For example, migrate an old positional value literal to `extension.Value{Type: extension.Int, Integer: 42}` and an old positional gate literal to `ir.ApplyGate{Gate: ir.X, Qubits: []ir.QubitID{q}, Origin: origin}`. The original `Register` signature, callback signature, and fixed-gate interface still work with these literals.

## Boolean and array values

Use `extension.Bool` and the `Value.Bool` field for Boolean signatures and values. Build fixed-size classical array types with `extension.ArrayType`. For example, `extension.ArrayType(extension.ArrayType(extension.Int, 2), 3)` represents Smith's `Int[2][3]`. Use `Value.Array` with an `extension.ArrayValue` containing the typed elements. The compiler validates returned shapes, element types, and finite floats. It copies arrays at the callback boundary to protect source-level immutability.

Host gate callbacks can still borrow individual qubits, including `q[i]` from Smith. Registers and arrays of qubits are unavailable in host signatures. For existing host names that became Smith keywords, import the original registration name under an alias, such as `import { if as test } from "host:example";`.

Phase 2B adds fields to `extension.Value` and `qasmith.Compiler`. Use keyed Go literals for both structs. The host registration and lowering callback signatures are unchanged.


## Trusted modules and opaque values

Register named structured types and explicitly exported compile-time functions through the existing registry. A terminal producer returns a typed circuit plan through a separate controlled callback. Existing borrowed-gate intrinsics remain supported. See the [host optimization extension guide](optimization/host-optimization-extensions.md).

The [QUBO standard library](optimization/qubo-language.md) uses the existing trusted registry. Numeric array signatures can constrain either the rank alone or the rank and numeric leaf type.
