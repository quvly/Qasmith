# Smith optimization examples

Run from the repository root.

```sh
go build ./cmd/qasmith
./qasmith check tsp.smith
./qasmith compile tsp.smith -o circuit.qasm
./qasmith run tsp.smith --shots 1000 --seed 42
./qasmith optimize tsp.smith --max-evals 200 --restarts 3 --shots 1000 --seed 42 --output-qasm optimized.qasm
```

[program.smith](program.smith) is the direct example. [imported.smith](imported.smith) constructs the same objective through [a reusable Smith helper](library/travel.smith). Both paths emit identical circuits at the same parameters. The repository root also includes [program-imported.smith](../../../program-imported.smith), which imports the root travel library.

[invalid.smith](invalid.smith) demonstrates a negative distance. Its static shapes pass `check`, and `compile` reports the invalid matrix entry during value validation.

The CLI reports both raw measurements and decoded TSP results. A feasible route may be absent in small samples. The [fixed-seed excerpt](expected/seed-42.txt) shows selected lines from the reference run. See the [custom module](custom-module/README.md) for a QUBO factory supplied by an embedding application.
