# Quantum Library Example

This example demonstrates Qasmith's core standard library and project structure.

## Structure

```
quantum-library/
├── qasmith.toml              # Project manifest
├── README.md                 # This file
└── main.smith                # Entry point
```

## Running

```bash
cd examples/quantum-library
qasmith run main.smith
```

## Features Demonstrated

1. **Core Library Imports**: Using `import { ... } from "core/rotations"`
2. **TOML Manifest**: Modern project configuration format
3. **Bell State**: Quantum entanglement example
4. **RZZ Rotation**: Composite rotation gate

## Core Library Modules

### core/rotations
- `rxx(theta, a, b)` - RXX rotation
- `rzz(theta, a, b)` - RZZ rotation
- `crz(theta, control, target)` - Controlled RZ

### core/gates
- `bell(a, b)` - Bell state preparation
- `ghz(a, b, c)` - GHZ state preparation
- `qft_single(q)` - Single-qubit QFT

### core/math
- `pi()` - π constant
- `e()` - e constant
- `abs(x)` - Absolute value
- `max(a, b)` - Maximum
- `min(a, b)` - Minimum
- `pow(base, exp)` - Power
- `factorial(n)` - Factorial
- `modulo(a, b)` - Modulo
- `is_even(n)` - Even check
- `is_odd(n)` - Odd check

### core/utils
- `identity(q)` - Identity operation
- `reset(q)` - Qubit reset
- `barrier(q)` - Optimization barrier

### core/measurement
- `measure_x(q)` - X-basis measurement
- `measure_y(q)` - Y-basis measurement
- `measure_z(q)` - Z-basis measurement
