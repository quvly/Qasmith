# Creating Your First Qasmith Package

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

2. Declare the local directory in `qasmith.json` at the project root.

   ```json
   {
     "name": "qasmith-example",
     "version": "0.1.0",
     "dependencies": {
       "entangle": { "path": "./packages/entangle" }
     }
   }
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

The compiler looks for the nearest `qasmith.json` above the canonical entry file. Without a manifest, the entry file's directory is the project root. A manifest requires a project name and a version in `major.minor.patch` form, optionally with prerelease and build suffixes. Dependency names start with an ASCII letter and may contain ASCII letters, digits, hyphens, and underscores. Dependency paths are relative to the manifest unless absolute and must point to real directories. Roots must be distinct and non-overlapping with other dependency roots. Packages may be inside the project or declared outside it.

A bare `package/module` import uses a dependency name from the entry project's manifest and adds `.smith` when the module path has no extension. Packages can import their own files using relative imports, or refer to another dependency by a name declared in that same manifest. Package-local manifests do not change resolution. The `std/` namespace is reserved. Compilation uses local files and does not fetch packages or run installation scripts.
