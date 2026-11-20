# Language reference

A source module contains `import` and `fn` declarations. Prefix an exported function with `pub`. Parameters and return types are explicit. Functions use lexical scopes with immutable parameters and `let` bindings. Statements end with a semicolon. Comments start with `//` or are enclosed by `/*` and `*/`.

```typescript
fn add(a: Int, b: Int) -> Int {
    let result = a + b;
    return result;
}

fn main() -> Int {
    return add(3, 5);
}
```

`Int` is a signed 64-bit integer. `Float` is an IEEE 754 binary64 value. Write `3` for an `Int` and `3.0`, `0.5`, `1.0e-3`, or `2e2` for a `Float`. The immutable `pi` constant is available in every module and uses Go's `math.Pi`.

Unary minus binds before `*`, `/`, and `%`, which bind before `+` and `-`. Binary operators of equal precedence associate left to right. Parentheses override precedence. Two integer operands produce an integer result, while mixed arithmetic converts the integer operand to `Float`. Passing or returning an `Int` where a `Float` is expected also widens it. Large integers may lose precision during this conversion. Implicit conversion from `Float` to `Int` is an error.

Integer division truncates toward zero. The `%` operator requires integer operands and its result has the sign of the dividend or is zero. Evaluation rejects division or remainder by zero, integer overflow, and non-finite floats. Numeric functions may call other pure numeric functions, including functions imported from another module.

```typescript
fn prepare(a: Qubit, b: Qubit) -> Void {
    h(a);
    cx(a, b);
}

fn main() -> MeasuredResult {
    let first = qubit();
    let second = qubit();
    prepare(first, second);
    return measure(first, second);
}
```

`Qubit` is an owned resource allocated by `let name = qubit();` in a quantum entry point. A helper borrows caller qubits through its parameters. It can apply `x`, `h`, `cx`, `rx`, `ry`, and `rz` or call other helpers. The entry point can allocate individual qubits or registers. The compiler checks that the control and target of every `cx` refer to distinct qubits, including when the operands are passed through other functions. Local bindings cannot copy qubit references.

Rotation gates accept an angle in radians followed by a borrowed qubit. The compiler evaluates the angle while lowering. Angles can use local values, arithmetic, `pi`, and calls to pure functions.

```typescript
pub fn rotate(theta: Float, q: Qubit) -> Void {
    ry(theta / 2.0, q);
}
```

The [example rotation module](../examples/modules/rotations.smith) exports `rzz(theta, a, b)`. It applies `cx(a, b)`, `rz(theta, b)`, and `cx(a, b)`. The two qubit operands must be distinct, including when the function is called through imported wrappers.

`MeasuredResult` is the return type for a quantum entry point. Its final statement is `return measure(q, ...);` with one or more distinct qubit arguments. Argument order determines measurement bit order. The entry point can allocate qubits that do not appear in the returned measurement. Helpers cannot allocate or measure qubits.

`Void` functions can end implicitly or with a bare `return;`. Only `Void` calls can be used as statements. An entry file must declare a parameterless `main` returning `Int` or `MeasuredResult`. Direct and indirect recursion are rejected.

## Registers and fixed-size arrays

`qreg(size)` allocates a register in a `let` binding in the quantum entry point. The size is a positive compile-time `Int` and may be computed by a pure function. `QReg<N>` in a function signature borrows exactly `N` qubits from the caller. The caller owns the register. Register bindings and individual qubit references cannot be copied into another `let` binding.

`q[i]` is a reference to the original qubit. Use it directly as a gate, helper, or measurement argument. Indexes are zero-based compile-time integers. `measure(q)` measures the whole register in increasing index order. To specify individual qubits or change the order, pass distinct qubit arguments. A single measurement expression accepts either one register or individual qubits.

Classical arrays are immutable values with a fixed shape and a single element type. Nested arrays must be rectangular. An empty array literal has no inferable type. Each array element has the same type, so use `[1.0, 2.0]` for `Float[2]`. Index nested arrays from the outside in. The **rightmost** dimension is outermost. For example, `Int[2][3]` is three rows of two integers, so `values[1][0]` selects the second row's first element. Arrays can be passed, returned, and bound to another local name. `len()` returns the outer length of an array or the size of a register.

## Compile-time control

`Bool` values are `true` and `false`. Use `==` and `!=` on two Boolean or two numeric operands. Ordering comparisons `<`, `<=`, `>`, and `>=` accept numeric operands. Mixed numeric comparisons use the same Int-to-Float widening as arithmetic.

From tightest to loosest, precedence is indexing, unary `-` and `!`, multiplication and division operators, addition and subtraction, ordering comparisons, equality comparisons, `&&`, and `||`. The Boolean operators short-circuit. Both operands still undergo type checking.

A `for i in start..end` loop uses increasing integers from `start` inclusive to `end` exclusive. Both bounds must be compile-time `Int` values. Equal bounds give zero iterations. The loop binding is immutable and local to the loop body. Loops can nest and can use outer loop bindings.

An `if` condition must be a compile-time `Bool`. An optional `else` supplies the other branch. Both branches are type-checked. Only the selected branch is evaluated and emits circuit operations. Aliasing checks on indexed qubits use their actual logical IDs after selection and expansion. Measurement remains a terminal return at the top level of the quantum entry function.

See the [circuit construction guide](circuit-construction.md) for complete examples.


## Opaque optimization values

The compiler-distributed optimization modules expose `OptimizationProblem`, an immutable typed compile-time value. A local Smith function can return it or accept it as a parameter. A terminal circuit-producing call is allowed only as the final return from the quantum entry point, with no separate quantum effects. See the [language integration guide](optimization/language-integration.md).

Use the [QUBO standard library](optimization/qubo-language.md) to build immutable optimization objectives in pure Smith.
