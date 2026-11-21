# CLI Overview

The Qasmith CLI provides commands to check, compile, run, and optimize Qasmith programs. This guide covers every command and common options.

## Global options

All commands accept:

- `-h, --help` — show help for a command
- Exit status is 0 on success, 1 for compilation or execution errors, and 2 for invalid CLI usage.

## Commands

### `qasmith version`

Print the installed version:

```bash
qasmith version
```

### `qasmith check <file>`

Check a program for errors without producing output:

```bash
qasmith check bell.smith
```

### `qasmith deps <file>`

List module dependencies:

```bash
qasmith deps bell.smith
```

### `qasmith compile <file>`

Compile a `.smith` file to OpenQASM 3.0:

```bash
qasmith compile bell.smith
qasmith compile bell.smith -o output.qasm
```

By default, output is written beside the source file with a `.qasm` extension.

### `qasmith run <file>`

Compile and run a program with the built-in statevector simulator:

```bash
qasmith run bell.smith --shots 1000 --seed 17
```

**Options:**

- `--shots N` — number of measurement shots (default: 1000)
- `--seed N` — random seed for reproducibility (default: varies)

### `qasmith eval <file>`

Evaluate a classical program:

```bash
qasmith eval examples/classical.smith
```

### `qasmith optimize <file>`

Compile and run a QAOA optimization problem:

```bash
qasmith optimize program.smith --max-evals 200 --restarts 3 --shots 1000 --seed 42
```

**Options:**

- `--max-evals N` — maximum function evaluations
- `--restarts N` — number of optimizer restarts
- `--shots N` — measurement shots per evaluation
- `--seed N` — random seed
- `-o <file>` — write optimized OpenQASM output

### `qasmith qaoa optimize <problem>`

Run QAOA optimization on a JSON problem file:

```bash
qasmith qaoa optimize problem.json --layers 1 --max-evals 200 --restarts 3 --seed 42 -o optimized.qasm
```

## Examples

### Bell state

```bash
qasmith check bell.smith
qasmith compile bell.smith
qasmith run bell.smith --shots 1000 --seed 17
```

### Rotations

```bash
qasmith run examples/rotations.smith --shots 100 --seed 17
```

### Circuit builder

```bash
qasmith run examples/circuit_builder.smith --shots 4096 --seed 17
```

### QAOA optimization

```bash
go run ./examples/optimization/qaoa
```

### TSP

```bash
go run ./examples/optimization/tsp
```

## Environment variables

The CLI reads no special environment variables. Paths are resolved relative to the working directory.

## Next steps

- [Language Reference](../language.md) — learn the full language syntax
- [Circuit Construction](../circuit-construction.md) — build quantum circuits
- [Optimization](../optimization/index.md) — solve optimization problems
