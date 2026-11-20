# Contributing

Build and test from the repository root using Go 1.22 or later.

```sh
gofmt -w cmd internal pkg tests
go test ./...
go vet ./...
go build ./cmd/qasmith
```

Include a minimal `.smith` example for changes to syntax or semantics. Keep backend changes grounded in the IR and add coverage for any change to generated instructions or measurement order. Module tests should exercise imports through real files. Describe user-facing behavior changes in the guides under `docs/`.

## Build a release

GitHub Actions runs the Go and VS Code checks before GoReleaser publishes a tagged release. Set the CLI version, then push the matching annotated tag. GoReleaser publishes six executables, per-file SHA-256 checksums and both installers. Run the **Build executables** workflow manually from a branch to produce snapshot artifacts without publishing a release.

To verify the packaging locally, run:

```sh
goreleaser check
goreleaser release --snapshot --clean
```
