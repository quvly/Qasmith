# Qasmith Module Resolution and Package Management

This document describes Qasmith's module resolution system, project structure conventions, and package management features.

## Table of Contents

- [Project Structure](#project-structure)
- [Module Resolution](#module-resolution)
- [Import Syntax](#import-syntax)
- [Project Manifest](#project-manifest)
- [Package Management](#package-management)
- [Standard Library](#standard-library)
- [Best Practices](#best-practices)
- [Migration Guide](#migration-guide)

## Project Structure

A typical Qasmith project follows this structure:

```
my-quantum-project/
├── qasmith.toml          # Project manifest (preferred)
├── qasmith.json          # Alternative manifest (JSON format)
├── src/
│   ├── main.smith        # Entry point
│   ├── algorithms/
│   │   ├── grover.smith
│   │   └── shor.smith
│   └── utils/
│       └── math.smith
├── libraries/            # Local project libraries
│   └── custom.smith
├── tests/
│   └── algorithms_test.smith
└── README.md
```

### Creating a New Project

Use the `qasmith init` command to create a new project:

```bash
qasmith init my-project
```

This creates:
- `qasmith.toml` - Project manifest
- `src/main.smith` - Entry point with a Bell state example
- `README.md` - Project documentation
- `.gitignore` - Git ignore rules

You can specify a template:

```bash
qasmith init my-project --template bell
qasmith init my-project --template ghz
qasmith init my-project --template superposition
```

### Entry Point

The entry point is the `.smith` file that contains the `main()` function. By convention, this is `src/main.smith`, but you can specify a different entry point when running commands:

```bash
qasmith run src/main.smith
qasmith compile src/main.smith -o output.qasm
```

## Module Resolution

Qasmith's module resolution system determines how `import` statements are resolved to actual files. The resolver follows a specific order and enforces security constraints.

### Resolution Order

When resolving an import, Qasmith checks the following locations in order:

1. **Standard Library** (`std/` prefix)
   - Built into the compiler
   - Always available
   - Version matches compiler version

2. **Declared Dependencies** (package imports)
   - Look up package name in `qasmith.toml` or `qasmith.json` dependencies
   - Resolve to package root directory
   - Resolve module path within package

3. **Relative Imports** (`./` or `../` prefix)
   - Resolve relative to importing file's directory
   - Validate within project or package root
   - Prevent path traversal attacks

### Resolution Algorithm

```
For import "std/module":
  1. Check if module is registered in standard library
  2. Return trusted module reference
  3. Fail if not found

For import "package/module":
  1. If package == "std", use standard library
  2. Else look up package in manifest dependencies
  3. Resolve module path within package root
  4. Validate path is within package root
  5. Load and cache module

For import "./path" or "../path":
  1. Resolve relative to importer's directory
  2. Determine package root (project or dependency)
  3. Validate resolved path is within package root
  4. Load and cache module
```

### Security Constraints

The module resolver enforces several security constraints:

- **Path Traversal Prevention**: Imports cannot escape their package root
- **Symlink Validation**: Symlinks are resolved and validated
- **No Absolute Paths**: Imports cannot use absolute paths
- **Circular Dependency Detection**: Circular imports are detected and reported
- **Case-Insensitive Collision Detection**: Prevents issues on case-insensitive filesystems

### Caching

Modules are cached by their canonical path to prevent:
- Duplicate loading of the same file
- Infinite loops from circular imports
- Inconsistent module identity

## Import Syntax

Qasmith supports three types of imports:

### Relative Imports

Import modules relative to the current file's location:

```smith
// Import from the same directory
import { function1 } from "./module.smith";

// Import from a parent directory
import { function2 } from "../libraries/utils.smith";

// Import from a subdirectory
import { function3 } from "./algorithms/grover.smith";
```

### Package Imports

Import from declared dependencies:

```smith
// Import from a package
import { feature } from "package-name/module";

// Import with alias
import { long_name as short } from "package/utils";
```

### Standard Library Imports

Import from the Qasmith standard library:

```smith
// Import optimization modules
import { qaoa } from "std/optimization/qaoa";
import { tsp } from "std/optimization/tsp";
import { qubo } from "std/optimization/qubo";
```

### Import Specifiers

You can import multiple symbols and use aliases:

```smith
import {
    function1,
    function2 as func2,
    function3
} from "./module.smith";
```

## Project Manifest

The project manifest defines project metadata and dependencies. Qasmith supports both TOML and JSON formats, with TOML being preferred.

### TOML Format (Preferred)

Create a `qasmith.toml` file:

```toml
name = "my-quantum-project"
version = "0.1.0"

[dependencies]
quantum-utils = { path = "../shared/quantum-utils" }
my-library = { path = "./libs/my-library" }
```

### JSON Format (Alternative)

Create a `qasmith.json` file:

```json
{
  "name": "my-quantum-project",
  "version": "0.1.0",
  "dependencies": {
    "quantum-utils": { "path": "../shared/quantum-utils" },
    "my-library": { "path": "./libs/my-library" }
  }
}
```

### Manifest Fields

#### Required Fields

- `name`: Project name (must match pattern `^[A-Za-z][A-Za-z0-9_-]*$`)
- `version`: Semantic version (e.g., `"0.1.0"`, `"1.2.3"`)

#### Optional Fields

- `dependencies`: Map of package dependencies

### Dependency Declaration

Dependencies are declared with a name and a local path:

```toml
[dependencies]
package-name = { path = "./path/to/package" }
```

**Constraints**:
- Package names must match pattern `^[A-Za-z][A-Za-z0-9_-]*$`
- Package names are case-insensitive
- The name `"std"` is reserved for the standard library
- Dependencies cannot overlap with the project root
- Dependencies cannot overlap with each other
- Paths can be relative or absolute

### Dependency Structure

A dependency package should have its own manifest:

```
my-library/
├── qasmith.toml
├── module1.smith
└── module2.smith
```

The dependency's manifest is not required for path-based dependencies, but recommended for clarity.

## Package Management

### Current Capabilities (Phase 1)

Qasmith currently supports:

✅ **Local Path Dependencies**
- Declare dependencies with local paths
- Relative or absolute paths
- Version validation

✅ **Standard Library**
- Built-in modules for common quantum operations
- Import via `std/` namespace

✅ **Project Initialization**
- `qasmith init` command
- Project templates
- Standard directory structure

### Future Capabilities (Planned)

The following features are planned for future releases:

#### Phase 2: Enhanced Project Structure
- Default entry point detection from manifest
- Project-aware CLI commands (compile without specifying file)
- Build configuration in manifest

#### Phase 3: Package Registry Foundation
- Version constraints for dependencies
- Lockfile (`qasmith.lock`) for reproducible builds
- `qasmith add` command for adding dependencies
- `qasmith update` command for updating dependencies

#### Phase 4: Package Registry
- Central package registry for community packages
- Automatic dependency resolution
- Package discovery and search
- Integrity verification

#### Phase 5: Advanced Features
- Workspace support for multi-package projects
- Conditional dependencies
- Build scripts and custom tasks
- Development dependencies

## Standard Library

The Qasmith standard library provides trusted modules for common quantum operations.

### Available Modules

#### Optimization

```smith
import { qaoa } from "std/optimization/qaoa";
import { tsp } from "std/optimization/tsp";
import { qubo } from "std/optimization/qubo";
```

**QAOA (Quantum Approximate Optimization Algorithm)**:
- Solves combinatorial optimization problems
- Accepts optimization problem definitions
- Returns quantum circuits for QAOA

**TSP (Traveling Salesman Problem)**:
- Encodes TSP instances as optimization problems
- Converts distance matrices to QUBO formulations

**QUBO (Quadratic Unconstrained Binary Optimization)**:
- Tools for building QUBO models
- Constraint handling
- Objective function construction

### Using Standard Library

```smith
import { qaoa } from "std/optimization/qaoa";
import { tsp } from "std/optimization/tsp";

fn main() -> MeasuredResult {
    let distances = [
        [0, 10, 15, 20],
        [10, 0, 35, 25],
        [15, 35, 0, 30],
        [20, 25, 30, 0]
    ];
    
    let problem = tsp(distances, 0);
    return qaoa(problem, [0.4], [0.3]);
}
```

## Best Practices

### Project Organization

1. **Use `src/` for source code**: Keep your main source files in `src/` for clarity
2. **Organize by feature**: Group related modules in subdirectories
3. **Use descriptive names**: Choose clear, descriptive names for modules and functions
4. **Document public APIs**: Use comments to document public functions

### Import Guidelines

1. **Prefer package imports over relative imports**: Use package imports for better clarity
2. **Keep import paths short**: Avoid deeply nested module structures
3. **Use aliases for clarity**: When importing multiple functions, use aliases to avoid name collisions
4. **Group imports logically**: Organize imports by standard library, packages, and relative imports

### Dependency Management

1. **Use semantic versioning**: Follow semver for your project versions
2. **Pin dependency paths**: Use relative paths for reproducible builds
3. **Test dependency updates**: Always test after updating dependencies
4. **Document dependencies**: List dependencies in your README

### Security

1. **Review dependencies**: Only use dependencies from trusted sources
2. **Validate paths**: Ensure dependency paths are within expected directories
3. **Avoid absolute paths**: Use relative paths for portability
4. **Test on multiple platforms**: Ensure paths work on Windows, macOS, and Linux

## Migration Guide

### From JSON to TOML

If you have an existing `qasmith.json` manifest, you can migrate to TOML:

**Before** (`qasmith.json`):
```json
{
  "name": "my-project",
  "version": "0.1.0",
  "dependencies": {
    "my-lib": { "path": "./libs/my-lib" }
  }
}
```

**After** (`qasmith.toml`):
```toml
name = "my-project"
version = "0.1.0"

[dependencies.my-lib]
path = "./libs/my-lib"
```

Both formats are supported, so migration is optional.

### From Flat Structure to Organized Structure

If you have a flat project structure:

```
project/
├── main.smith
├── utils.smith
└── algorithms.smith
```

You can reorganize to:

```
project/
├── qasmith.toml
├── src/
│   ├── main.smith
│   ├── utils.smith
│   └── algorithms.smith
└── README.md
```

Update your imports accordingly:

```smith
// Before
import { helper } from "./utils.smith";

// After (if you move main.smith to src/)
import { helper } from "./utils.smith";  // Still works with relative imports
```

## Troubleshooting

### Common Errors

#### "undeclared package"

**Cause**: You're trying to import from a package that's not declared in your manifest.

**Solution**: Add the package to your `qasmith.toml`:

```toml
[dependencies.my-package]
path = "./path/to/package"
```

#### "path escapes root"

**Cause**: An import is trying to access a file outside the project or package root.

**Solution**: Ensure your import paths don't use `../` to escape the project root. Restructure your project if needed.

#### "circular module dependency"

**Cause**: Two or more modules import each other, creating a cycle.

**Solution**: Refactor to break the cycle. Extract shared code into a separate module.

#### "expected a regular .smith file"

**Cause**: The resolved path doesn't point to a `.smith` file.

**Solution**: Ensure your import path ends in `.smith` for relative imports.

### Debugging Imports

Use the `qasmith deps` command to see the resolved dependency tree:

```bash
qasmith deps src/main.smith
```

This shows all modules that will be loaded, helping you identify resolution issues.

## Examples

### Example 1: Simple Project

```
hello-quantum/
├── qasmith.toml
└── src/
    └── main.smith
```

**qasmith.toml**:
```toml
name = "hello-quantum"
version = "0.1.0"
```

**src/main.smith**:
```smith
fn main() -> MeasuredResult {
    let q = qubit();
    h(q);
    return measure(q);
}
```

### Example 2: Project with Libraries

```
quantum-project/
├── qasmith.toml
├── src/
│   ├── main.smith
│   └── algorithms/
│       └── grover.smith
└── libraries/
    └── utils.smith
```

**qasmith.toml**:
```toml
name = "quantum-project"
version = "0.1.0"
```

**src/main.smith**:
```smith
import { grover_search } from "./algorithms/grover.smith";
import { helper } from "../libraries/utils.smith";

fn main() -> MeasuredResult {
    let q = qreg(3);
    grover_search(q);
    return measure(q);
}
```

**src/algorithms/grover.smith**:
```smith
pub fn grover_search(q: QReg<3>) -> Void {
    // Grover's algorithm implementation
}
```

**libraries/utils.smith**:
```smith
pub fn helper() -> Int {
    return 42;
}
```

### Example 3: Project with Dependencies

```
my-app/
├── qasmith.toml
└── src/
    └── main.smith

quantum-utils/  (separate repository)
├── qasmith.toml
├── rotations.smith
└── measurements.smith
```

**my-app/qasmith.toml**:
```toml
name = "my-app"
version = "0.1.0"

[dependencies.quantum-utils]
path = "../quantum-utils"
```

**my-app/src/main.smith**:
```smith
import { rzz } from "quantum-utils/rotations";
import { measure_all } from "quantum-utils/measurements";

fn main() -> MeasuredResult {
    let q = qreg(2);
    rzz(0.5, q[0], q[1]);
    return measure_all(q);
}
```

## Further Reading

- [Qasmith Language Guide](language.md)
- [Core Library Reference](core-library.md)
- [Optimization Examples](optimization/index.md)
- [Contributing to Qasmith](../CONTRIBUTING.md)
