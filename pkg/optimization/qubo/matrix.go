package qubo

import "fmt"

// FromMatrix imports xᵀQx + offset. It folds the diagonal into linear terms
// and sums Q[i][j] with Q[j][i] for each distinct pair.
func FromMatrix(matrix [][]float64, offset float64) (Model, error) {
	b, err := New(len(matrix))
	if err != nil {
		return Model{}, err
	}
	if err := b.SetOffset(offset); err != nil {
		return Model{}, err
	}
	for i, row := range matrix {
		if len(row) != len(matrix) {
			return Model{}, fmt.Errorf("matrix row %d has length %d, expected %d", i, len(row), len(matrix))
		}
	}
	for i, row := range matrix {
		if err := b.AddLinear(i, row[i]); err != nil {
			return Model{}, err
		}
		for j := i + 1; j < len(matrix); j++ {
			if err := b.AddQuadratic(i, j, row[j]+matrix[j][i]); err != nil {
				return Model{}, err
			}
		}
	}
	return b.Build()
}
