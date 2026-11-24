# Contributing to Qasmith

Thank you for your interest in contributing to Qasmith! This guide helps you get started.

## Code of conduct

Be respectful and constructive in all interactions.

## How to contribute

### Reporting bugs

Open an issue on [GitHub](https://github.com/quvly/Qasmith/issues) with:

- A clear description of the bug
- Steps to reproduce
- Expected vs actual behavior
- Qasmith version (`qasmith version`)
- Your OS and architecture

### Suggesting features

Open an issue describing the feature and why it would be useful.

### Pull requests

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Make your changes
4. Add or update tests
5. Run the test suite: `go test ./...`
6. Commit with a clear message
7. Push and open a pull request

## Development setup

### Prerequisites

- Go 1.22 or later
- Git

### Build

```bash
git clone https://github.com/quvly/Qasmith.git
cd Qasmith
go build ./cmd/qasmith
```

### Test

```bash
go test ./...
```

### Run examples

```bash
./qasmith run examples/program.smith --shots 1000
```

## Project structure

- `cmd/qasmith/` — CLI entry point
- `pkg/` — public Go packages
- `internal/` — internal compiler implementation
- `stdlib/` — standard library modules
- `docs/` — documentation
- `examples/` — example programs
- `tests/` — integration tests

## Documentation

This documentation is built with [MkDocs Material](https://squidfunk.github.io/mkdocs-material/) and hosted on GitHub Pages.

To build locally:

```bash
pip install -r requirements-docs.txt
mkdocs serve
```

Visit `http://localhost:8000` to preview changes.

## Style guide

- Follow Go conventions for Go code
- Use `snake_case` for Smith code identifiers
- Add documentation comments for public APIs
- Keep functions small and focused

## Phase compliance reports

Internal development phase compliance reports are in [`docs/internal/`](internal/). These are historical self-assessments from each development phase. For each claim in them, the linked test or evidence file proves it.

## License

By contributing, you agree that your contributions will be licensed under the Apache-2.0 License.
