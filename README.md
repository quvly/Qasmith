# Qasmith

Qasmith is a high-level quantum programming language for developers who want familiar functions, modules, and packages. Version 0.9.0 compiles reusable `.smith` modules to OpenQASM 3.0 and includes a local ideal statevector simulator. The CLI builds with the Go toolchain. The VS Code extension adds live diagnostics, navigation and compiler commands.

## Install

Download the installer for your platform from the [latest GitHub release](https://github.com/quvly/Qasmith/releases/latest). Both installers verify the executable's SHA-256 checksum. Releases support x86-64 and ARM64 on all three platforms.

**macOS and Linux**

```sh
curl -fLO https://github.com/quvly/Qasmith/releases/latest/download/install.sh
sh install.sh
export PATH="$HOME/.local/bin:$PATH"
qasmith version
```

The installer defaults to `~/.local/bin`. Add the export to your shell configuration to keep it across sessions. Use `--version v0.9.0` to pin a release or `--bin-dir DIRECTORY` to choose a destination.

**Windows PowerShell**

```powershell
Invoke-WebRequest https://github.com/quvly/Qasmith/releases/latest/download/install.ps1 -OutFile install.ps1
.\install.ps1
qasmith version
```

The installer defaults to `%LOCALAPPDATA%\Qasmith\bin` and adds it to your user PATH. Use `-Version v0.9.0`, `-BinDir DIRECTORY`, or `-NoPath` to override the defaults.

### Build from source

With Go 1.22 or later, build from the repository and run the resulting CLI:

```sh
go build ./cmd/qasmith
./qasmith version
```

Use `go install ./cmd/qasmith` to install into your Go binary directory.

## First program

Save this as a `.smith` file:

```typescript
fn main() -> MeasuredResult {
    let a = qubit();
    let b = qubit();
    h(a);
    cx(a, b);
    return measure(a, b);
}
```

The repository includes this circuit in [examples/program.smith](examples/program.smith). The root [bell.smith](bell.smith) assembles the same circuit through a local package with a private helper.

```sh
./qasmith check bell.smith
./qasmith deps bell.smith
./qasmith compile bell.smith
./qasmith compile bell.smith -o output.qasm
./qasmith run bell.smith --shots 1000 --seed 17
./qasmith eval examples/classical.smith
./qasmith run examples/rotations.smith --shots 100 --seed 17
./qasmith run examples/circuit_builder.smith --shots 4096 --seed 17
```

`compile` writes beside the source by default. The Bell example produces only `00` and `11`, with roughly equal counts. The classical example prints `8`. The [rotation example](examples/rotations.smith) passes a `Float` angle to an imported function and applies `rx`, `ry`, and `rz`. Simulation defaults to 1,000 shots and a varying seed. Exit status is 0 on success, 1 for a compilation or execution error, and 2 for invalid CLI usage.

## Standalone optimization libraries

The Go packages under `pkg/optimization` build and evaluate QUBO and Ising models, convert between their energy conventions, add generic penalties, serialize models, and solve small QUBOs by bounded exhaustive search.

```sh
go run ./examples/optimization/qubo
```

See the [QUBO guide](docs/optimization/qubo.md) and [optimization architecture](docs/optimization/architecture.md).

## Hybrid QAOA

Run the reproducible [QAOA example](examples/optimization/qaoa/README.md) or use the QUBO JSON CLI to compile, sample, and optimize generic circuits.

```sh
./qasmith qaoa optimize examples/optimization/qaoa/problem.json --layers 1 --max-evals 200 --restarts 3 --seed 42 -o optimized.qasm
```

See the [QAOA guide](docs/optimization/qaoa.md) for exact and sampled objectives, resource limits, and Go integration.

## Travelling salesman application

The TSP CLI takes a distance matrix through fixed-start QUBO encoding, generic QAOA, and strict route decoding. The independent classical solver gives a reference for small instances.

```sh
go run ./examples/optimization/tsp
```

See the [TSP guide](docs/optimization/tsp.md) for the API, limits, and interpretation of feasibility and route metrics.

## Optimization in Smith

The root [program.smith](program.smith) builds and constrains a generic QUBO, then returns a QAOA circuit. The [TSP source example](tsp.smith) remains available. Compile, sample, and optimize it through the general CLI.

```sh
./qasmith check program.smith
./qasmith run program.smith --shots 1000 --seed 42
./qasmith optimize program.smith --max-evals 200 --restarts 3 --shots 1000 --seed 42
```

Start with [Building Your First Quantum Optimization Model in Qasmith](docs/optimization/custom-models.md) or browse the [QUBO examples](examples/optimization/qubo-language/README.md).

## Documentation

- [Circuit construction with registers, matrices, and static control](docs/circuit-construction.md)
- [Language syntax and resource rules](docs/language.md)
- [Modules, visibility, and imports](docs/modules.md)
- [Creating Your First Qasmith Package](docs/packages.md)
- [Compiler architecture, IR, OpenQASM, and simulation](docs/architecture.md)
- [Extending the Qasmith Compiler in Go](docs/extensions.md)
- [Phase 3C compliance report](docs/phase-3c-compliance.md)
- [Phase 3B compliance report](docs/phase-3b-compliance.md)
- [Phase 3A compliance report](docs/phase-3a-compliance.md)
- [Phase 2D compliance report](docs/phase-2d-compliance.md)
- [Phase 2C compliance report](docs/phase-2c-compliance.md)
- [Phase 2B compliance report](docs/phase-2b-compliance.md)
- [Limitations and roadmap](docs/roadmap.md)
- [Contributing](CONTRIBUTING.md)

Qasmith is licensed under [Apache-2.0](LICENSE).

## Editor tooling

Install the local VSIX and follow the [Visual Studio Code guide](docs/tooling/vscode.md). Other editors can connect to the standalone [Qasmith language server](docs/tooling/lsp.md)..
