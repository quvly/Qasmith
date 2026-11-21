# First Program

This guide walks you through writing, compiling, and running your first Qasmith program. You'll create a Bell state circuit and measure the result.

## Prerequisites

- Qasmith CLI installed — see [Installation](installation.md)

## Create a Bell state

Save the following code in a file called `bell.smith`:

```typescript
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

## Run a classical program

Qasmith can also run classical programs that compute with integers and floats. Save this as `classical.smith`:

```typescript
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

- [CLI Overview](cli-overview.md) — explore all the Qasmith commands
- [Language Reference](../language.md) — learn the full language syntax
- [Modules](../modules.md) — organize code across multiple files
- [Optimization](../optimization/index.md) — build quantum optimization models
