# Installation

Qasmith is available for macOS, Linux, and Windows. Both installers verify the executable's SHA-256 checksum.

Releases support **x86-64 and ARM64 on macOS and Linux**. Windows uses a PowerShell installer with x86-64 and ARM64 support.

## Download the installer

Download the installer for your platform from the [latest GitHub release](https://github.com/quvly/Qasmith/releases/latest).

### macOS and Linux

```bash
curl -fLO https://github.com/quvly/Qasmith/releases/latest/download/install.sh
sh install.sh
export PATH="$HOME/.local/bin:$PATH"
qasmith version
```

The installer defaults to `~/.local/bin`. Add the `export` line to your shell configuration (`~/.bashrc`, `~/.zshrc`, etc.) to keep the binary on your PATH across sessions.

Use `--version v0.10.1` to pin a release or `--bin-dir DIRECTORY` to choose a destination.

### Windows PowerShell

```powershell
Invoke-WebRequest https://github.com/quvly/Qasmith/releases/latest/download/install.ps1 -OutFile install.ps1
.\install.ps1
qasmith version
```

The installer defaults to `%LOCALAPPDATA%\Qasmith\bin` and adds it to your user PATH. Use `-Version v0.10.1`, `-BinDir DIRECTORY`, or `-NoPath` to override the defaults.

## Build from source

With Go 1.22 or later, build from the repository:

```bash
git clone https://github.com/quvly/Qasmith.git
cd Qasmith
go build ./cmd/qasmith
./qasmith version
```

Use `go install ./cmd/qasmith` to install into your Go binary directory.

## Verify the installation

After installing, verify the CLI works:

```bash
qasmith version
```

You should see output showing the installed version, such as `Qasmith v0.10.1`.

## Editor support

Install the Qasmith VS Code extension from the [Visual Studio Code Marketplace](https://marketplace.visualstudio.com/items?itemName=Quvly.qasmith-language) for live diagnostics, code navigation, and compiler commands.

For other editors, see the [Language Server](../tooling/lsp.md) guide.

## Next steps

- [Write your first program](first-program.md) to get hands-on with Qasmith
- [Learn the CLI](cli-overview.md) to compile, run, and optimize your programs
