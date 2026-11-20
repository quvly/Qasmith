package ising

import "github.com/quvly/Qasmith/pkg/optimization/internal/energy"

// JSONSchema is the wire format identifier for Ising models.
const JSONSchema = "qasmith.ising.v1"

var format = energy.Format{Schema: JSONSchema, Linear: "fields", Pairs: "couplings"}

// MarshalJSON encodes a model using the versioned Ising schema.
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
