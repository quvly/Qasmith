package stdlib

import (
	"fmt"

	"github.com/quvly/Qasmith/pkg/extension"
	"github.com/quvly/Qasmith/pkg/optimization/plan"
	"github.com/quvly/Qasmith/pkg/optimization/qubo"
)

func quboFunctions() []extension.Function {
	model := extension.Parameter{Type: plan.OptimizationProblem().Name()}
	integer := extension.Parameter{Type: extension.Int}
	scalar := extension.Parameter{Type: extension.Float}
	indices := extension.Parameter{Rank: 1, Type: extension.Int}
	makeFunction := func(name, documentation string, names []string, parameters []extension.Parameter, check func(extension.StaticLimits, []extension.Type) error,
		evaluate func(extension.StaticContext, []extension.Value) (qubo.Model, error)) extension.Function {
		return extension.Function{Name: name, Documentation: documentation, ParameterNames: names, Parameters: parameters, Return: model.Type, CheckTypes: check,
			Evaluate: func(ctx extension.StaticContext, args []extension.Value) (extension.Value, error) {
				m, err := evaluate(ctx, args)
				if err != nil {
					return extension.Value{}, err
				}
				problem, err := plan.FromModel(m)
				if err != nil {
					return extension.Value{}, err
				}
				return plan.OptimizationProblem().Pack(problem)
			}}
	}
	return []extension.Function{
		makeFunction("qubo", "Creates an empty binary objective.", []string{"variables", "offset"}, []extension.Parameter{integer, scalar}, nil, emptyQUBO),
		makeFunction("fromMatrix", "Imports xᵀQx with the given offset.", []string{"matrix", "offset"}, []extension.Parameter{{Rank: 2}, scalar}, checkQUBOMatrix, matrixQUBO),
		makeFunction("addLinear", "Adds a linear coefficient to a new model.", []string{"model", "index", "coefficient"}, []extension.Parameter{model, integer, scalar}, nil, editQUBO(linearTerm)),
		makeFunction("addQuadratic", "Adds an interaction to a new model.", []string{"model", "first", "second", "coefficient"}, []extension.Parameter{model, integer, integer, scalar}, nil, editQUBO(quadraticTerm)),
		makeFunction("addExactlyOne", "Adds a squared exactly-one penalty.", []string{"model", "indices", "penalty"}, []extension.Parameter{model, indices, scalar}, nil, editQUBO(exactlyOne)),
		makeFunction("addSquaredLinear", "Adds a weighted squared equality penalty.", []string{"model", "indices", "weights", "target", "penalty"}, []extension.Parameter{model, indices, {Rank: 1}, scalar, scalar}, checkEqualLengths, editQUBO(squaredLinear)),
		makeFunction("scale", "Scales all coefficients and the offset.", []string{"model", "factor"}, []extension.Parameter{model, scalar}, nil, scaleQUBO),
		makeFunction("combine", "Adds two objectives with the same variable count.", []string{"left", "right"}, []extension.Parameter{model, model}, nil, combineQUBO),
	}
}

func checkQUBOMatrix(l extension.StaticLimits, types []extension.Type) error {
	row, n, _ := types[0].Array()
	_, width, _ := row.Array()
	if n != width {
		return fmt.Errorf("QUBO matrix must be square, got %d by %d", n, width)
	}
	if n > l.MaxModelVariables {
		return fmt.Errorf("optimization variable limit exceeded (maximum %d)", l.MaxModelVariables)
	}
	return nil
}

func checkEqualLengths(_ extension.StaticLimits, types []extension.Type) error {
	_, indices, _ := types[1].Array()
	_, weights, _ := types[2].Array()
	if indices != weights {
		return fmt.Errorf("variable and weight arrays must have equal lengths")
	}
	return nil
}

func emptyQUBO(ctx extension.StaticContext, args []extension.Value) (qubo.Model, error) {
	n := args[0].Integer
	if n <= 0 {
		return qubo.Model{}, fmt.Errorf("QUBO variable count must be positive")
	}
	l := ctx.Limits()
	if n > int64(l.MaxModelVariables) {
		return qubo.Model{}, fmt.Errorf("optimization variable limit exceeded (maximum %d)", l.MaxModelVariables)
	}
	if err := ctx.Spend(1); err != nil {
		return qubo.Model{}, err
	}
	builder, err := qubo.New(int(n))
	if err != nil {
		return qubo.Model{}, err
	}
	if err := builder.SetOffset(args[1].Float); err != nil {
		return qubo.Model{}, err
	}
	return builder.Build()
}

func matrixQUBO(ctx extension.StaticContext, args []extension.Value) (qubo.Model, error) {
	rows := args[0].Array.Elements
	n := len(rows)
	l := ctx.Limits()
	if n == 0 || n > l.MaxMatrixElements/n {
		return qubo.Model{}, fmt.Errorf("matrix element limit exceeded (maximum %d)", l.MaxMatrixElements)
	}
	if err := ctx.Spend(n * n); err != nil {
		return qubo.Model{}, err
	}
	matrix := make([][]float64, n)
	for i, row := range rows {
		matrix[i] = make([]float64, n)
		for j, value := range row.Array.Elements {
			number, err := numeric(value)
			if err != nil {
				return qubo.Model{}, fmt.Errorf("matrix[%d][%d]: %w", i, j, err)
			}
			matrix[i][j] = number
		}
	}
	return qubo.FromMatrix(matrix, args[1].Float)
}

// Only generic models can be edited. The domain owner is responsible for
// defining any future operation that can keep its metadata consistent.
func genericModel(value extension.Value) (qubo.Model, error) {
	model, err := plan.OptimizationProblem().Unpack(value)
	if err != nil {
		return qubo.Model{}, err
	}
	if model.HasDecoder() {
		return qubo.Model{}, fmt.Errorf("QUBO transformation cannot modify a model with a domain decoder")
	}
	return model.QUBO(), nil
}

func chargeModel(ctx extension.StaticContext, m qubo.Model) error {
	linear, pairs := m.TermCounts()
	for _, size := range []int{1, linear, pairs} {
		if err := ctx.Spend(size); err != nil {
			return err
		}
	}
	return nil
}

type quboEdit func(extension.StaticContext, *qubo.Builder, []extension.Value, int) error

func editQUBO(edit quboEdit) func(extension.StaticContext, []extension.Value) (qubo.Model, error) {
	return func(ctx extension.StaticContext, args []extension.Value) (qubo.Model, error) {
		model, err := genericModel(args[0])
		if err != nil {
			return qubo.Model{}, err
		}
		if err := chargeModel(ctx, model); err != nil {
			return qubo.Model{}, err
		}
		builder, err := qubo.New(model.Variables())
		if err != nil {
			return qubo.Model{}, err
		}
		if err := builder.AddModel(model); err != nil {
			return qubo.Model{}, err
		}
		if err := edit(ctx, builder, args[1:], model.Variables()); err != nil {
			return qubo.Model{}, err
		}
		return builder.Build()
	}
}

func index(value int64, n int) (int, error) {
	if value < 0 || value >= int64(n) {
		return 0, fmt.Errorf("variable index %d is outside model range 0..%d", value, n-1)
	}
	return int(value), nil
}

func linearTerm(_ extension.StaticContext, b *qubo.Builder, args []extension.Value, n int) error {
	i, err := index(args[0].Integer, n)
	if err != nil {
		return err
	}
	return b.AddLinear(i, args[1].Float)
}

func quadraticTerm(_ extension.StaticContext, b *qubo.Builder, args []extension.Value, n int) error {
	i, err := index(args[0].Integer, n)
	if err != nil {
		return err
	}
	j, err := index(args[1].Integer, n)
	if err != nil {
		return err
	}
	return b.AddQuadratic(i, j, args[2].Float)
}

func constraintIndices(ctx extension.StaticContext, value extension.Value, n int, penalty float64) ([]int, error) {
	if penalty < 0 {
		return nil, fmt.Errorf("penalty weight must be finite and nonnegative")
	}
	count := len(value.Array.Elements)
	if count > n {
		return nil, fmt.Errorf("constraint has more indices than model variables; indices must be distinct")
	}
	if err := ctx.Spend(count); err != nil {
		return nil, err
	}
	// Charge the potential nonzero pair expansion before calling the existing
	// penalty engine. The division checks keep the size computation in range.
	if penalty != 0 && count > 1 {
		left, right := count, count-1
		if left%2 == 0 {
			left /= 2
		} else {
			right /= 2
		}
		maximum := ctx.Limits().MaxStructuredWork
		if left > maximum/right {
			return nil, fmt.Errorf("structured evaluation work limit exceeded (maximum %d)", maximum)
		}
		if err := ctx.Spend(left * right); err != nil {
			return nil, err
		}
	}
	indices := make([]int, count)
	for i, arg := range value.Array.Elements {
		j, err := index(arg.Integer, n)
		if err != nil {
			return nil, err
		}
		indices[i] = j
	}
	return indices, nil
}

func exactlyOne(ctx extension.StaticContext, b *qubo.Builder, args []extension.Value, n int) error {
	indices, err := constraintIndices(ctx, args[0], n, args[1].Float)
	if err != nil {
		return err
	}
	return b.AddExactlyOne(indices, args[1].Float)
}

func squaredLinear(ctx extension.StaticContext, b *qubo.Builder, args []extension.Value, n int) error {
	indices, err := constraintIndices(ctx, args[0], n, args[3].Float)
	if err != nil {
		return err
	}
	weights := make([]qubo.Weight, len(indices))
	for i, index := range indices {
		value, err := numeric(args[1].Array.Elements[i])
		if err != nil {
			return fmt.Errorf("weight[%d]: %w", i, err)
		}
		weights[i] = qubo.Weight{Index: index, Coefficient: value}
	}
	return b.AddSquaredPenalty(weights, args[2].Float, args[3].Float)
}

func scaleQUBO(ctx extension.StaticContext, args []extension.Value) (qubo.Model, error) {
	model, err := genericModel(args[0])
	if err != nil {
		return qubo.Model{}, err
	}
	if err := chargeModel(ctx, model); err != nil {
		return qubo.Model{}, err
	}
	return model.Scale(args[1].Float)
}

func combineQUBO(ctx extension.StaticContext, args []extension.Value) (qubo.Model, error) {
	left, err := genericModel(args[0])
	if err != nil {
		return qubo.Model{}, err
	}
	right, err := genericModel(args[1])
	if err != nil {
		return qubo.Model{}, err
	}
	if left.Variables() != right.Variables() {
		return qubo.Model{}, fmt.Errorf("cannot combine QUBO models with different variable counts")
	}
	for _, model := range []qubo.Model{left, right} {
		if err := chargeModel(ctx, model); err != nil {
			return qubo.Model{}, err
		}
	}
	b, err := qubo.New(left.Variables())
	if err != nil {
		return qubo.Model{}, err
	}
	for _, model := range []qubo.Model{left, right} {
		if err := b.AddModel(model); err != nil {
			return qubo.Model{}, err
		}
	}
	return b.Build()
}
