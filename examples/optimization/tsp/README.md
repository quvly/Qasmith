# Four-city TSP example

[problem.json](problem.json) is a complete symmetric distance matrix with city 0 as the start. The public Go example loads it, prints the variable mapping, and checks the exhaustive QUBO minimum against independent tour enumeration. It exports the QUBO and two measured QAOA circuits.

```sh
go run ./examples/optimization/tsp
```

The generated files go under `examples/optimization/tsp/generated`. Use `-out` to select another directory. [reference-output.txt](reference-output.txt) records one run with depth 1, initial gamma 0.4 and beta 0.3, seed 42, 1,000 shots per measurement, and an exact objective search with 200 evaluations and three additional starts.

## Run the workflow with the CLI

```sh
go build ./cmd/qasmith
problem=examples/optimization/tsp/problem.json
./qasmith tsp inspect "$problem"
./qasmith tsp solve-classical "$problem"
./qasmith tsp qubo "$problem" -o examples/optimization/tsp/generated/tsp.qubo.json
./qasmith tsp compile "$problem" --layers 1 --gammas 0.4 --betas 0.3 -o examples/optimization/tsp/generated/initial.qasm
./qasmith tsp run "$problem" --layers 1 --gammas 0.4 --betas 0.3 --shots 1000 --seed 42 --exact
./qasmith tsp optimize "$problem" --layers 1 --max-evals 200 --restarts 3 --shots 1000 --seed 42 -o examples/optimization/tsp/generated/optimized.qasm
```

The QUBO export works as input to the generic `qasmith qaoa` commands. Its variables are position-major. For example, `100001010` selects cities 1, 3, and 2 in order and closes the tour at city 0.

Feasibility counts all valid tours. Optimal sample rate counts only valid samples at the independently computed minimum. Both rates use all shots as the denominator. Expected QUBO energy includes invalid-state penalties, so it differs from the travel distance of a valid candidate. The X mixer can yield infeasible bitstrings, which the decoder reports without repair.

See the [TSP model and API guide](../../../docs/optimization/tsp.md).
