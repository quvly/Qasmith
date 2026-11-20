package tsp

import (
	"fmt"

	"github.com/quvly/Qasmith/pkg/optimization/internal/energy"
	"github.com/quvly/Qasmith/pkg/optimization/qaoa"
)

// Violation identifies a violated one-hot constraint or malformed assignment.
// Index is a position or original city for the corresponding Kind.
type Violation struct {
	Kind     string
	Index    int
	Count    int
	Expected int
}

// Decoded contains a closed route and original distance only for a feasible assignment.
type Decoded struct {
	Feasible   bool
	Route      []int
	Distance   float64
	Violations []Violation
}

// Decode rejects assignments that fail either one-hot condition.
func (p Problem) Decode(bits []bool) (Decoded, error) {
	e, err := p.Encoding()
	if err != nil {
		return Decoded{}, err
	}
	if len(bits) != e.variables {
		return Decoded{Violations: []Violation{{Kind: "assignment length", Count: len(bits), Expected: e.variables, Index: -1}}}, nil
	}
	m := e.cities - 1
	cityCounts := make([]int, m)
	route := make([]int, e.cities+1)
	route[0], route[len(route)-1] = e.start, e.start
	var result Decoded
	for pos := range m {
		count := 0
		for i := range m {
			if bits[pos*m+i] {
				count++
				cityCounts[i]++
				route[pos+1] = e.city(i)
			}
		}
		if count != 1 {
			result.Violations = append(result.Violations, Violation{Kind: "position occupancy", Index: pos, Count: count, Expected: 1})
		}
	}
	for i, count := range cityCounts {
		if count != 1 {
			result.Violations = append(result.Violations, Violation{Kind: "city occurrences", Index: e.city(i), Count: count, Expected: 1})
		}
	}
	if len(result.Violations) != 0 {
		return result, nil
	}
	distance, err := p.routeDistance(route)
	if err != nil {
		return Decoded{}, err
	}
	return Decoded{Feasible: true, Route: route, Distance: distance}, nil
}

// DecodeBitstring applies Qasmith's measurement order, with variable zero first.
func (p Problem) DecodeBitstring(pattern string) (Decoded, error) {
	bits, err := qaoa.DecodeBitstring(pattern)
	if err != nil {
		return Decoded{}, err
	}
	return p.Decode(bits)
}

// RouteDistance validates a closed Hamiltonian tour and sums its original directed edges.
func (p Problem) RouteDistance(route []int) (float64, error) {
	n := p.Cities()
	if n < 3 || len(route) != n+1 || route[0] != p.start || route[n] != p.start {
		return 0, fmt.Errorf("route must contain %d cities and return to start %d", n, p.start)
	}
	seen := make([]bool, n)
	for _, city := range route[:n] {
		if err := energy.Index(city, n); err != nil {
			return 0, err
		}
		if seen[city] {
			return 0, fmt.Errorf("route visits city %d more than once", city)
		}
		seen[city] = true
	}
	return p.routeDistance(route)
}

func (p Problem) routeDistance(route []int) (float64, error) {
	var sum energy.Sum
	for i := 1; i < len(route); i++ {
		if err := sum.Add(p.distances[route[i-1]][route[i]]); err != nil {
			return 0, fmt.Errorf("tour distance: %w", err)
		}
	}
	return sum.Value()
}
