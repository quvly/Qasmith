# Troubleshooting

- **Highlighting works but live language features are missing.** In a trusted workspace, confirm `qasmith version` returns the expected version. GUI-launched applications may see a different PATH from your terminal. Set `qasmith.executablePath` in User Settings and use **Qasmith: Restart Language Server**.
- **The executable is inside your project.** Automatic discovery skips workspace binaries, including symlink targets within the workspace. An explicitly selected absolute executable in User Settings is allowed in a trusted workspace.
- **The compiler refuses to run in Restricted Mode.** Highlighting and snippets remain available. Grant Workspace Trust only for a project you trust. Qasmith does not download packages, load source-specified native plugins or transmit source to a cloud service.
- **A command reports an unsaved file.** Choose **Save and Continue** or save the affected sources and retry. Open **Qasmith: Show Compiler Output** for errors or process exit codes.
- **Find References omits an entry point.** Open that entry as well. References are limited to the current graph and graphs of open documents.
- **The server stopped.** Check the Qasmith output channel, optionally enable `qasmith.lsp.trace`, then restart from the Command Palette.
