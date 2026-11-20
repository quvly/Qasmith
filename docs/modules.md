# Modules and imports

Each `.smith` file defines a module with its own scope. Import a symbol using its exported name, and optionally bind it under a local alias.

```typescript
import { prepareBell as entangle, flip } from "./quantum/gates.smith";
import { add } from "./math.smith";
```

Relative imports include the `.smith` extension and start at the importing file's directory. The compiler resolves symbolic links and checks that a relative import stays inside its project or package root. It loads a shared module once per compilation and reports import cycles with the chain of files.

Export a function by writing `pub fn`. A function defined with `fn` is private. An exported function can call private helpers or imported symbols in its defining module. Callers only see functions they import by name. Local declarations and bindings may not reuse an imported name.

Calls resolve lexical bindings first, then module functions, explicit imports, and compiler built-ins. A lexical value used as a callee is an error. The original gate, allocation, and measurement built-in names are reserved for module declarations and import aliases. `len` and `qreg` resolve as built-ins when there is no earlier binding, so existing functions with those names remain callable. The new control-flow and Boolean keywords are reserved in Smith source. The `pi` constant is also reserved in local bindings and parameters. A module's imports are never implicitly re-exported.

Use `qasmith check program.smith` to validate every function in the resolved graph, and `qasmith deps program.smith` to print source-file dependencies. Shared modules are marked after their first occurrence.

Imported functions can accept fixed-size arrays and borrowed registers using the same signature checks as local functions. See [register helpers and layers](circuit-construction.md#borrow-registers-in-reusable-functions).

Trusted standard modules are resolved from explicit registrations and support import aliases and use from local packages. See the [standard optimization modules](optimization/standard-library.md).
