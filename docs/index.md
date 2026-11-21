# Qasmith Documentation

Welcome to the Qasmith documentation. Qasmith is a high-level quantum programming language for developers who want familiar functions, modules, and packages.

## What is Qasmith?

Qasmith compiles reusable `.smith` modules to OpenQASM 3.0 and includes a local ideal statevector simulator. The language provides:

- **Familiar syntax** - TypeScript-like syntax with functions, modules, and packages
- **Type safety** - Strong typing with `Int`, `Float`, `Qubit`, and `MeasuredResult`
- **Resource management** - Owned qubits with borrow checking
- **Quantum gates** - Standard gates: `x`, `h`, `cx`, `rx`, `ry`, `rz`
- **Optimization** - Built-in QUBO, Ising, and QAOA support
- **Simulation** - Local statevector simulator with shot-based execution

## Quick Start

Install Qasmith and create your first quantum program:

```bash
# Install (macOS/Linux)
curl -fLO https://github.com/quvly/Qasmith/releases/latest/download/install.sh
sh install.sh
export PATH="$HOME/.local/bin:$PATH"

# Verify installation
qasmith version
```

Create a Bell state circuit:

```typescript
fn main() -> MeasuredResult {
    let a = qubit();
    let b = qubit();
    h(a);
    cx(a, b);
    return measure(a, b);
}
```

Compile and run:

```bash
qasmith compile bell.smith
qasmith run bell.smith --shots 1000
```

## Documentation Sections

### Getting Started
- [Installation](getting-started/installation.md) - Install Qasmith on your platform
- [First Program](getting-started/first-program.md) - Write and run your first quantum program
- [CLI Overview](getting-started/cli-overview.md) - Learn the command-line interface

### Language Reference
- [Syntax](language.md) - Language syntax and resource rules
- [Circuit Construction](circuit-construction.md) - Build quantum circuits with registers and gates
- [Modules](modules.md) - Organize code with modules and imports
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
