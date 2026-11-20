// Package energy implements shared sparse coefficients and numerical helpers.
package energy

import (
	"cmp"
	"fmt"
	"math"
	"math/big"
	"slices"
)

// Linear is a coefficient on a single variable.
type Linear struct {
	Index       int     `json:"index"`
	Coefficient float64 `json:"coefficient"`
}

// Pair is a coefficient on two distinct variables in ascending index order.
type Pair struct {
	I           int     `json:"i"`
	J           int     `json:"j"`
	Coefficient float64 `json:"coefficient"`
}

type Model struct {
	Variables int
	Offset    float64
	Linear    []Linear
	Pairs     []Pair
}

func Finite(x float64) bool { return !math.IsNaN(x) && !math.IsInf(x, 0) }

func Check(value float64) error {
	if !Finite(value) {
		return fmt.Errorf("coefficient must be finite")
	}
	return nil
}

func Index(i, n int) error {
	if i < 0 || i >= n {
		return fmt.Errorf("variable index %d outside [0, %d)", i, n)
	}
	return nil
}

func New(n int, offset float64, linear []Linear, pairs []Pair) (Model, error) {
	linear, pairs = slices.Clone(linear), slices.Clone(pairs)
	slices.SortFunc(linear, func(a, b Linear) int { return cmp.Compare(a.Index, b.Index) })
	slices.SortFunc(pairs, func(a, b Pair) int { return ComparePair(a, b) })
	m := Model{Variables: n, Offset: offset, Linear: linear, Pairs: pairs}
	if err := m.Validate(); err != nil {
		return Model{}, err
	}
	m.Linear = slices.DeleteFunc(linear, func(t Linear) bool { return t.Coefficient == 0 })
	m.Pairs = slices.DeleteFunc(pairs, func(t Pair) bool { return t.Coefficient == 0 })
	return m, nil
}

func ComparePair(a, b Pair) int {
	if n := cmp.Compare(a.I, b.I); n != 0 {
		return n
	}
	return cmp.Compare(a.J, b.J)
}

func (m Model) Validate() error {
	if m.Variables <= 0 {
		return fmt.Errorf("variable count must be positive")
	}
	if err := Check(m.Offset); err != nil {
		return fmt.Errorf("offset: %w", err)
	}
	for i, term := range m.Linear {
		if err := Index(term.Index, m.Variables); err != nil {
			return err
		}
		if i > 0 && m.Linear[i-1].Index >= term.Index {
			return fmt.Errorf("duplicate or unordered linear index %d", term.Index)
		}
		if err := Check(term.Coefficient); err != nil {
			return fmt.Errorf("linear index %d: %w", term.Index, err)
		}
	}
	for i, term := range m.Pairs {
		if err := Index(term.I, m.Variables); err != nil {
			return err
		}
		if err := Index(term.J, m.Variables); err != nil {
			return err
		}
		if term.I >= term.J {
			return fmt.Errorf("expected canonical pair i < j, got (%d, %d)", term.I, term.J)
		}
		if i > 0 && ComparePair(m.Pairs[i-1], term) >= 0 {
			return fmt.Errorf("duplicate or unordered pair (%d, %d)", term.I, term.J)
		}
		if err := Check(term.Coefficient); err != nil {
			return fmt.Errorf("pair (%d, %d): %w", term.I, term.J, err)
		}
	}
	return nil
}

func (m Model) At(i int) (float64, error) {
	if err := Index(i, m.Variables); err != nil {
		return 0, err
	}
	index, found := slices.BinarySearchFunc(m.Linear, i, func(t Linear, i int) int { return cmp.Compare(t.Index, i) })
	if found {
		return m.Linear[index].Coefficient, nil
	}
	return 0, nil
}

func (m Model) Between(i, j int) (float64, error) {
	if err := Index(i, m.Variables); err != nil {
		return 0, err
	}
	if err := Index(j, m.Variables); err != nil {
		return 0, err
	}
	if i == j {
		return 0, fmt.Errorf("pair requires distinct variables")
	}
	if i > j {
		i, j = j, i
	}
	index, found := slices.BinarySearchFunc(m.Pairs, Pair{I: i, J: j}, ComparePair)
	if found {
		return m.Pairs[index].Coefficient, nil
	}
	return 0, nil
}

// Sum uses Neumaier compensation and reports overflow at each step.
type Sum struct{ sum, correction float64 }

func (s *Sum) Add(value float64) error {
	next := s.sum + value
	if !Finite(next) {
		return fmt.Errorf("non-finite intermediate energy")
	}
	if math.Abs(s.sum) >= math.Abs(value) {
		s.correction += (s.sum - next) + value
	} else {
		s.correction += (value - next) + s.sum
	}
	if !Finite(s.correction) {
		return fmt.Errorf("non-finite energy correction")
	}
	s.sum = next
	return nil
}

func (s Sum) Value() (float64, error) {
	n := s.sum + s.correction
	if !Finite(n) {
		return 0, fmt.Errorf("non-finite final energy")
	}
	return n, nil
}

// Exact converts a finite binary64 value to its exact rational representation.
func Exact(value float64) *big.Rat { return new(big.Rat).SetFloat64(value) }

func Rounded(value *big.Rat) (float64, error) {
	v, _ := value.Float64()
	if err := Check(v); err != nil {
		return 0, fmt.Errorf("coefficient overflow: %w", err)
	}
	return v, nil
}
