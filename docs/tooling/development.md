# Building and testing editor tooling

## Go compiler and language server

From the repository root:

```sh
gofmt -w .
go test ./...
go vet ./...
go build ./cmd/qasmith
go test ./internal/lsp -run '^$' -bench EditorGraph -benchmem
```

## VS Code extension

The VS Code extension is maintained in a [separate repository](https://github.com/quvly/qasmith-visualstudiocode-extension). Build, test, lint, and VSIX packaging instructions live there. That repository runs its own CI for extension checks.
