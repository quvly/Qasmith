# Qasmith Documentation

Welcome to the Qasmith documentation. Qasmith is a high-level quantum programming language for developers who want familiar functions, modules, and packages.

## What is Qasmith?

Qasmith compiles reusable `.smith` modules to OpenQASM 3.0 and includes a local ideal statevector simulator. The language provides:

- **Familiar syntax** - TypeScript-like syntax with functions, modules, and packages
- **Type safety** - Strong typing with `Int`, `Float`, `Qubit`, and `MeasuredResult`
- **Resource management** - Owned qubits with borrow checking
- **Quantum gates** - Standard gates: `x`, `h`, `cx`, `rx`, `ry`, `rz`
- **Core library** - Built-in quantum operations and utilities
- **Package management** - TOML manifests for dependency management
- **Project initialization** - Quick start with `qasmith init`
- **Optimization** - Built-in QUBO, Ising, and QAOA support
- **Simulation** - Local statevector simulator with shot-based execution

## Quick Start

### 1. Install Qasmith

```bash
# Install (macOS/Linux)
curl -fLO https://github.com/quvly/Qasmith/releases/latest/download/install.sh
sh install.sh
export PATH="$HOME/.local/bin:$PATH"

# Verify installation
qasmith version
```

### 2. Create a New Project

```bash
# Initialize a new project
qasmith init my-quantum-project
cd my-quantum-project
```

This creates a complete project structure:

```
my-quantum-project/
├── qasmith.toml          # Project manifest (TOML format)
├── README.md             # Project documentation
├── .gitignore            # Git ignore rules
└── src/
    └── main.smith        # Entry point
```

### 3. Use the Core Library

Edit `src/main.smith` to use the built-in core library:

```smith
import { bell } from "core/gates";
import { rzz } from "core/rotations";

fn main() -> MeasuredResult {
    let q = qreg(2);
    
    // Create a Bell state using core library
    bell(q[0], q[1]);
    
    // Apply RZZ rotation
    rzz(0.5, q[0], q[1]);
    
    return measure(q);
}
```

### 4. Run Your Program

```bash
qasmith run src/main.smith --shots 1000
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

## Core Library

Qasmith includes a built-in core library with essential quantum operations:

### Available Modules

- **`core/rotations`** - Rotation gates (`rxx`, `rzz`, `crz`)
- **`core/gates`** - Common gate compositions (`bell`, `ghz`, `qft_single`)
- **`core/math`** - Mathematical utilities (`pi`, `e`, `abs`, `max`, `min`, `pow`, `factorial`, `modulo`, `is_even`, `is_odd`)
- **`core/utils`** - Circuit utilities (`identity`, `reset`, `barrier`)
- **`core/measurement`** - Measurement operations (`measure_x`, `measure_y`, `measure_z`)

### Import Syntax

```smith
import { rzz } from "core/rotations";
import { bell } from "core/gates";
import { abs, max } from "core/math";
```

See the [Core Library Reference](core-library.md) for complete API documentation.

## Project Management

### TOML Manifests

Qasmith supports both TOML and JSON manifests, with TOML being the preferred format:

```toml
# qasmith.toml
name = "my-project"
version = "0.1.0"
description = "A quantum computing project"

[dependencies]
quantum-utils = { path = "../shared/quantum-utils" }
```

### Project Initialization

Create new projects with templates:

```bash
# Basic project
qasmith init my-project

# With specific template
qasmith init my-project --template bell
qasmith init my-project --template ghz
qasmith init my-project --template superposition
```

### Import Types

Qasmith supports three types of imports:

1. **Core Library** - Built-in standard library
   ```smith
   import { rzz } from "core/rotations";
   ```

2. **Relative Imports** - Project-local modules
   ```smith
   import { helper } from "./utils/math.smith";
   ```

3. **Package Imports** - External dependencies
   ```smith
   import { feature } from "quantum-utils/module";
   ```

## Documentation Sections

### Getting Started
- [Installation](getting-started/installation.md) - Install Qasmith on your platform
- [Quick Start](getting-started/quickstart.md) - Comprehensive getting started guide
- [First Program](getting-started/first-program.md) - Write and run your first quantum program
- [CLI Overview](getting-started/cli-overview.md) - Learn the command-line interface

### Language Reference
- [Syntax](language.md) - Language syntax and resource rules
- [Circuit Construction](circuit-construction.md) - Build quantum circuits with registers and gates
- [Modules](modules.md) - Organize code with modules and imports
- [Core Library](core-library.md) - Built-in quantum operations and utilities
- [Packages](packages.md) - Create and distribute Qasmith packages

### Optimization
- [QUBO](optimization/qubo.md) - Quadratic Unconstrained Binary Optimization
- [Ising](optimization/ising.md) - Ising model formulation
- [QAOA](optimization/qaoa.md) - Quantum Approximate Optimization Algorithm
- [TSP](optimization/tsp.md) - Travelling Salesman Problem
- [Custom Models](optimization/custom-models.md) - Build your own optimization models

### Architecture
- [Compiler Architecture](architecture.md) - Understand the compiler, IR, and OpenQASM output
- [Extensions](extensions.md) - Extend the Qasmith compiler in Go
- [Roadmap](roadmap.md) - Current limitations and future plans

### Developer Tools
- [VS Code](tooling/vscode.md) - Visual Studio Code extension
- [Language Server](tooling/lsp.md) - LSP integration for other editors

## Built by Quvly

**Quvly created Qasmith and is building an enterprise execution engine for quantum experiments.** Bring a research question or a complex optimization problem. Quvly brings AI-assisted modeling, quantum execution, classical benchmarks, and result verification into one research workspace.

[Explore Quvly](https://quvly.eu) · [Request early access](https://quvly.eu/#early-access)

## License

Qasmith is licensed under [Apache-2.0](https://github.com/quvly/Qasmith/blob/main/LICENSE).
