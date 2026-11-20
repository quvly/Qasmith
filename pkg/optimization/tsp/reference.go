package tsp

import (
	"context"
	"fmt"
	"slices"
)

// DefaultClassicalLimit bounds permutation search unless explicitly overridden.
const DefaultClassicalLimit = 9

// MaxClassicalLimit bounds explicitly configured permutation searches.
const MaxClassicalLimit = 11

// ClassicalResult describes the lexicographically first shortest closed tour.
type ClassicalResult struct {
	Route        []int
	Distance     float64
	OptimalTours int
	ToursChecked int
}

// SolveClassical enumerates non-start city permutations independently of QUBO.
// A zero limit selects the default.
func SolveClassical(ctx context.Context, p Problem, limit int) (ClassicalResult, error) {
	if limit == 0 {
		limit = DefaultClassicalLimit
	}
	if limit < 3 || limit > MaxClassicalLimit {
		return ClassicalResult{}, fmt.Errorf("classical city limit must be between 3 and %d", MaxClassicalLimit)
	}
	n := p.Cities()
	if n < 3 {
		return ClassicalResult{}, fmt.Errorf("invalid TSP problem")
	}
	if n > limit {
		return ClassicalResult{}, fmt.Errorf("classical permutation limit is %d cities (received %d)", limit, n)
	}
	route := make([]int, n+1)
	route[0], route[n] = p.start, p.start
	pos := 1
	for city := range n {
		if city != p.start {
			route[pos] = city
			pos++
		}
	}
	var result ClassicalResult
	for {
		if err := ctx.Err(); err != nil {
			return ClassicalResult{}, err
		}
		d, err := p.routeDistance(route)
		if err != nil {
			return ClassicalResult{}, err
		}
		if result.ToursChecked == 0 || d < result.Distance {
			result.Route, result.Distance, result.OptimalTours = slices.Clone(route), d, 1
		} else if d == result.Distance {
			result.OptimalTours++
		}
		result.ToursChecked++
		if !nextPermutation(route[1:n]) {
			return result, nil
		}
	}
}

func nextPermutation(v []int) bool {
	i := len(v) - 2
	for i >= 0 && v[i] >= v[i+1] {
		i--
	}
	if i < 0 {
		return false
	}
	j := len(v) - 1
	for v[j] <= v[i] {
		j--
	}
	v[i], v[j] = v[j], v[i]
	slices.Reverse(v[i+1:])
	return true
}
