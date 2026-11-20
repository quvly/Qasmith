# QAOA extension interfaces

The QAOA package emits ordinary `ir.Program` values. Use `Problem.Build` to feed existing tools, or `Problem.Export` to select an `ir.Backend`. Build options accept existing IR passes. The pipeline revalidates the IR, output register layout, measurement order, and gate budget after each pass. The default evaluator uses the unmodified synthesized circuit.

Implement `qaoa.Evaluator` to supply expectations from another execution method. Implement `qaoa.Optimizer` to substitute a classical search. The optimizer receives an `Objective` that returns expected energy. The hybrid helper uses built-in local sampling for its final measurement, even when an alternative evaluator supplies the search objective. Evaluate or build separately to use an external execution pipeline end to end.

`qasmith.Statevector` and `qasmith.SimulateIR` expose the existing local simulator for validated IR. Amplitudes and measured-qubit IDs are newly allocated, with amplitudes taken before terminal measurement. Existing source compilation, host intrinsic, backend, and pass interfaces keep their signatures.

## Proposed Phase 3 architecture

A future library could expose immutable structured optimization inputs, parameter lists, and results through a typed Smith binding layer. That layer should validate the same shapes and limits as Go and keep static circuit construction separate from runtime execution and result handling. A future `std/optimization` module would depend on that value system. Phase 2D uses Go and QUBO JSON entry points.

For alternative mixers or remote execution, route typed circuits through explicit synthesis and execution boundaries. Keep expectations, sample provenance, and optional classical references distinct in result records. Such integration can reuse the current mathematical models and optimizer interfaces.
