# Project Structure Example

This example demonstrates Qasmith's recommended project structure conventions.

## Structure

```
project-structure/
├── qasmith.toml              # Project manifest
├── README.md                 # This file
├── src/
│   ├── main.smith            # Entry point
│   └── algorithms/
│       └── bell.smith        # Quantum algorithms
├── libraries/
│   └── rotations.smith       # Local project libraries
└── packages/
    └── math-utils/           # Package dependency
        ├── qasmith.toml
        └── arithmetic.smith
```

## Features Demonstrated

1. **Project Manifest**: Uses TOML format (`qasmith.toml`)
2. **Organized Source Code**: Source files in `src/` directory
3. **Local Libraries**: Project-specific libraries in `libraries/`
4. **Package Dependencies**: External packages in `packages/`
5. **Multiple Import Types**:
   - Relative imports: `./libraries/rotations.smith`
   - Package imports: `math-utils/arithmetic`
   - Nested modules: `./algorithms/bell.smith`

## Running

```bash
# Compile to OpenQASM
qasmith compile src/main.smith

# Run simulation
qasmith run src/main.smith

# View dependency tree
qasmith deps src/main.smith
```

## Key Concepts

### Manifest Format

The project manifest (`qasmith.toml`) declares:
- Project name and version
- Package dependencies with paths

### Import Resolution

Imports are resolved in this order:
1. Standard library (`std/`)
2. Declared packages (from manifest)
3. Relative paths (from importing file)

### Project Conventions

- `src/` - Main source code
- `libraries/` - Project-specific libraries
- `packages/` - Package dependencies
- `tests/` - Test files (future)

## Learning More

See the [Module Resolution Guide](../../docs/module-resolution.md) for comprehensive documentation.
