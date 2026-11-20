// Package qubo represents sparse quadratic objectives over binary variables.
package qubo

import (
	"fmt"
	"slices"

	"github.com/quvly/Qasmith/pkg/optimization/internal/energy"
)

// LinearTerm is a coefficient on a single binary variable.
type LinearTerm = energy.Linear

// QuadraticTerm holds one canonical interaction with I < J.
type QuadraticTerm = energy.Pair

// Model is a finalized sparse QUBO. Copies share read-only coefficient storage.
// UnmarshalJSON replaces the receiver and should not run concurrently with readers.
type Model struct{ data energy.Model }

// Variables returns the number of binary variables.
func (m Model) Variables() int { return m.data.Variables }

// Offset returns the constant term.
func (m Model) Offset() float64 { return m.data.Offset }

// Linear returns the coefficient for index i, or zero for an absent term.
func (m Model) Linear(i int) (float64, error) { return m.data.At(i) }

// Quadratic returns the interaction for two distinct indices in either order.
// Diagonal terms are retrieved with Linear.
func (m Model) Quadratic(i, j int) (float64, error) { return m.data.Between(i, j) }

// LinearTerms returns a copy of the nonzero linear terms in index order.
func (m Model) LinearTerms() []LinearTerm { return slices.Clone(m.data.Linear) }

// QuadraticTerms returns a copy of the nonzero canonical pairs in lexicographic order.
func (m Model) QuadraticTerms() []QuadraticTerm { return slices.Clone(m.data.Pairs) }

// Validate checks the shape, canonical order and coefficient finiteness.
func (m Model) Validate() error { return m.data.Validate() }

// Evaluate computes the energy for a binary assignment in index order.
func (m Model) Evaluate(bits []bool) (float64, error) {
	if m.Variables() <= 0 {
		return 0, fmt.Errorf("invalid QUBO model")
	}
	if len(bits) != m.Variables() {
		return 0, fmt.Errorf("assignment length %d, expected %d", len(bits), m.Variables())
	}
	var sum energy.Sum
	if err := sum.Add(m.Offset()); err != nil {
		return 0, err
	}
	for _, t := range m.data.Linear {
		if bits[t.Index] {
			if err := sum.Add(t.Coefficient); err != nil {
				return 0, err
			}
		}
	}
	for _, t := range m.data.Pairs {
		if bits[t.I] && bits[t.J] {
			if err := sum.Add(t.Coefficient); err != nil {
				return 0, err
			}
		}
	}
	return sum.Value()
}

// TermCounts returns the numbers of stored nonzero linear and quadratic terms.
func (m Model) TermCounts() (linear, quadratic int) { return len(m.data.Linear), len(m.data.Pairs) }
