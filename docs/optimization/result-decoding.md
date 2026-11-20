# Optimization result decoding

Each observation retains its bitstring, count, probability estimate, and original QUBO energy. The first bit is variable zero. The objective estimate weights original QUBO energies by their measured frequencies. An optional decoder gets its own copy of the counts and best-observed assignment and can return a typed domain report through `plan.DomainResult`. Decoding errors preserve the raw result and report the source of the plan.

The TSP decoder reuses the Phase 3A route report. Valid assignments become closed tours with distances from the original matrix. Invalid assignments retain constraint violations. The CLI prints all raw counts and QUBO energies, then supplements them with feasible and infeasible sample counts, feasibility rate, and best valid route when present. Go callers can inspect each decoded observation by asserting the domain result to `tsp.ReportMetrics`.

The best observed QUBO assignment, best feasible sampled route, and independent classical optimum are separate. With no valid sampled tour, the CLI reports “none observed” and the Go report has a nil best feasible route. Optimal tour sample rate uses all shots as its denominator and is available when the bounded classical solver completes. The TSP standard decoder enables that solver within its default city limit.
