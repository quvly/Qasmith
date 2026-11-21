# Creating Your First Qasmith Package

This guide walks you through creating a reusable Qasmith package that can be imported by other projects.

## Option 1: Quick Start with Project Initialization

The easiest way to start is with `qasmith init`:

```bash
# Create a new project
qasmith init my-quantum-project
cd my-quantum-project
```

This creates a complete project structure:

```
my-quantum-project/
├── qasmith.toml
├── README.md
├── .gitignore
└── src/
    └── main.smith
```

## Option 2: Manual Package Creation

If you prefer to create a package from scratch:

### 1. Create the package structure

Create `packages/entangle/bell.smith` and export a reusable function:

```smith
fn internalPreparation(q: Qubit) -> Void {
    h(q);
}

pub fn prepareBell(a: Qubit, b: Qubit) -> Void {
    internalPreparation(a);
    cx(a, b);
}
```

### 2. Declare the package in your manifest

Create `qasmith.toml` at the project root (recommended) or `qasmith.json`:

**TOML format (recommended):**
```toml
name = "qasmith-example"
version = "0.1.0"

[dependencies]
entangle = { path = "./packages/entangle" }
```

**JSON format:**
```json
{
  "name": "qasmith-example",
  "version": "0.1.0",
  "dependencies": {
    "entangle": { "path": "./packages/entangle" }
  }
}
```

### 3. Import and use the package

In your `main.smith`:

```smith
import { prepareBell } from "entangle/bell";

fn main() -> MeasuredResult {
    let a = qubit();
    let b = qubit();
    prepareBell(a, b);
    return measure(a, b);
}
```

### 4. Compile and run

```bash
qasmith deps main.smith
qasmith compile main.smith
qasmith run main.smith --shots 1000 --seed 17
```

The generated circuit applies a Hadamard to the first qubit and a controlled X to the pair. Simulation produces `00` and `11`.

## Using the Core Library

Qasmith includes a built-in core library with common quantum operations. You can use it without any setup:

```smith
import { rzz } from "core/rotations";
import { bell } from "core/gates";
import { abs, max } from "core/math";

fn main() -> MeasuredResult {
    let q = qreg(2);
    
    // Use core library functions
    bell(q[0], q[1]);
    rzz(0.5, q[0], q[1]);
    
    return measure(q);
}
```

See the [Core Library Reference](core-library.md) for all available modules.

## Package Manifest Rules

The compiler looks for the nearest `qasmith.toml` (preferred) or `qasmith.json` above the canonical entry file. Without a manifest, the entry file's directory is the project root.

A manifest requires:
- **Project name**: Must start with an ASCII letter and contain only ASCII letters, digits, hyphens, and underscores
- **Version**: In `major.minor.patch` form, optionally with prerelease and build suffixes

### Dependency Rules

- Dependency paths are relative to the manifest unless absolute
- Paths must point to real directories
- Dependency roots must be distinct and non-overlapping
- Packages may be inside the project or declared outside it
- The `core/` and `std/` namespaces are reserved

### Import Resolution

- A bare `package/module` import uses a dependency name from the entry project's manifest
- The `.smith` extension is added automatically when the module path has no extension
- Packages can import their own files using relative imports
- Packages can refer to other dependencies declared in the same manifest
- Package-local manifests do not change resolution

## Best Practices

### Project Structure

```
my-project/
├── qasmith.toml              # Project manifest
├── src/
│   ├── main.smith            # Entry point
│   └── utils/
│       └── helpers.smith     # Project modules
├── libraries/
│   └── rotations.smith       # Local libraries
└── packages/
    └── my-package/           # External dependencies
        ├── qasmith.toml
        └── module.smith
```

### When to Use Each Import Type

- **Core library** (`core/*`): Use for standard quantum operations that are universally needed
- **Relative imports** (`./...`): Use for project-specific modules
- **Package imports** (`package/*`): Use for reusable libraries shared across projects

### Version Control

Commit your manifest file to version control, but consider whether to commit the `packages/` directory:

- **Commit packages/** if they are part of your project
- **Exclude packages/** if they are external dependencies that users should fetch separately

## Next Steps

- [Core Library Reference](core-library.md) - Explore all core library modules
- [Modules](modules.md) - Learn more about module organization
- [Language Reference](language.md) - Full language syntax
