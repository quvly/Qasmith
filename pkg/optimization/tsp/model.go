// Package tsp models complete directed or symmetric travelling salesman instances.
package tsp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"slices"

	"github.com/quvly/Qasmith/pkg/optimization/internal/energy"
)

// JSONSchema identifies the versioned TSP input format.
const JSONSchema = "qasmith.tsp.v1"

// Problem is a validated complete distance matrix and fixed starting city.
// Copies share read-only distances. JSON decoding replaces the receiving value.
type Problem struct {
	distances [][]float64
	start     int
	name      string
	maximum   float64
	symmetric bool
}

// New validates and copies the original directed distances.
func New(distances [][]float64, start int) (Problem, error) {
	n := len(distances)
	if n < 3 {
		return Problem{}, fmt.Errorf("distance matrix must have at least three cities")
	}
	if start < 0 || start >= n {
		return Problem{}, fmt.Errorf("starting city %d is outside [0, %d)", start, n)
	}
	for i, row := range distances {
		if len(row) != n {
			return Problem{}, fmt.Errorf("distances[%d] has %d entries, expected %d", i, len(row), n)
		}
	}
	p := Problem{start: start, distances: make([][]float64, n), symmetric: true}
	for i, row := range distances {
		for j, d := range row {
			if !energy.Finite(d) || d < 0 {
				return Problem{}, fmt.Errorf("distances[%d][%d] must be finite and nonnegative", i, j)
			}
			if i == j && d != 0 {
				return Problem{}, fmt.Errorf("distances[%d][%d] must be zero", i, j)
			}
			p.maximum = max(p.maximum, d)
			p.symmetric = p.symmetric && d == distances[j][i]
		}
		p.distances[i] = slices.Clone(row)
	}
	return p, nil
}

func (p Problem) Cities() int          { return len(p.distances) }
func (p Problem) Start() int           { return p.start }
func (p Problem) Name() string         { return p.name }
func (p Problem) Symmetric() bool      { return p.symmetric }
func (p Problem) MaxDistance() float64 { return p.maximum }

// Distance returns the original distance from city i to city j.
func (p Problem) Distance(i, j int) (float64, error) {
	if err := energy.Index(i, p.Cities()); err != nil {
		return 0, err
	}
	if err := energy.Index(j, p.Cities()); err != nil {
		return 0, err
	}
	return p.distances[i][j], nil
}

// UnmarshalJSON validates the schema and matrix before replacing the receiver.
func (p *Problem) UnmarshalJSON(data []byte) error {
	d := json.NewDecoder(bytes.NewReader(data))
	open, err := d.Token()
	if err != nil {
		return err
	}
	if open != json.Delim('{') {
		return fmt.Errorf("expected TSP JSON object")
	}
	var schema, name string
	var start int
	var matrix [][]*float64
	fields := map[string]any{"schema": &schema, "name": &name, "start": &start, "distances": &matrix}
	seen := make(map[string]bool)
	for d.More() {
		token, err := d.Token()
		if err != nil {
			return err
		}
		key := token.(string)
		field, exists := fields[key]
		if !exists || seen[key] {
			return fmt.Errorf("unknown or duplicate TSP field %q", key)
		}
		var raw json.RawMessage
		if err := d.Decode(&raw); err != nil {
			return err
		}
		if bytes.Equal(raw, []byte("null")) {
			return fmt.Errorf("%s cannot be null", key)
		}
		if err := json.Unmarshal(raw, field); err != nil {
			return fmt.Errorf("%s: %w", key, err)
		}
		seen[key] = true
	}
	if _, err := d.Token(); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return fmt.Errorf("unexpected data after TSP object")
	}
	for _, key := range []string{"schema", "start", "distances"} {
		if !seen[key] {
			return fmt.Errorf("missing TSP field %q", key)
		}
	}
	if schema != JSONSchema {
		return fmt.Errorf("unsupported TSP schema %q", schema)
	}
	distances := make([][]float64, len(matrix))
	for i, row := range matrix {
		distances[i] = make([]float64, len(row))
		for j, d := range row {
			if d == nil {
				return fmt.Errorf("distances[%d][%d] cannot be null", i, j)
			}
			distances[i][j] = *d
		}
	}
	next, err := New(distances, start)
	if err != nil {
		return err
	}
	next.name = name
	*p = next
	return nil
}
