package qubo

import "github.com/quvly/Qasmith/pkg/optimization/internal/energy"

// JSONSchema is the wire format identifier for QUBO models.
const JSONSchema = "qasmith.qubo.v1"

var format = energy.Format{Schema: JSONSchema, Linear: "linear", Pairs: "quadratic"}

// MarshalJSON encodes a model using the versioned QUBO schema.
func (m Model) MarshalJSON() ([]byte, error) { return format.Encode(m.data) }

// UnmarshalJSON replaces a model after the complete JSON document validates.
func (m *Model) UnmarshalJSON(data []byte) error {
	next, err := format.Decode(data)
	if err != nil {
		return err
	}
	m.data = next
	return nil
}
