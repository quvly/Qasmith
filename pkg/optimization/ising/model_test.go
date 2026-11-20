package ising_test

import (
	"encoding/json"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/quvly/Qasmith/pkg/optimization/ising"
)

func TestIsingModel(t *testing.T) {
	fields := []ising.Field{{Index: 2, Coefficient: -2}, {Index: 0, Coefficient: 1.5}}
	couplings := []ising.Coupling{{I: 1, J: 2, Coefficient: 3}, {I: 0, J: 2, Coefficient: .25}}
	h, err := ising.New(3, -.25, fields, couplings)
	if err != nil {
		t.Fatal(err)
	}
	fields[0].Coefficient = 0
	couplings[0].Coefficient = 0
	cloned := h.Couplings()
	cloned[0].Coefficient = 0
	spins := ising.Spins([]bool{true, false, true})
	if !reflect.DeepEqual(spins, []int8{-1, 1, -1}) {
		t.Fatalf("spins %v", spins)
	}
	if got, err := h.Evaluate(spins); err != nil || got != -2.5 {
		t.Fatalf("energy %g: %v", got, err)
	}
	if c, err := h.Coupling(2, 0); err != nil || c != .25 {
		t.Fatalf("pair %g, %v", c, err)
	}
	if c, err := h.Field(1); err != nil || c != 0 {
		t.Fatalf("missing field %g, %v", c, err)
	}
	for _, bad := range [][]int8{{-1, 0, 1}, {-1, 1, 2}, {-1, 1}} {
		if _, err := h.Evaluate(bad); err == nil {
			t.Fatalf("accepted %v", bad)
		}
	}
	if _, err := h.Coupling(0, 0); err == nil {
		t.Fatal("accepted self coupling")
	}
	if _, err := h.Field(3); err == nil {
		t.Fatal("accepted out-of-range field")
	}
}

func TestIsingValidationAndJSON(t *testing.T) {
	for _, fields := range [][]ising.Field{
		{{Index: 2, Coefficient: 1}}, {{Index: 0, Coefficient: math.NaN()}},
		{{Index: 1, Coefficient: 1}, {Index: 1, Coefficient: 2}},
	} {
		if _, err := ising.New(2, 0, fields, nil); err == nil {
			t.Fatalf("accepted %v", fields)
		}
	}
	for _, pairs := range [][]ising.Coupling{
		{{I: 0, J: 2, Coefficient: 1}}, {{I: 1, J: 0, Coefficient: 1}}, {{I: 0, J: 0, Coefficient: 1}},
		{{I: 0, J: 1, Coefficient: math.Inf(-1)}}, {{I: 0, J: 1, Coefficient: 1}, {I: 0, J: 1, Coefficient: 2}},
	} {
		if _, err := ising.New(2, 0, nil, pairs); err == nil {
			t.Fatalf("accepted %v", pairs)
		}
	}
	h, err := ising.New(2, math.MaxFloat64, []ising.Field{{Index: 0, Coefficient: math.MaxFloat64}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.Evaluate([]int8{1, -1}); err == nil {
		t.Fatal("accepted energy overflow")
	}
	h, err = ising.New(2, .7, []ising.Field{{Index: 1, Coefficient: math.SmallestNonzeroFloat64}}, []ising.Coupling{{I: 0, J: 1, Coefficient: -.2}})
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(h)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"schema":"qasmith.ising.v1"`) {
		t.Fatalf("wire format %s", data)
	}
	var restored ising.Model
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(restored)
	if err != nil || string(encoded) != string(data) || !reflect.DeepEqual(restored.Fields(), h.Fields()) {
		t.Fatalf("round-trip %s, %v", encoded, err)
	}
	if err := json.Unmarshal([]byte(strings.Replace(string(data), "ising.v1", "qubo.v1", 1)), &restored); err == nil {
		t.Fatal("accepted wrong schema")
	}
}
