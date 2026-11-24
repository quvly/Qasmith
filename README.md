# Qasmith

Qasmith is a high-level quantum programming language for developers who want familiar functions, modules, and packages. It compiles reusable `.smith` modules to OpenQASM 3.0 and includes a local ideal statevector simulator. The CLI builds with the Go toolchain. The VS Code extension adds live diagnostics, navigation and compiler commands.

## Install

Download the installer for your platform from the [latest GitHub release](https://github.com/quvly/Qasmith/releases/latest). Both installers verify the executable's SHA-256 checksum. Releases support x86-64 and ARM64 on macOS and Linux; Windows uses a PowerShell installer with x86-64 and ARM64 support.

**macOS and Linux**

```sh
curl -fLO https://github.com/quvly/Qasmith/releases/latest/download/install.sh
sh install.sh --version v0.10.1
export PATH="$HOME/.local/bin:$PATH"
qasmith version
```

The installer defaults to `~/.local/bin`. Add the export to your shell configuration to keep it across sessions. The `--version` flag is required: pass `--version latest` for the most recent release, or `--version v0.10.1` to pin a specific release. Use `--bin-dir DIRECTORY` to choose a destination.

**Windows PowerShell**

```powershell
Invoke-WebRequest https://github.com/quvly/Qasmith/releases/latest/download/install.ps1 -OutFile install.ps1
.\install.ps1 -Version v0.10.1
qasmith version
```

The installer defaults to `%LOCALAPPDATA%\Qasmith\bin` and adds it to your user PATH. The `-Version` parameter is required: pass `-Version latest` for the most recent release, or `-Version v0.10.1` to pin a specific release. Use `-BinDir DIRECTORY` or `-NoPath` to override the defaults.

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

## Where to go next

- **Optimization:** The Go packages under `pkg/optimization` build and evaluate QUBO and Ising models, QAOA circuits, and TSP solvers. See the [QUBO guide](docs/optimization/qubo.md), [QAOA guide](docs/optimization/qaoa.md), and [TSP guide](docs/optimization/tsp.md).
- **Smith examples:** The root [program.smith](program.smith) builds a QUBO and returns a QAOA circuit. The [TSP source example](tsp.smith) solves the travelling salesman problem. Browse more in [examples/](examples/).
- **Real-world TSP:** Load CSV distance matrices with `qasmith csv-to-smith`. See the [TSP CSV example](examples/tsp-csv/README.md).
- **Language and modules:** [Circuit construction](docs/circuit-construction.md), [language syntax](docs/language.md), [modules and imports](docs/modules.md), [packages](docs/packages.md).
- **Architecture:** [Compiler architecture](docs/architecture.md), [extending in Go](docs/extensions.md), [limitations](docs/roadmap.md).
- **Editor tooling:** Install from the [VS Code Marketplace](https://marketplace.visualstudio.com/items?itemName=Quvly.qasmith-language) or use the [language server](docs/tooling/lsp.md) with other editors.

## Status and claims

The following claims are backed by tests or documentation in this repository:

- **Bell state simulation produces only `00` and `11` with roughly equal counts.**
  Statevector cross-check: `tests/cross-check/cross_check_test.go:TestCorpusStatevectors/bell`.
  Statistical test: `tests/cross-check/statistical_test.go:TestChiSquareBellState`.
- **Classical evaluation of `examples/classical.smith` returns `8`.**
  Integration test: `tests/integration/cli_test.go:TestCLI`.
- **The compiler rejects programs that copy qubit references, reuse a qubit
  after measurement, or pass the same qubit as both CX operands.**
  Semantic tests: `pkg/qasmith/compiler_test.go` and `pkg/qasmith/rzz_test.go`.
- **Generated OpenQASM is syntactically valid per the reference parser.**
  CI job: `openqasm-validation` in `.github/workflows/ci.yml`.
  Historical evidence: `docs/internal/phase-2a-compliance.md`.
- **The simulator statevector matches analytical computation for 20 circuits
  (Bell, GHZ, rotations, registers, controlled gates).**
  Cross-check corpus: `tests/cross-check/corpus/` and
  `tests/cross-check/cross_check_test.go`.

Other claims in the README are documentation of intent and should not be read
as guarantees until corresponding tests are added.

## Documentation

- [Circuit construction with registers, matrices, and static control](docs/circuit-construction.md)
- [Language syntax and resource rules](docs/language.md)
- [Modules, visibility, and imports](docs/modules.md)
- [Creating Your First Qasmith Package](docs/packages.md)
- [Compiler architecture, IR, OpenQASM, and simulation](docs/architecture.md)
- [Extending the Qasmith Compiler in Go](docs/extensions.md)
- [Limitations and roadmap](docs/roadmap.md)
- [Contributing](CONTRIBUTING.md)

Qasmith is licensed under [Apache-2.0](LICENSE).

## About Quvly

**Quvly created Qasmith and is building an enterprise execution engine for quantum experiments.** Quvly brings AI-assisted modeling, quantum execution, classical benchmarks, and result verification into one research workspace. The goal is to make each experiment reproducible and give teams the evidence they need to evaluate its results. [Explore Quvly](https://quvly.eu)

Quvly is in private research alpha, and early access is open. Qasmith is open source and available today. [Request early access at Quvly](https://quvly.eu/#early-access)

## Commercial relationship

Qasmith is an open-source project licensed under the Apache License 2.0. It was
created by [Quvly](https://quvly.eu), which also maintains a separate commercial
quantum experiment execution engine. The commercial product is not part of this
repository and is not covered by the Apache-2.0 license. Contributions to
Qasmith are welcome under the terms of the license; see
[CONTRIBUTING.md](CONTRIBUTING.md).
