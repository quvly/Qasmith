// Package ising represents Hamiltonians over spins in {-1,+1}.
// Binary zero maps to spin +1, and binary one maps to spin -1.
package ising

import (
	"fmt"
	"slices"

	"github.com/quvly/Qasmith/pkg/optimization/internal/energy"
)

// Field holds a local coefficient at a single variable index.
type Field = energy.Linear

// Coupling holds one canonical interaction with I < J.
type Coupling = energy.Pair

// Model is a finalized sparse Ising Hamiltonian. Copies share read-only storage.
// UnmarshalJSON replaces the receiver and should not run concurrently with readers.
type Model struct{ data energy.Model }

// New constructs a model from fields and canonical couplings. The inputs
// may be unordered. Duplicate indices or pairs are rejected.
func New(variables int, offset float64, fields []Field, couplings []Coupling) (Model, error) {
	data, err := energy.New(variables, offset, fields, couplings)
	return Model{data: data}, err
}

// Variables returns the number of spins.
func (m Model) Variables() int { return m.data.Variables }

// Offset returns the constant term.
func (m Model) Offset() float64 { return m.data.Offset }

// Field returns the local coefficient at i, or zero when absent.
func (m Model) Field(i int) (float64, error) { return m.data.At(i) }

// Coupling returns the interaction between two distinct indices in either order.
func (m Model) Coupling(i, j int) (float64, error) { return m.data.Between(i, j) }

// Fields returns a copy of the nonzero fields in index order.
func (m Model) Fields() []Field { return slices.Clone(m.data.Linear) }

// Couplings returns a copy of the nonzero canonical couplings in pair order.
func (m Model) Couplings() []Coupling { return slices.Clone(m.data.Pairs) }

// Validate checks the shape, coefficient order and finiteness.
func (m Model) Validate() error { return m.data.Validate() }

// Spins maps bits to spins using z = 1 - 2*x in the original variable order.
func Spins(bits []bool) []int8 {
	spins := make([]int8, len(bits))
	for i, bit := range bits {
		spins[i] = 1
		if bit {
			spins[i] = -1
		}
	}
	return spins
}

// Evaluate computes the energy for a spin assignment in index order.
func (m Model) Evaluate(spins []int8) (float64, error) {
	if m.Variables() <= 0 {
		return 0, fmt.Errorf("invalid Ising model")
	}
	if len(spins) != m.Variables() {
		return 0, fmt.Errorf("assignment length %d, expected %d", len(spins), m.Variables())
	}
	for i, spin := range spins {
		if spin != -1 && spin != 1 {
			return 0, fmt.Errorf("invalid spin %d at index %d: expected -1 or +1", spin, i)
		}
	}
	var sum energy.Sum
	if err := sum.Add(m.Offset()); err != nil {
		return 0, err
	}
	for _, t := range m.data.Linear {
		if err := sum.Add(t.Coefficient * float64(spins[t.Index])); err != nil {
			return 0, err
		}
	}
	for _, t := range m.data.Pairs {
		if err := sum.Add(t.Coefficient * float64(spins[t.I]*spins[t.J])); err != nil {
			return 0, err
		}
	}
	return sum.Value()
}
