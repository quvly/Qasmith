# Core Library

The Qasmith core library provides essential quantum computing primitives and utility functions. It is embedded in the compiler and available to all Qasmith projects without any configuration.

## Import Syntax

```typescript
import { rzz } from "core/rotations";
import { bell, ghz } from "core/gates";
import { abs, max } from "core/math";
```

## Modules

### `core/rotations`

Rotation gates for quantum circuits.

#### `rxx(theta: Float, a: Qubit, b: Qubit) -> Void`

Apply RXX rotation gate to two qubits.

#### `rzz(theta: Float, a: Qubit, b: Qubit) -> Void`

Apply RZZ rotation gate to two qubits.

#### `crz(theta: Float, control: Qubit, target: Qubit) -> Void`

Apply controlled RZ rotation.

### `core/gates`

Common quantum gate compositions.

#### `bell(a: Qubit, b: Qubit) -> Void`

Create a Bell state (maximally entangled pair).

#### `ghz(a: Qubit, b: Qubit, c: Qubit) -> Void`

Create a GHZ state (three-qubit entangled state).

#### `qft_single(q: Qubit) -> Void`

Apply single-qubit quantum Fourier transform.

### `core/math`

Mathematical utility functions.

#### `pi() -> Float`

Returns the mathematical constant π (3.14159...).

#### `e() -> Float`

Returns Euler's number e (2.71828...).

#### `abs(x: Float) -> Float`

Returns the absolute value of x.

#### `max(a: Float, b: Float) -> Float`

Returns the maximum of two values.

#### `min(a: Float, b: Float) -> Float`

Returns the minimum of two values.

#### `pow(base: Float, exp: Int) -> Float`

Returns base raised to the power of exp.

#### `factorial(n: Int) -> Int`

Returns n factorial (n!).

#### `modulo(a: Int, b: Int) -> Int`

Returns a modulo b.

#### `is_even(n: Int) -> Bool`

Returns true if n is even.

#### `is_odd(n: Int) -> Bool`

Returns true if n is odd.

### `core/utils`

General utility functions.

#### `identity(q: Qubit) -> Void`

Apply identity operation (no-op).

#### `reset(q: Qubit) -> Void`

Reset qubit to |0⟩ state.

#### `barrier(q: QReg) -> Void`

Apply barrier to prevent optimization across boundary.

### `core/measurement`

Measurement utilities.

#### `measure_x(q: Qubit) -> MeasuredResult`

Measure qubit in X basis.

#### `measure_y(q: Qubit) -> MeasuredResult`

Measure qubit in Y basis.

#### `measure_z(q: Qubit) -> MeasuredResult`

Measure qubit in Z basis (computational basis).

## Examples

### Using Rotation Gates

```typescript
import { rzz } from "core/rotations";

fn main() -> MeasuredResult {
    let q = qreg(2);
    h(q[0]);
    rzz(0.5, q[0], q[1]);
    return measure(q);
}
```

### Creating Entangled States

```typescript
import { bell } from "core/gates";

fn main() -> MeasuredResult {
    let q = qreg(2);
    bell(q[0], q[1]);
    return measure(q);
}
```

### Mathematical Operations

```typescript
import { pi, abs } from "core/math";

fn main() -> MeasuredResult {
    let angle = pi() / 4;
    let q = qubit();
    rz(angle, q);
    return measure(q);
}
```

## Design Philosophy

The core library is intentionally minimal and focused on:

- **Essential primitives**: Common quantum gates and operations
- **No external dependencies**: Always available, versioned with compiler
- **Predictable behavior**: Pure quantum operations without side effects
- **Clear naming**: Descriptive names that match quantum computing conventions

For more specialized functionality, create reusable packages and distribute them via the module system.
