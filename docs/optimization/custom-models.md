# Building Your First Quantum Optimization Model in Qasmith

Create a file called `my-objective.smith` and write a two-variable binary objective. No custom Go extension is needed.

```typescript
import { qubo, addLinear, addQuadratic, addExactlyOne } from "std/optimization/qubo";
import { qaoa } from "std/optimization/qaoa";

fn main() -> MeasuredResult {
    let base = qubo(2, 0.5);
    let linear = addLinear(base, 0, -2.0);
    let interaction = addQuadratic(linear, 0, 1, 0.75);
    let model = addExactlyOne(interaction, [0, 1], 4.0);
    return qaoa(model, [0.4], [0.3]);
}
```

The unconstrained energy is `0.5 − 2x₀ + 0.75x₀x₁`. The penalty adds cost when neither or both variables are selected. Each binding keeps its own immutable model.

```sh
qasmith check my-objective.smith
qasmith compile my-objective.smith -o initial.qasm
qasmith run my-objective.smith --shots 1000 --seed 42
qasmith optimize my-objective.smith --max-evals 200 --restarts 3 --shots 1000 --seed 42 --output-qasm optimized.qasm
```

The simulator returns counts and energy for each observed bitstring. The first bit is variable zero. Expected energy averages across observations. The best observed assignment is a separate result. The optimizer searches for lower expected energy, starts at the source parameters, and leaves the source file unchanged.

To reuse the model, move the declarations that construct it into a public function returning `OptimizationProblem`. Import that function from the entry program and keep the final `qaoa` call there. Follow the [pure Smith library and package guide](optimization-packages.md) to distribute it locally.
