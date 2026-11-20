package qubo

import (
	"fmt"
	"maps"

	"github.com/quvly/Qasmith/pkg/optimization/internal/energy"
)

// Builder accumulates coefficients before finalization. Use New to initialize one.
// Use it through its pointer. To share coefficients, build an immutable Model.
type Builder struct {
	n      int
	offset float64
	linear map[int]float64
	pairs  map[[2]int]float64
}

// New creates an empty builder for a positive number of binary variables.
func New(variables int) (*Builder, error) {
	if variables <= 0 {
		return nil, fmt.Errorf("variable count must be positive")
	}
	return &Builder{n: variables, linear: make(map[int]float64), pairs: make(map[[2]int]float64)}, nil
}

// Build returns an independent immutable snapshot of the current coefficients.
func (b *Builder) Build() (Model, error) {
	linear := make([]LinearTerm, 0, len(b.linear))
	for i, c := range b.linear {
		linear = append(linear, LinearTerm{Index: i, Coefficient: c})
	}
	pairs := make([]QuadraticTerm, 0, len(b.pairs))
	for p, c := range b.pairs {
		pairs = append(pairs, QuadraticTerm{I: p[0], J: p[1], Coefficient: c})
	}
	m, err := energy.New(b.n, b.offset, linear, pairs)
	return Model{data: m}, err
}

// AddLinear adds a coefficient to an existing linear term.
func (b *Builder) AddLinear(i int, coefficient float64) error {
	if err := energy.Index(i, b.n); err != nil {
		return err
	}
	return accumulate(b.linear, i, coefficient)
}

// AddQuadratic adds an interaction, normalizing reversed pairs and folding diagonals.
func (b *Builder) AddQuadratic(i, j int, coefficient float64) error {
	if i == j {
		return b.AddLinear(i, coefficient)
	}
	if err := energy.Index(i, b.n); err != nil {
		return err
	}
	if err := energy.Index(j, b.n); err != nil {
		return err
	}
	if i > j {
		i, j = j, i
	}
	return accumulate(b.pairs, [2]int{i, j}, coefficient)
}

// SetOffset replaces the constant term.
func (b *Builder) SetOffset(offset float64) error {
	if err := energy.Check(offset); err != nil {
		return fmt.Errorf("offset: %w", err)
	}
	b.offset = offset
	return nil
}

// AddOffset adds to the constant term.
func (b *Builder) AddOffset(offset float64) error { return b.SetOffset(b.offset + offset) }

func accumulate[K comparable](terms map[K]float64, key K, coefficient float64) error {
	next := terms[key] + coefficient
	if err := energy.Check(next); err != nil {
		return fmt.Errorf("term %v: %w", key, err)
	}
	if next == 0 {
		delete(terms, key)
	} else {
		terms[key] = next
	}
	return nil
}

func (b *Builder) copy() Builder {
	return Builder{n: b.n, offset: b.offset, linear: maps.Clone(b.linear), pairs: maps.Clone(b.pairs)}
}

// AddModel adds a compatible model atomically, including its offset.
func (b *Builder) AddModel(model Model) error {
	if err := model.Validate(); err != nil {
		return err
	}
	if b.n != model.Variables() {
		return fmt.Errorf("variable counts differ: %d and %d", b.n, model.Variables())
	}
	next := b.copy()
	if err := next.AddOffset(model.Offset()); err != nil {
		return err
	}
	for _, t := range model.data.Linear {
		if err := next.AddLinear(t.Index, t.Coefficient); err != nil {
			return err
		}
	}
	for _, t := range model.data.Pairs {
		if err := next.AddQuadratic(t.I, t.J, t.Coefficient); err != nil {
			return err
		}
	}
	*b = next
	return nil
}

// Scale returns a new model with every coefficient and the offset multiplied by factor.
func (m Model) Scale(factor float64) (Model, error) {
	if err := m.Validate(); err != nil {
		return Model{}, err
	}
	if err := energy.Check(factor); err != nil {
		return Model{}, fmt.Errorf("scale: %w", err)
	}
	linear, pairs := m.LinearTerms(), m.QuadraticTerms()
	for i := range linear {
		linear[i].Coefficient *= factor
	}
	for i := range pairs {
		pairs[i].Coefficient *= factor
	}
	data, err := energy.New(m.Variables(), m.Offset()*factor, linear, pairs)
	return Model{data: data}, err
}
