# Modules and imports

Each `.smith` file defines a module with its own scope. Import a symbol using its exported name, and optionally bind it under a local alias.

```typescript
import { prepareBell as entangle, flip } from "./quantum/gates.smith";
import { add } from "./math.smith";
import { rzz } from "core/rotations";
```

## Import Types

Qasmith supports three types of imports:

### Core Library

Import from the embedded standard library:

```typescript
import { rzz } from "core/rotations";
import { bell } from "core/gates";
import { abs, max } from "core/math";
```

The core library is bundled with the compiler and always available. See [Core Library](core-library.md) for the full API reference.

### Relative Imports

Relative imports include the `.smith` extension and start at the importing file's directory. The compiler resolves symbolic links and checks that a relative import stays inside its project or package root. It loads a shared module once per compilation and reports import cycles with the chain of files.

### Package Imports

Import from declared dependencies:

```typescript
import { prepareBell } from "entangle/bell";
```

A bare `package/module` import uses a dependency name from the entry project's manifest. Packages can import their own files using relative imports, or refer to another dependency by a name declared in that same manifest.

## Visibility and Scope

Export a function by writing `pub fn`. A function defined with `fn` is private. An exported function can call private helpers or imported symbols in its defining module. Callers only see functions they import by name. Local declarations and bindings may not reuse an imported name.

Calls resolve lexical bindings first, then module functions, explicit imports, and compiler built-ins. A lexical value used as a callee is an error. The original gate, allocation, and measurement built-in names are reserved for module declarations and import aliases. `len` and `qreg` resolve as built-ins when there is no earlier binding, so existing functions with those names remain callable. The new control-flow and Boolean keywords are reserved in Smith source. The `pi` constant is also reserved in local bindings and parameters. A module's imports are never implicitly re-exported.

## CLI Commands

Use `qasmith check program.smith` to validate every function in the resolved graph, and `qasmith deps program.smith` to print source-file dependencies. Shared modules are marked after their first occurrence.

Imported functions can accept fixed-size arrays and borrowed registers using the same signature checks as local functions. See [register helpers and layers](circuit-construction.md#borrow-registers-in-reusable-functions).

Trusted standard modules are resolved from explicit registrations and support import aliases and use from local packages. See the [standard optimization modules](optimization/standard-library.md).

## Creating Your First Qasmith Package

1. Create `packages/entangle/bell.smith` and export a reusable function.

   ```typescript
   fn internalPreparation(q: Qubit) -> Void {
       h(q);
   }

   pub fn prepareBell(a: Qubit, b: Qubit) -> Void {
       internalPreparation(a);
       cx(a, b);
   }
   ```

2. Declare the local directory in `qasmith.toml` at the project root.

   ```toml
   name = "qasmith-example"
   version = "0.1.0"

   [dependencies.entangle]
   path = "./packages/entangle"
   ```

3. Import the exported function in `program.smith`.

   ```typescript
   import { prepareBell } from "entangle/bell";

   fn main() -> MeasuredResult {
       let a = qubit();
       let b = qubit();
       prepareBell(a, b);
       return measure(a, b);
   }
   ```

4. Compile and run. The generated circuit applies a Hadamard to the first qubit and a controlled X to the pair. Simulation produces `00` and `11`.

   ```sh
   qasmith deps program.smith
   qasmith compile program.smith
   qasmith run program.smith --shots 1000 --seed 17
   ```

## Project Manifests

The compiler looks for the nearest `qasmith.toml` (preferred) or `qasmith.json` above the canonical entry file. Without a manifest, the entry file's directory is the project root.

### TOML Format (Recommended)

```toml
name = "my-quantum-project"
version = "0.1.0"
description = "A quantum computing project"

[dependencies]
quantum-utils = { path = "../shared/quantum-utils" }
```

### JSON Format

```json
{
  "name": "my-quantum-project",
  "version": "0.1.0",
  "dependencies": {
    "quantum-utils": { "path": "../shared/quantum-utils" }
  }
}
```

A manifest requires a project name and a version in `major.minor.patch` form, optionally with prerelease and build suffixes. Dependency names start with an ASCII letter and may contain ASCII letters, digits, hyphens, and underscores. Dependency paths are relative to the manifest unless absolute and must point to real directories. Roots must be distinct and non-overlapping with other dependency roots. Packages may be inside the project or declared outside it.

The `std/` and `core/` namespaces are reserved. Compilation uses local files and does not fetch packages or run installation scripts.

## Learn More

For comprehensive details on module resolution, project structure, and package management workflows, see the [Module Resolution Guide](module-resolution.md).
