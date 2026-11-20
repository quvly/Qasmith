# Building and testing editor tooling

From the repository root:

```sh
gofmt -w .
go test ./...
go vet ./...
go build ./cmd/qasmith
go test ./internal/lsp -run '^$' -bench EditorGraph -benchmem
```

Build and package the extension with Node.js, npm and the pinned official VS Code packaging tool:

```sh
cd editors/vscode
npm ci
npm run compile
npm run lint
npm test
npx vsce package --out qasmith-language-0.9.0.vsix
```

Run real extension-host tests from the repository root. The runner uses an isolated profile and temporary workspace. Set `VSCODE_EXECUTABLE` to the desktop VS Code executable if using an existing installation. Without that variable the test tooling obtains VS Code.

```sh
QASMITH_TEST_CLI="$PWD/qasmith" npm --prefix editors/vscode run test:integration
```

The VSIX includes the TypeScript extension's compiled JavaScript, its production dependencies and language assets. The Go executable is installed separately. Install the VSIX locally to verify packaging. [VS Code packaging documentation](https://code.visualstudio.com/api/working-with-extensions/publishing-extension)
