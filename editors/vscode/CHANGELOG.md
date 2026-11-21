# Changelog

## 0.11.0

### Features
- **Real-time quantum circuit visualization**: Added a sidebar panel that renders quantum circuits as SVG as you type
  - Parses qubit allocations, single-qubit gates, and multi-qubit controlled gates
  - Renders CNOT/CX with proper notation (control dot + target circle)
  - Supports CZ, SWAP, and other controlled operations
  - Column-based layout algorithm ensures non-overlapping gates
  - Updates automatically with 300ms debounce
  - Activate via the circuit board icon in the activity bar or `Qasmith: Visualize Circuit` command

## 0.9.0

- Added Smith highlighting, language configuration and snippets.
- Integrated the Qasmith language server for diagnostics, completion, hover, navigation, signature help and document symbols.
- Added Qasmith compiler, simulator, optimizer and OpenQASM preview commands.
