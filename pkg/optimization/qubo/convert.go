package qubo

import (
	"fmt"
	"math/big"

	"github.com/quvly/Qasmith/pkg/optimization/internal/energy"
	"github.com/quvly/Qasmith/pkg/optimization/ising"
)

// ToIsing substitutes x = (1-z)/2 and preserves the full constant term.
// Contributions are accumulated exactly before rounding each coefficient to Float64.
func (m Model) ToIsing() (ising.Model, error) {
	if err := m.Validate(); err != nil {
		return ising.Model{}, err
	}
	offset := energy.Exact(m.Offset())
	fields := make(map[int]*big.Rat)
	subtractField := func(i int, amount *big.Rat) {
		if fields[i] == nil {
			fields[i] = new(big.Rat)
		}
		fields[i].Sub(fields[i], amount)
	}
	for _, t := range m.data.Linear {
		half := energy.Exact(t.Coefficient)
		half.Quo(half, big.NewRat(2, 1))
		offset.Add(offset, half)
		subtractField(t.Index, half)
	}
	couplings := make([]ising.Coupling, 0, len(m.data.Pairs))
	for _, t := range m.data.Pairs {
		quarter := energy.Exact(t.Coefficient)
		quarter.Quo(quarter, big.NewRat(4, 1))
		j, err := energy.Rounded(quarter)
		if err != nil {
			return ising.Model{}, fmt.Errorf("coupling (%d, %d): %w", t.I, t.J, err)
		}
		couplings = append(couplings, ising.Coupling{I: t.I, J: t.J, Coefficient: j})
		offset.Add(offset, quarter)
		subtractField(t.I, quarter)
		subtractField(t.J, quarter)
	}
	h := make([]ising.Field, 0, len(fields))
	for _, i := range sortedIndices(fields) {
		coefficient, err := energy.Rounded(fields[i])
		if err != nil {
			return ising.Model{}, fmt.Errorf("field %d: %w", i, err)
		}
		h = append(h, ising.Field{Index: i, Coefficient: coefficient})
	}
	c, err := energy.Rounded(offset)
	if err != nil {
		return ising.Model{}, fmt.Errorf("Ising offset: %w", err)
	}
	return ising.New(m.Variables(), c, h, couplings)
}
