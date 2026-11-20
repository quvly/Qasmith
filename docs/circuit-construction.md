# Circuit construction

Use a quantum register to allocate several qubits at once. A compile-time loop can apply a gate to each element. This program is available as [examples/register.smith](../examples/register.smith).

```typescript
fn main() -> MeasuredResult {
    let q = qreg(4);
    for i in 0..len(q) {
        h(q[i]);
    }
    return measure(q);
}
```

```sh
go build ./cmd/qasmith
./qasmith check examples/register.smith
./qasmith compile examples/register.smith
./qasmith run examples/register.smith --shots 4096 --seed 17
```

The circuit has four logical qubits and four measurements. In the ideal simulator, each of the 16 bitstrings has probability 1/16. The result string starts with the measurement of `q[0]`.

## Borrow registers in reusable functions

A function can borrow a register of a declared size. It can also accept an array of angles. This is the helper in [examples/quantum/layers.smith](../examples/quantum/layers.smith).

```typescript
pub fn applyLayer(q: QReg<4>, angles: Float[4]) -> Void {
    for i in 0..4 {
        rz(angles[i], q[i]);
    }
}
```

The [layer example](../examples/register_layer.smith) imports this helper and a separate preparation function. The compiler checks both signatures, binds the original logical qubits, and evaluates the angles from the caller's array.

## Build circuits from a matrix

[examples/circuit_builder.smith](../examples/circuit_builder.smith) uses an immutable matrix of Float angles and the imported Smith implementation of RZZ. It visits pairs in increasing index order. The condition selects just the six pairs with `i < j`.

```typescript
for i in 0..len(q) {
    for j in 0..len(q) {
        if i < j {
            rzz(weights[i][j], q[i], q[j]);
        }
    }
}
```

The complete example starts with four H gates and lowers each RZZ to CX, RZ, CX. With the default pass configuration, the backend receives 22 gates and four terminal measurements. Loop indices, matrix access, and branch selection are resolved during compilation.

```sh
./qasmith compile examples/circuit_builder.smith
./qasmith run examples/circuit_builder.smith --shots 4096 --seed 17
```

Use `q[i]` directly in a call such as `cx(q[i], q[j])`. The reference keeps the caller's qubit identity. A separate binding such as `let a = q[0];` is rejected. To select measurement order, use explicit arguments such as `return measure(q[3], q[1]);`.
