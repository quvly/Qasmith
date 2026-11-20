package tsp

import (
	"fmt"
	"math"

	"github.com/quvly/Qasmith/pkg/optimization/internal/energy"
	"github.com/quvly/Qasmith/pkg/optimization/qubo"
)

// Encoding maps non-start cities in ascending city order, within each position.
type Encoding struct {
	cities, start, variables int
}

// Encoding constructs the fixed-start positional mapping for this problem.
func (p Problem) Encoding() (Encoding, error) {
	m := p.Cities() - 1
	if m < 2 || m > int(^uint(0)>>1)/m {
		return Encoding{}, fmt.Errorf("invalid TSP encoding size")
	}
	return Encoding{cities: p.Cities(), start: p.Start(), variables: m * m}, nil
}

func (e Encoding) Variables() int { return e.variables }

// Index maps a non-start city and a zero-based position to a variable index.
func (e Encoding) Index(city, position int) (int, error) {
	if city < 0 || city >= e.cities || city == e.start {
		return 0, fmt.Errorf("city %d is not an encoded non-start city", city)
	}
	m := e.cities - 1
	if position < 0 || position >= m {
		return 0, fmt.Errorf("position %d is outside [0, %d)", position, m)
	}
	if city > e.start {
		city--
	}
	return position*m + city, nil
}

// Variable returns the city and zero-based position for a variable index.
func (e Encoding) Variable(index int) (city, position int, err error) {
	if err := energy.Index(index, e.variables); err != nil {
		return 0, 0, err
	}
	m := e.cities - 1
	return e.city(index % m), index / m, nil
}

func (e Encoding) city(ordinal int) int {
	if ordinal >= e.start {
		return ordinal + 1
	}
	return ordinal
}

// UniformFeasibility is (n-1)! / 2^((n-1)^2) for the initial uniform distribution.
func (e Encoding) UniformFeasibility() float64 {
	if e.variables == 0 {
		return 0
	}
	m := e.cities - 1
	p := 1.0
	for i := 1; i <= m; i++ {
		p *= math.Ldexp(float64(i), -m)
	}
	return p
}

// DefaultPenalty returns a conservative, representable weight strictly above
// the upper bound on a complete tour. An explicit penalty can be used if this fails.
func (p Problem) DefaultPenalty() (float64, error) {
	if p.Cities() < 3 {
		return 0, fmt.Errorf("invalid TSP problem")
	}
	bound := float64(p.Cities()) * p.maximum
	penalty := max(1, 2*bound)
	if !energy.Finite(penalty) || penalty <= bound || !energy.Finite(2*float64(p.Cities()-1)*penalty) {
		return 0, fmt.Errorf("automatic penalty exceeds Float64 range; rescale distances or specify a positive penalty explicitly")
	}
	return penalty, nil
}

// DefaultMaxVariables is the default encoding budget.
const DefaultMaxVariables = 4096

// EncodeOptions configures the QUBO encoding. Nil Penalty selects the automatic
// weight and zero MaxVariables selects DefaultMaxVariables.
type EncodeOptions struct {
	Penalty      *float64
	MaxVariables int
}

// ToQUBO constructs the directed travel objective and applies generic exactly-one
// penalties to every position and non-start city.
func ToQUBO(p Problem, opts EncodeOptions) (qubo.Model, Encoding, error) {
	e, err := p.Encoding()
	if err != nil {
		return qubo.Model{}, Encoding{}, err
	}
	limit := opts.MaxVariables
	if limit == 0 {
		limit = DefaultMaxVariables
	}
	if limit < 1 || e.variables > limit {
		return qubo.Model{}, Encoding{}, fmt.Errorf("TSP encoding needs %d variables, limit is %d", e.variables, limit)
	}
	penalty, err := resolvePenalty(p, opts.Penalty)
	if err != nil {
		return qubo.Model{}, Encoding{}, err
	}
	model, err := buildQUBO(p, e, penalty)
	if err != nil {
		return qubo.Model{}, Encoding{}, err
	}
	return model, e, nil
}

func resolvePenalty(p Problem, custom *float64) (float64, error) {
	if custom == nil {
		return p.DefaultPenalty()
	}
	if !energy.Finite(*custom) || *custom <= 0 {
		return 0, fmt.Errorf("penalty must be positive and finite")
	}
	return *custom, nil
}

func buildQUBO(p Problem, e Encoding, penalty float64) (qubo.Model, error) {
	b, err := qubo.New(e.variables)
	if err != nil {
		return qubo.Model{}, err
	}
	m := e.cities - 1
	ids := make([]int, m)
	for position := range m {
		for i := range m {
			ids[i] = position*m + i
		}
		if err := b.AddExactlyOne(ids, penalty); err != nil {
			return qubo.Model{}, fmt.Errorf("position %d constraint: %w", position, err)
		}
	}
	for i := range m {
		for position := range m {
			ids[position] = position*m + i
		}
		if err := b.AddExactlyOne(ids, penalty); err != nil {
			return qubo.Model{}, fmt.Errorf("city %d constraint: %w", e.city(i), err)
		}
		if err := b.AddLinear(i, p.distances[e.start][e.city(i)]); err != nil {
			return qubo.Model{}, err
		}
		if err := b.AddLinear((m-1)*m+i, p.distances[e.city(i)][e.start]); err != nil {
			return qubo.Model{}, err
		}
	}
	for position := 0; position < m-1; position++ {
		for i := range m {
			for j := range m {
				// Self transitions have zero distance by model validation.
				if i != j {
					if err := b.AddQuadratic(position*m+i, (position+1)*m+j, p.distances[e.city(i)][e.city(j)]); err != nil {
						return qubo.Model{}, err
					}
				}
			}
		}
	}
	return b.Build()
}
