# Model immutability and resource budgets

Every QUBO transformation returns a validated model snapshot. An earlier binding can be passed to multiple helpers or retained through imports. No fields are exposed for Smith mutation, and opaque values have no scalar conversion.

The generic transforms reject inputs with attached domain decoders. A QUBO value remains generic by construction unless its Go integration attaches a decoder. TSP keeps its established route metadata and continues to work with the QAOA producer.

Model construction charges work before copying terms, allocating matrix conversions, or expanding constraints. Penalties charge their potential pair expansion at positive strength. Repeated transformations therefore share a cumulative work limit even when intermediate bindings are later unused. The model coefficient limit includes the offset. The compiler applies its existing array, variable, depth, step, gate, and layer limits. See [execution plan limits](optimization-plans.md) for defaults and flags.

Generic intermediate models hold their original QUBO. The final QAOA recipe converts the finished model to Ising. This allows a finite interim QUBO to be scaled before Ising conversion is required. Circuit export and the local simulation width limit remain independent.
