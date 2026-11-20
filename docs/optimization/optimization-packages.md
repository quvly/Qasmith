# Pure Smith optimization packages

A Smith helper can both return and receive an `OptimizationProblem`. Define your model with the [standard exports](qubo-language.md), then expose it from an ordinary public function.

```typescript
import { qubo, addLinear } from "std/optimization/qubo";

pub fn weightedPair(first: Float, second: Float) -> OptimizationProblem {
    let base = qubo(2, 0.0);
    let a = addLinear(base, 0, first);
    return addLinear(a, 1, second);
}
```

For a relative module, import it with `import { weightedPair } from "./models.smith";`. A second helper can accept the result and apply a constraint. The [imported example](../../examples/optimization/qubo-language/imported.smith) passes a model through two ordinary Smith modules.

To share the function as a local dependency, put the file inside a dependency directory and declare its path in `qasmith.json`. This manifest format maps a package name to a local directory:

```json
{
  "name": "model-demo",
  "version": "0.1.0",
  "dependencies": {
    "myoptimization": {"path": "./packages/myoptimization"}
  }
}
```

Then use `import { buildProblem } from "myoptimization/models";`. The [working packaged example](../../examples/optimization/qubo-language/packaged.smith) includes its manifest and package. No registry changes or custom Go host are required. See the [general package guide](../packages.md).
