# Quick Start Guide

Welcome to Qasmith! This guide will help you get started with quantum programming in Qasmith.

## Installation

### Prerequisites

- Go 1.21 or later (for building from source)
- Or download a pre-built binary from the releases page

### Building from Source

```bash
git clone https://github.com/quvly/Qasmith.git
cd Qasmith
go build -o qasmith ./cmd/qasmith
```

Add the `qasmith` binary to your PATH.

### Verifying Installation

```bash
qasmith version
```

Expected output: `Qasmith v0.10.1`

## Your First Quantum Program

### Step 1: Create a New Project

```bash
qasmith init hello-quantum
cd hello-quantum
```

This creates a new project with the following structure:

```
hello-quantum/
├── qasmith.toml          # Project manifest
├── src/
│   └── main.smith        # Entry point
├── README.md             # Documentation
└── .gitignore            # Git ignore rules
```

### Step 2: Examine the Code

Open `src/main.smith`:

```smith
fn main() -> MeasuredResult {
    let q = qreg(2);
    h(q[0]);
    cx(q[0], q[1]);
    return measure(q);
}
```

This program:
1. Creates a 2-qubit register
2. Applies a Hadamard gate to the first qubit (creating superposition)
3. Applies a CNOT gate (creating entanglement)
4. Measures all qubits

This creates a Bell state, where the qubits are entangled and will always measure the same value.

### Step 3: Run the Program

```bash
qasmith run src/main.smith
```

Expected output:

```
Qasmith Quantum Simulator

Qubits: 2
Shots: 1000

Results:
00: 501 (50.1%)
11: 499 (49.9%)
```

The results show that the qubits are entangled: they always measure the same value (either both 0 or both 1), with roughly 50% probability each.

### Step 4: Compile to OpenQASM

```bash
qasmith compile src/main.smith
```

This generates `src/main.qasm` with the OpenQASM 3.0 representation of your quantum circuit.

## Understanding the Basics

### Quantum Primitives

**Qubits**:
```smith
let q = qubit();      // Single qubit
let q = qreg(5);      // 5-qubit register
```

**Gates**:
```smith
h(q);                 // Hadamard gate
x(q);                 // Pauli X gate
cx(control, target);  // Controlled X (CNOT)
rz(theta, q);         // Z rotation by theta radians
```

**Measurement**:
```smith
return measure(q);    // Measure qubit(s)
```

### Classical Computation

```smith
let x = 5;
let y = x + 3;
let result = y * 2;
```

### Control Flow

```smith
// If statement
if x > 5 {
    h(q);
} else {
    x(q);
}

// For loop
for i in 0..5 {
    h(q[i]);
}
```

## Project Structure

### Recommended Layout

```
my-project/
├── qasmith.toml          # Project manifest
├── src/
│   ├── main.smith        # Entry point
│   ├── algorithms/       # Quantum algorithms
│   │   ├── grover.smith
│   │   └── shor.smith
│   └── utils/            # Utility functions
│       └── math.smith
├── libraries/            # Project libraries
│   └── rotations.smith
└── tests/                # Test files
    └── algorithms_test.smith
```

### Creating a Project with Dependencies

```bash
qasmith init quantum-app
cd quantum-app
```

Edit `qasmith.toml` to add dependencies:

```toml
name = "quantum-app"
version = "0.1.0"

[dependencies]
quantum-utils = { path = "../shared/quantum-utils" }
```

### Importing Modules

```smith
// Relative imports
import { helper } from "./utils/math.smith";
import { rzz } from "../libraries/rotations.smith";

// Package imports
import { feature } from "quantum-utils/module";

// Standard library
import { qaoa } from "std/optimization/qaoa";
```

## Common Patterns

### Bell State

```smith
fn bell_state(a: Qubit, b: Qubit) -> Void {
    h(a);
    cx(a, b);
}
```

### GHZ State

```smith
fn ghz_state(qubits: QReg<3>) -> Void {
    h(qubits[0]);
    cx(qubits[0], qubits[1]);
    cx(qubits[1], qubits[2]);
}
```

### Quantum Fourier Transform (1 qubit)

```smith
fn qft1(q: Qubit) -> Void {
    h(q);
}
```

### Measurement in Different Bases

```smith
// Z basis (default)
return measure(q);

// X basis
fn measure_x(q: Qubit) -> MeasuredResult {
    h(q);
    return measure(q);
}
```

## Using the Standard Library

### QAOA (Quantum Approximate Optimization Algorithm)

```smith
import { qaoa } from "std/optimization/qaoa";
import { tsp } from "std/optimization/tsp";

fn main() -> MeasuredResult {
    let distances = [
        [0, 10, 15, 20],
        [10, 0, 35, 25],
        [15, 35, 0, 30],
        [20, 25, 30, 0]
    ];
    
    let problem = tsp(distances, 0);
    return qaoa(problem, [0.4], [0.3]);
}
```

### QUBO (Quadratic Unconstrained Binary Optimization)

```smith
import { qubo } from "std/optimization/qubo";

fn main() -> MeasuredResult {
    // Define your QUBO problem
    // ...
    return qaoa(problem, gammas, betas);
}
```

## CLI Commands

### Basic Commands

```bash
# Check syntax and types
qasmith check src/main.smith

# Compile to OpenQASM
qasmith compile src/main.smith -o output.qasm

# Run simulation
qasmith run src/main.smith --shots 10000 --seed 42

# View dependency tree
qasmith deps src/main.smith
```

### Optimization Commands

```bash
# Run QAOA optimization
qasmith qaoa --help

# Run TSP solver
qasmith tsp --help
```

### Project Commands

```bash
# Initialize a new project
qasmith init my-project

# Initialize with template
qasmith init my-project --template ghz
```

## VS Code Integration

### Installation

1. Install the Qasmith VS Code extension from the marketplace
2. Set the path to the `qasmith` executable in settings:
   ```json
   {
     "qasmith.executablePath": "/path/to/qasmith"
   }
   ```

### Features

- Syntax highlighting
- Error checking and diagnostics
- Go to definition
- Find references
- Code completion
- Circuit visualization

### Commands

- `Qasmith: Check` - Validate the current file
- `Qasmith: Compile` - Compile to OpenQASM
- `Qasmith: Run` - Run simulation
- `Qasmith: Preview OpenQASM` - Preview the compiled circuit
- `Qasmith: Restart Language Server` - Restart the LSP server

## Troubleshooting

### Common Issues

#### "undeclared package"

**Solution**: Add the package to your `qasmith.toml`:
```toml
[dependencies.my-package]
path = "./path/to/package"
```

#### "path escapes root"

**Solution**: Ensure imports don't use `../` to escape the project root.

#### "circular module dependency"

**Solution**: Refactor to break the import cycle.

### Getting Help

- Check the [Module Resolution Guide](../module-resolution.md)
- Review [example projects](../../examples/)
- Open an issue on GitHub

## Next Steps

1. **Explore Examples**: Check out the [examples directory](../../examples/)
2. **Read the Guide**: Study the [Module Resolution Guide](../module-resolution.md)
3. **Try Optimization**: Experiment with QAOA and TSP examples
4. **Build Projects**: Create your own quantum algorithms
5. **Contribute**: Help improve Qasmith!

## Resources

- [Module Resolution Guide](../module-resolution.md)
- [Core Library Reference](../core-library.md)
- [Example Projects](../../examples/)
- [Contributing Guide](../../CONTRIBUTING.md)
- [GitHub Repository](https://github.com/quvly/Qasmith)

Happy quantum programming!
