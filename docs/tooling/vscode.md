# Developing Qasmith Programs in Visual Studio Code

Install VS Code 1.90 or newer and Qasmith CLI 0.9.0. From the Qasmith checkout, build the CLI with `go build ./cmd/qasmith` and install the extension with:

```sh
code --install-extension editors/vscode/qasmith-vscode-0.9.0.vsix
```

Use **Extensions → Install from VSIX…** if the `code` command is unavailable. Put Qasmith on PATH or set its absolute path in User Settings. Open a project directory and grant Workspace Trust to enable compiler processes. Highlighting, bracket matching, comment toggling, indentation, folding and snippets are available in Restricted Mode.

## Create a QUBO circuit

Save the following as `program.smith`. The same example is available at [matrix.smith](../../examples/optimization/qubo-language/matrix.smith).

```smith
import { fromMatrix } from "std/optimization/qubo";
import { qaoa } from "std/optimization/qaoa";

fn main() -> MeasuredResult {
    let model = fromMatrix([[-2.0, 4.0, 0.0], [0.0, 3.0, -1.0], [0.0, 0.0, -0.5]], 1.0);
    return qaoa(model, [0.4], [0.3]);
}
```

Hover over `fromMatrix` to see its signature and documentation. Use Trigger Parameter Hints inside the `qaoa` call. For a local imported function, use **Go to Definition** to open the defining file and **Find All References** to see references in the loaded graphs. Open Outline to see functions in the current file. Use the `fn`, `main`, `qreg`, `for`, `import` and `qaoa` snippets to start edits.

Try changing `model` to an undefined name in the return expression. The Problems view shows a compiler diagnostic while the file is still unsaved. Restore the name, then use the Command Palette:

| Qasmith command | Result |
| --- | --- |
| Check Current File | Runs the compiler's check and reports the result in Qasmith output. |
| Compile Current File | Writes `program.qasm` by default and asks before replacing an existing file. |
| Run Current File | Samples the compiled circuit with the configured shots and seed. |
| Optimize Current File | Searches for QAOA angles with the configured evaluation budget, then reports results. |
| Show Generated OpenQASM | Compiles and opens the actual output beside the source. |
| Restart Language Server | Stops the old server and retries startup. |
| Show Compiler Output | Opens the Qasmith output channel. |

The commands prompt before saving modified Smith or manifest files. Cancel to leave those changes unsaved. The LSP uses the in-memory Smith buffers. CLI commands use saved files.

## Settings

| Setting under `qasmith` | Default |
| --- | --- |
| `executablePath` | Empty. Uses PATH. Set an absolute trusted path in User Settings. |
| `lsp.enabled` | `true` |
| `lsp.trace` | `off`. Also accepts `messages` and `verbose`. |
| `diagnostics.enabled` | `true` |
| `diagnostics.debounceMs` | `200`, with a range of 0–2000. |
| `run.shots` | `1000`, with a range of 1–1,000,000. |
| `run.seed` | `42`. Accepts safe integers. |
| `optimize.maxEvaluations` | `200`, with a range of 1–100,000. |
| `optimize.restarts` | `3`, with a range of 0–64. |
| `optimize.evaluationMode` | `exact`. Also accepts `sampled`. |
| `compile.outputPath` | Empty. Relative paths resolve beside the source. |
| `verbose` | `false`. Logs process arguments when enabled. |

See [Troubleshooting](troubleshooting.md) for executable discovery and Workspace Trust.
