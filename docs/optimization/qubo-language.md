# QUBO models in Smith

Import `std/optimization/qubo` to construct immutable `OptimizationProblem` values. The exports are host-backed adapters to the existing Go QUBO package. The Smith functions and packages that call them need only Smith source.

Every export returns an `OptimizationProblem`.

| Export | Parameters | Behavior |
| --- | --- | --- |
| `qubo` | `variables: Int, offset: Float` | Creates an empty model with positive variable count and the supplied offset. |
| `fromMatrix` | `matrix: NumericMatrix, offset: Float` | Imports a fixed-size square matrix as `xᵀQx + offset`. |
| `addLinear` | `model, index: Int, coefficient: Float` | Adds to one linear coefficient. |
| `addQuadratic` | `model, first: Int, second: Int, coefficient: Float` | Adds to a canonical pair. Equal indices add a linear term. |
| `addExactlyOne` | `model, indices: IntArray, penalty: Float` | Adds the squared penalty for selecting exactly one listed variable. |
| `addSquaredLinear` | `model, indices: IntArray, weights: NumericArray, target: Float, penalty: Float` | Adds a weighted squared equality penalty. |
| `scale` | `model, factor: Float` | Multiplies all coefficients and the offset. |
| `combine` | `left, right` | Adds two models of the same size by variable index, including the offsets. |

Here `model`, `left`, and `right` are `OptimizationProblem` values. Array type names in the table describe trusted signature shapes. Write ordinary fixed-size Smith arrays. Matrices and numeric vectors have homogeneous Int or Float leaves. Index vectors must contain Int values. Integer matrix and weight entries must be exactly representable as binary64. Scalar Float parameters follow the ordinary Smith Int-to-Float promotion rule. Non-finite coefficients and results are rejected.

Indices start at zero. For a general matrix the linear coefficient is the diagonal entry, and a canonical off-diagonal coefficient is the sum of both mirrored entries. Use one triangle for a single contribution. A symmetric matrix with both entries populated contributes their sum. See the [Go numerical convention](qubo.md).

`check` verifies signatures, known array shapes, effects, and evaluable Smith expressions without calling host model builders. Invalid model indices, duplicate constraint variables, incompatible model sizes, and decoder protection are checked during compilation.
