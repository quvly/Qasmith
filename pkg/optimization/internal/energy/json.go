package energy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// Format specifies the serialized field names of one optimization model.
type Format struct {
	Schema, Linear, Pairs string
}

func (f Format) Encode(m Model) ([]byte, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	linear := make([]float64, m.Variables)
	for _, term := range m.Linear {
		linear[term.Index] = term.Coefficient
	}
	pairs := m.Pairs
	if pairs == nil {
		pairs = []Pair{}
	}
	return json.Marshal(map[string]any{
		"schema": f.Schema, "variables": m.Variables, "offset": m.Offset,
		f.Linear: linear, f.Pairs: pairs,
	})
}

func (f Format) Decode(source []byte) (Model, error) {
	var fields map[string]json.RawMessage
	decoder := json.NewDecoder(bytes.NewReader(source))
	start, err := decoder.Token()
	if err != nil {
		return Model{}, err
	}
	if start != json.Delim('{') {
		return Model{}, fmt.Errorf("expected model object")
	}
	fields = make(map[string]json.RawMessage)
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return Model{}, err
		}
		name := token.(string)
		switch name {
		case "schema", "variables", "offset", f.Linear, f.Pairs:
		default:
			return Model{}, fmt.Errorf("unknown model field %q", name)
		}
		if _, exists := fields[name]; exists {
			return Model{}, fmt.Errorf("duplicate model field %q", name)
		}
		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			return Model{}, err
		}
		if bytes.Equal(raw, []byte("null")) {
			return Model{}, fmt.Errorf("model field %q cannot be null", name)
		}
		fields[name] = raw
	}
	if _, err := decoder.Token(); err != nil {
		return Model{}, err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return Model{}, fmt.Errorf("unexpected data after model")
	}
	var schema string
	var n int
	var offset float64
	var coefficients []*float64
	var interactions []json.RawMessage
	for _, field := range []struct {
		name  string
		value any
	}{
		{"schema", &schema}, {"variables", &n}, {"offset", &offset}, {f.Linear, &coefficients}, {f.Pairs, &interactions},
	} {
		raw, ok := fields[field.name]
		if !ok {
			return Model{}, fmt.Errorf("missing model field %q", field.name)
		}
		if err := json.Unmarshal(raw, field.value); err != nil {
			return Model{}, fmt.Errorf("%s: %w", field.name, err)
		}
	}
	if schema != f.Schema {
		return Model{}, fmt.Errorf("unsupported schema %q, expected %q", schema, f.Schema)
	}
	if len(coefficients) != n {
		return Model{}, fmt.Errorf("%s length %d, expected %d", f.Linear, len(coefficients), n)
	}
	var linear []Linear
	for i, coefficient := range coefficients {
		if coefficient == nil {
			return Model{}, fmt.Errorf("%s[%d] cannot be null", f.Linear, i)
		}
		if *coefficient != 0 {
			linear = append(linear, Linear{Index: i, Coefficient: *coefficient})
		}
	}
	pairs := make([]Pair, 0, len(interactions))
	for index, raw := range interactions {
		pair, err := decodePair(raw)
		if err != nil {
			return Model{}, fmt.Errorf("%s[%d]: %w", f.Pairs, index, err)
		}
		pairs = append(pairs, pair)
	}
	return New(n, offset, linear, pairs)
}

func decodePair(raw json.RawMessage) (Pair, error) {
	d := json.NewDecoder(bytes.NewReader(raw))
	start, err := d.Token()
	if err != nil {
		return Pair{}, err
	}
	if start != json.Delim('{') {
		return Pair{}, fmt.Errorf("expected pair object")
	}
	var pair Pair
	seen := make(map[string]bool, 3)
	for d.More() {
		token, err := d.Token()
		if err != nil {
			return Pair{}, err
		}
		key := token.(string)
		if seen[key] {
			return Pair{}, fmt.Errorf("duplicate pair field %q", key)
		}
		seen[key] = true
		var value json.RawMessage
		if err := d.Decode(&value); err != nil {
			return Pair{}, err
		}
		if bytes.Equal(value, []byte("null")) {
			return Pair{}, fmt.Errorf("pair field %q cannot be null", key)
		}
		var destination any
		switch key {
		case "i":
			destination = &pair.I
		case "j":
			destination = &pair.J
		case "coefficient":
			destination = &pair.Coefficient
		default:
			return Pair{}, fmt.Errorf("unknown pair field %q", key)
		}
		if err := json.Unmarshal(value, destination); err != nil {
			return Pair{}, err
		}
	}
	if _, err := d.Token(); err != nil {
		return Pair{}, err
	}
	if len(seen) != 3 {
		return Pair{}, fmt.Errorf("pair requires i, j, and coefficient")
	}
	return pair, nil
}
