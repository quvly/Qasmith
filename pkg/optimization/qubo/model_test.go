package qubo_test

import (
	"encoding/json"
	"math"
	"math/rand"
	"reflect"
	"strings"
	"testing"

	"github.com/quvly/Qasmith/pkg/optimization/ising"
	"github.com/quvly/Qasmith/pkg/optimization/qubo"
)

func builder(t *testing.T, n int) *qubo.Builder {
	t.Helper()
	b, err := qubo.New(n)
	must(t, err)
	return b
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func build(t *testing.T, b *qubo.Builder) qubo.Model {
	t.Helper()
	model, err := b.Build()
	must(t, err)
	return model
}

func assignments(n int, test func([]bool)) {
	for mask := 0; mask < 1<<n; mask++ {
		bits := make([]bool, n)
		for i := range bits {
			bits[i] = mask&(1<<i) != 0
		}
		test(bits)
	}
}

func near(got, want float64) bool {
	return math.Abs(got-want) <= 1e-12+1e-12*math.Max(math.Abs(got), math.Abs(want))
}

func TestCanonicalSparseModel(t *testing.T) {
	b := builder(t, 4)
	must(t, b.AddLinear(0, 1))
	must(t, b.AddQuadratic(0, 0, -3))
	must(t, b.AddQuadratic(3, 1, -5))
	must(t, b.AddQuadratic(1, 3, 3))
	must(t, b.AddQuadratic(0, 2, 0))
	must(t, b.SetOffset(3))
	must(t, b.AddOffset(-.25))
	m := build(t, b)
	must(t, b.AddLinear(0, 10))
	terms := m.QuadraticTerms()
	if !reflect.DeepEqual(terms, []qubo.QuadraticTerm{{I: 1, J: 3, Coefficient: -2}}) {
		t.Fatalf("pairs %v", terms)
	}
	terms[0].Coefficient = 100
	linear := m.LinearTerms()
	linear[0].Coefficient = 100
	for _, pair := range [][2]int{{1, 3}, {3, 1}} {
		c, err := m.Quadratic(pair[0], pair[1])
		must(t, err)
		if c != -2 {
			t.Fatalf("interaction = %g", c)
		}
	}
	if got, err := m.Linear(2); err != nil || got != 0 {
		t.Fatalf("absent linear = %g, %v", got, err)
	}
	if got, err := m.Quadratic(0, 3); err != nil || got != 0 {
		t.Fatalf("absent interaction = %g, %v", got, err)
	}
	if got, err := m.Evaluate([]bool{true, true, false, true}); err != nil || got != -1.25 {
		t.Fatalf("energy = %g, %v", got, err)
	}
}

func TestInvalidAndStableEnergy(t *testing.T) {
	if _, err := qubo.New(0); err == nil {
		t.Fatal("accepted zero dimension")
	}
	b := builder(t, 3)
	for _, err := range []error{
		b.AddLinear(-1, 0), b.AddQuadratic(0, 3, 1), b.AddLinear(0, math.Inf(-1)),
		b.AddQuadratic(0, 1, math.NaN()), b.SetOffset(math.Inf(1)), b.AddOffset(math.NaN()),
	} {
		if err == nil {
			t.Fatal("accepted invalid term")
		}
	}
	must(t, b.SetOffset(1e16))
	must(t, b.AddLinear(0, 1))
	must(t, b.AddLinear(1, -1e16))
	m := build(t, b)
	if _, err := m.Evaluate([]bool{true}); err == nil {
		t.Fatal("accepted wrong assignment length")
	}
	if got, err := m.Evaluate([]bool{true, true, false}); err != nil || got != 1 {
		t.Fatalf("compensated result %g, %v", got, err)
	}
	must(t, b.SetOffset(math.MaxFloat64))
	must(t, b.AddLinear(2, math.MaxFloat64))
	if err := b.AddLinear(2, math.MaxFloat64); err == nil {
		t.Fatal("accepted coefficient overflow")
	}
	if _, err := build(t, b).Evaluate([]bool{false, false, true}); err == nil {
		t.Fatal("accepted energy overflow")
	}
	// Compensation can overflow the final result even when each ordinary partial sum is finite.
	b = builder(t, 2)
	must(t, b.SetOffset(math.MaxFloat64))
	must(t, b.AddLinear(0, math.Ldexp(1, 969)))
	must(t, b.AddLinear(1, math.Ldexp(1, 969)))
	if _, err := build(t, b).Evaluate([]bool{true, true}); err == nil || !strings.Contains(err.Error(), "final") {
		t.Fatalf("expected final energy overflow, got %v", err)
	}
}

func TestMatrixConventions(t *testing.T) {
	for _, q := range [][][]float64{
		{{-2, 4}, {0, 3}},
		{{-2, 0}, {4, 3}},
		{{-2, 2}, {2, 3}},
		{{-2, 7}, {-3, 3}},
	} {
		model, err := qubo.FromMatrix(q, 1)
		must(t, err)
		for _, tc := range []struct {
			bits   []bool
			energy float64
		}{
			{[]bool{false, false}, 1}, {[]bool{true, false}, -1}, {[]bool{false, true}, 4}, {[]bool{true, true}, 6},
		} {
			got, err := model.Evaluate(tc.bits)
			if err != nil || got != tc.energy {
				t.Fatalf("matrix %v at %v: %g, %v", q, tc.bits, got, err)
			}
		}
	}
	for _, q := range [][][]float64{nil, {{1}, {2}}, {{math.NaN()}},
		{{0, math.Inf(1)}, {0, 0}}, {{0, math.MaxFloat64}, {math.MaxFloat64, 0}},
	} {
		if _, err := qubo.FromMatrix(q, 0); err == nil {
			t.Fatalf("accepted %v", q)
		}
	}
	if _, err := qubo.FromMatrix([][]float64{{1}}, math.Inf(-1)); err == nil {
		t.Fatal("accepted invalid offset")
	}
}

func TestGoldenIsingConversion(t *testing.T) {
	model, err := qubo.FromMatrix([][]float64{{-2, 4}, {0, 3}}, 1)
	must(t, err)
	h, err := model.ToIsing()
	must(t, err)
	field0, err := h.Field(0)
	must(t, err)
	field1, err := h.Field(1)
	must(t, err)
	j, err := h.Coupling(1, 0)
	must(t, err)
	if h.Offset() != 2.5 || field0 != 0 || field1 != -2.5 || j != 1 || len(h.Fields()) != 1 || len(h.Couplings()) != 1 {
		t.Fatalf("conversion: offset %g fields %v pairs %v", h.Offset(), h.Fields(), h.Couplings())
	}
}

func TestCompleteRandomEnergyLandscapes(t *testing.T) {
	rng := rand.New(rand.NewSource(1729))
	for _, sparse := range []bool{false, true} {
		for trial := range 24 {
			n := 1 + trial%7
			matrix := make([][]float64, n)
			for i := range matrix {
				matrix[i] = make([]float64, n)
				for j := range matrix[i] {
					if !sparse || rng.Intn(4) == 0 {
						matrix[i][j] = float64(rng.Intn(41)-20) / 7
					}
				}
			}
			offset := float64(1+rng.Intn(9)) / 11
			model, err := qubo.FromMatrix(matrix, offset)
			must(t, err)
			h, err := model.ToIsing()
			must(t, err)
			assignments(n, func(bits []bool) {
				// Directly multiply xᵀQx using the original matrix, including both triangles.
				want := offset
				for i, row := range matrix {
					if !bits[i] {
						continue
					}
					for j, coefficient := range row {
						if bits[j] {
							want += coefficient
						}
					}
				}
				q, err := model.Evaluate(bits)
				must(t, err)
				e, err := h.Evaluate(ising.Spins(bits))
				must(t, err)
				if !near(q, want) || !near(e, want) || !near(q, e) {
					t.Fatalf("sparse=%v trial=%d bits=%v: direct=%g QUBO=%g Ising=%g", sparse, trial, bits, want, q, e)
				}
			})
		}
	}
}

func TestConversionLargeValues(t *testing.T) {
	b := builder(t, 2)
	must(t, b.SetOffset(math.MaxFloat64))
	must(t, b.AddLinear(0, math.MaxFloat64))
	must(t, b.AddLinear(1, -math.MaxFloat64))
	h, err := build(t, b).ToIsing()
	must(t, err)
	if h.Offset() != math.MaxFloat64 {
		t.Fatalf("offset = %g", h.Offset())
	}
	must(t, b.AddLinear(1, math.MaxFloat64))
	if _, err := build(t, b).ToIsing(); err == nil {
		t.Fatal("accepted nonrepresentable Ising offset")
	}
}

func TestJSONStableAndStrict(t *testing.T) {
	const good = `{"schema":"qasmith.qubo.v1","variables":3,"offset":1.25,"linear":[0,1.0000000000000002,0],"quadratic":[{"i":1,"j":2,"coefficient":-2},{"i":0,"j":2,"coefficient":3}]}`
	var m qubo.Model
	must(t, json.Unmarshal([]byte(good), &m))
	canonical, err := json.Marshal(m)
	must(t, err)
	if !strings.Contains(string(canonical), `"quadratic":[{"i":0,"j":2,"coefficient":3},{"i":1,"j":2,"coefficient":-2}]`) {
		t.Fatalf("not canonical: %s", canonical)
	}
	var recovered qubo.Model
	must(t, json.Unmarshal(canonical, &recovered))
	double, err := json.Marshal(recovered)
	must(t, err)
	if string(double) != string(canonical) {
		t.Fatalf("round-trip changed bytes: %s vs %s", double, canonical)
	}
	for _, bad := range []string{
		"null", "{}",
		strings.Replace(good, "qasmith.qubo.v1", "qasmith.qubo.v2", 1),
		strings.Replace(good, `"variables":3`, `"variables":3,"variables":3`, 1),
		strings.Replace(good, `"variables":3`, `"variables":1000000000`, 1),
		strings.Replace(good, `"offset":1.25`, `"offset":1e999`, 1),
		strings.Replace(good, `"offset":1.25`, `"offset":NaN`, 1),
		strings.Replace(good, `[0,1.0000000000000002,0]`, `[null,0,0]`, 1),
		strings.Replace(good, `"i":0,`, `"i":-1,`, 1),
		strings.Replace(good, `"i":0,`, `"i":2,`, 1),
		strings.Replace(good, `"i":0,`, `"i":1,`, 1),
		strings.Replace(good, `"i":0,`, "", 1),
		strings.Replace(good, `"i":0,`, `"i":0,"i":0,`, 1),
		good + "[]",
	} {
		if err := json.Unmarshal([]byte(bad), &recovered); err == nil {
			t.Fatalf("accepted %s", bad)
		}
	}
	after, err := json.Marshal(recovered)
	must(t, err)
	if string(after) != string(canonical) {
		t.Fatalf("failed decode changed receiver: %s", after)
	}
}
