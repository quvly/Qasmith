# First Program

This guide walks you through writing, compiling, and running your first Qasmith program. You'll create a Bell state circuit and measure the result.

## Prerequisites

- Qasmith CLI installed — see [Installation](installation.md)

## Option 1: Using Project Initialization (Recommended)

The easiest way to start is with `qasmith init`:

```bash
# Create a new project
qasmith init bell-state
cd bell-state
```

This creates a complete project structure with a manifest file:

```
bell-state/
├── qasmith.toml
├── README.md
├── .gitignore
└── src/
    └── main.smith
```

Edit `src/main.smith` to create a Bell state:

```smith
fn main() -> MeasuredResult {
    let a = qubit();
    let b = qubit();
    h(a);
    cx(a, b);
    return measure(a, b);
}
```

## Option 2: Creating a Single File

If you prefer to work with a single file, save the following code in a file called `bell.smith`:

```smith
fn main() -> MeasuredResult {
    let a = qubit();
    let b = qubit();
    h(a);
    cx(a, b);
    return measure(a, b);
}
```

This program:

1. Allocates two qubits `a` and `b`
2. Applies a Hadamard gate to `a`, putting it in superposition
3. Applies a CNOT gate with `a` as control and `b` as target, entangling them
4. Measures both qubits and returns the result

## Check the program

Verify the program is well-formed:

```bash
qasmith check bell.smith
```

`check` reports any errors but does not emit output. Exit status is 0 on success, 1 for errors.

## List dependencies

See the modules this program depends on:

```bash
qasmith deps bell.smith
```

## Compile to OpenQASM 3.0

Compile the program to OpenQASM 3.0:

```bash
qasmith compile bell.smith
```

This writes `bell.qasm` next to the source file. Specify a different output path with `-o`:

```bash
qasmith compile bell.smith -o output.qasm
```

The generated OpenQASM looks like:

```
OPENQASM 3.0;
include "stdgates.inc";

qubit[2] q;
bit[2] c;

h q[0];
cx q[0], q[1];
c[0] = measure q[0];
c[1] = measure q[1];
```

## Run the simulation

Run the program with the built-in statevector simulator:

```bash
qasmith run bell.smith --shots 1000 --seed 17
```

You should see roughly equal counts for `00` and `11`, confirming the Bell state:

```
00: 498
11: 502
```

## Using the Core Library

Qasmith includes a built-in core library with common quantum operations. Let's rewrite our Bell state example using the core library:

```smith
import { bell } from "core/gates";

fn main() -> MeasuredResult {
    let a = qubit();
    let b = qubit();
    bell(a, b);
    return measure(a, b);
}
```

The core library provides reusable quantum operations without requiring manual setup. See the [Core Library Reference](../core-library.md) for all available modules.

## Run a classical program

Qasmith can also run classical programs that compute with integers and floats. Save this as `classical.smith`:

```smith
fn main() -> Int {
    let x = 3 + 5;
    return x;
}
```

Run it with:

```bash
qasmith eval classical.smith
```

Output:

```
8
```

## What's next

- [Quick Start](quickstart.md) — comprehensive getting started guide
- [Core Library](../core-library.md) — explore built-in quantum operations
- [CLI Overview](cli-overview.md) — explore all the Qasmith commands
- [Language Reference](../language.md) — learn the full language syntax
- [Modules](../modules.md) — organize code across multiple files
- [Optimization](../optimization/index.md) — build quantum optimization models
