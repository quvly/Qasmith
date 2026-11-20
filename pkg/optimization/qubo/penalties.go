package qubo

import (
	"fmt"
	"math/big"
	"slices"

	"github.com/quvly/Qasmith/pkg/optimization/internal/energy"
)

// Weight binds a variable to its coefficient in a linear constraint.
type Weight struct {
	Index       int
	Coefficient float64
}

func sortedIndices[V any](m map[int]V) []int {
	indices := make([]int, 0, len(m))
	for index := range m {
		indices = append(indices, index)
	}
	slices.Sort(indices)
	return indices
}

// AddSquaredPenalty adds lambda*(sum(weights[i]*x[i])-target)^2 atomically.
func (b *Builder) AddSquaredPenalty(weights []Weight, target, lambda float64) error {
	if !energy.Finite(lambda) || lambda < 0 {
		return fmt.Errorf("penalty strength must be finite and nonnegative")
	}
	if err := energy.Check(target); err != nil {
		return fmt.Errorf("penalty target: %w", err)
	}
	terms := make(map[int]*big.Rat, len(weights))
	for _, weight := range weights {
		if err := energy.Index(weight.Index, b.n); err != nil {
			return err
		}
		if _, found := terms[weight.Index]; found {
			return fmt.Errorf("duplicate penalty variable %d", weight.Index)
		}
		if err := energy.Check(weight.Coefficient); err != nil {
			return fmt.Errorf("weight at %d: %w", weight.Index, err)
		}
		terms[weight.Index] = energy.Exact(weight.Coefficient)
	}
	if lambda == 0 {
		return nil
	}
	indices := sortedIndices(terms)
	next := b.copy()
	strength, t := energy.Exact(lambda), energy.Exact(target)
	twiceTarget := new(big.Rat).Mul(big.NewRat(2, 1), t)
	twiceStrength := new(big.Rat).Mul(big.NewRat(2, 1), strength)
	update := func(current float64, increment *big.Rat) (float64, error) {
		return energy.Rounded(increment.Add(increment, energy.Exact(current)))
	}
	constant := new(big.Rat).Mul(t, t)
	var err error
	next.offset, err = update(b.offset, constant.Mul(constant, strength))
	if err != nil {
		return fmt.Errorf("penalty offset: %w", err)
	}
	for position, i := range indices {
		w := terms[i]
		if w.Sign() == 0 {
			continue
		}
		line := new(big.Rat).Sub(w, twiceTarget)
		line.Mul(line, w).Mul(line, strength)
		linear, err := update(b.linear[i], line)
		if err != nil {
			return fmt.Errorf("penalty linear term %d: %w", i, err)
		}
		if linear == 0 {
			delete(next.linear, i)
		} else {
			next.linear[i] = linear
		}
		for _, j := range indices[position+1:] {
			if terms[j].Sign() == 0 {
				continue
			}
			pair := [2]int{i, j}
			cross := new(big.Rat).Mul(w, terms[j])
			cross.Mul(cross, twiceStrength)
			coefficient, err := update(b.pairs[pair], cross)
			if err != nil {
				return fmt.Errorf("penalty pair (%d, %d): %w", i, j, err)
			}
			if coefficient == 0 {
				delete(next.pairs, pair)
			} else {
				next.pairs[pair] = coefficient
			}
		}
	}
	*b = next
	return nil
}

// AddExactlyOne penalizes assignments that select other than one index.
func (b *Builder) AddExactlyOne(indices []int, lambda float64) error {
	if len(indices) == 0 {
		return fmt.Errorf("exactly-one requires at least one variable")
	}
	weights := make([]Weight, len(indices))
	for i, index := range indices {
		weights[i] = Weight{Index: index, Coefficient: 1}
	}
	return b.AddSquaredPenalty(weights, 1, lambda)
}
