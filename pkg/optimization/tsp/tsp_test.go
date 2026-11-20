package tsp_test

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/quvly/Qasmith/pkg/optimization/bruteforce"
	"github.com/quvly/Qasmith/pkg/optimization/ising"
	"github.com/quvly/Qasmith/pkg/optimization/qaoa"
	"github.com/quvly/Qasmith/pkg/optimization/tsp"
)

var reference = [][]float64{{0, 10, 15, 20}, {10, 0, 35, 25}, {15, 35, 0, 30}, {20, 25, 30, 0}}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func symmetric(t *testing.T) tsp.Problem {
	t.Helper()
	p, err := tsp.New(reference, 0)
	must(t, err)
	return p
}

func close(a, b float64) bool { return math.Abs(a-b) <= 1e-12+1e-12*math.Max(math.Abs(a), math.Abs(b)) }

func TestProblemAndEncoding(t *testing.T) {
	p := symmetric(t)
	if !p.Symmetric() || p.MaxDistance() != 35 || p.Cities() != 4 {
		t.Fatal(p)
	}
	for _, tc := range []struct {
		d      [][]float64
		s      int
		reason string
	}{
		{[][]float64{{0, 1}, {1, 0}}, 0, "at least three"},
		{[][]float64{{0, 1, 2}, {1}, {2, 3, 0}}, 0, "distances[1]"},
		{reference, 4, "starting city"},
		{[][]float64{{0, 1, 2}, {1, 0, -1}, {2, 3, 0}}, 0, "nonnegative"},
		{[][]float64{{0, 1, 2}, {1, 0, math.NaN()}, {2, 3, 0}}, 0, "finite"},
		{[][]float64{{0, 1, 2}, {1, 0, 3}, {2, math.Inf(1), 0}}, 0, "finite"},
		{[][]float64{{1, 1, 2}, {1, 0, 3}, {2, 3, 0}}, 0, "must be zero"},
	} {
		if _, err := tsp.New(tc.d, tc.s); err == nil || !strings.Contains(err.Error(), tc.reason) {
			t.Fatalf("wanted %s, got %v", tc.reason, err)
		}
	}
	for start := range reference {
		p, err := tsp.New(reference, start)
		must(t, err)
		e, err := p.Encoding()
		must(t, err)
		if e.Variables() != 9 || e.UniformFeasibility() != 6.0/512 {
			t.Fatalf("wrong encoding size or baseline")
		}
		for pos := range 3 {
			i := pos * 3
			for city := range reference {
				if city == start {
					continue
				}
				index, err := e.Index(city, pos)
				must(t, err)
				if index != i {
					t.Fatalf("start %d city %d pos %d got %d, want %d", start, city, pos, index, i)
				}
				c, p, err := e.Variable(i)
				must(t, err)
				if c != city || p != pos {
					t.Fatalf("inverse %d/%d", c, p)
				}
				i++
			}
		}
		if _, err := e.Index(start, 0); err == nil {
			t.Fatal("start encoded")
		}
		if _, err := e.Index((start+1)%4, 3); err == nil {
			t.Fatal("invalid position")
		}
		if _, _, err := e.Variable(9); err == nil {
			t.Fatal("invalid index")
		}
	}
}

func TestJSONAndImmutableDistances(t *testing.T) {
	const good = `{"schema":"qasmith.tsp.v1","name":"example","start":1,"distances":[[0,1,2],[4,0,8],[9,3,0]]}`
	var p tsp.Problem
	must(t, json.Unmarshal([]byte(good), &p))
	if p.Symmetric() || p.Start() != 1 || p.Name() != "example" {
		t.Fatal("directed input lost")
	}
	for _, bad := range []string{
		"null", "{}", strings.Replace(good, "tsp.v1", "tsp.v2", 1),
		strings.Replace(good, `"start":1,`, "", 1), strings.Replace(good, `"start":1`, `"start":null`, 1),
		strings.Replace(good, `"start":1`, `"start":1,"start":0`, 1),
		strings.Replace(good, `0,1,2`, `0,1,null`, 1), strings.Replace(good, `0,1,2`, `0,1,1e999`, 1),
	} {
		if err := json.Unmarshal([]byte(bad), &p); err == nil {
			t.Fatalf("accepted %s", bad)
		}
	}
	matrix := [][]float64{{0, 1, 2}, {4, 0, 8}, {9, 3, 0}}
	q, err := tsp.New(matrix, 0)
	must(t, err)
	matrix[1][2] = 0
	if value, err := q.Distance(1, 2); err != nil || value != 8 {
		t.Fatalf("copied distance %g %v", value, err)
	}
}

func TestCompleteAssignmentLandscape(t *testing.T) {
	for _, tc := range []struct {
		name     string
		matrix   [][]float64
		start    int
		distance float64
		ties     int
	}{
		{"reference", reference, 0, 80, 2},
		{"directed", [][]float64{{0, 5, 18, 7}, {2, 0, 15, 3}, {4, 6, 0, 10}, {9, 12, 1, 0}}, 2, 13, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, err := tsp.New(tc.matrix, tc.start)
			must(t, err)
			penalty, err := p.DefaultPenalty()
			must(t, err)
			if penalty <= float64(p.Cities())*p.MaxDistance() {
				t.Fatalf("unsafe penalty %g", penalty)
			}
			q, e, err := tsp.ToQUBO(p, tsp.EncodeOptions{})
			must(t, err)
			h, err := q.ToIsing()
			must(t, err)
			best, err := tsp.SolveClassical(context.Background(), p, 0)
			must(t, err)
			if best.Distance != tc.distance || best.OptimalTours != tc.ties || best.ToursChecked != 6 {
				t.Fatalf("classical %+v", best)
			}
			ground, err := bruteforce.Solve(q)
			must(t, err)
			if ground.Energy != tc.distance {
				t.Fatalf("QUBO minimum %+v", ground)
			}
			var cities []int
			for city := range tc.matrix {
				if city != tc.start {
					cities = append(cities, city)
				}
			}
			var feasible, optimal int
			for mask := 0; mask < 512; mask++ {
				bits := make([]bool, e.Variables())
				for i := range bits {
					bits[i] = mask&(1<<i) != 0
				}
				// Independent polynomial and separate row/column constraint counts.
				var travel, posCost, cityCost float64
				var occupancy [3][3]float64
				for pos := range 3 {
					for c := range 3 {
						if bits[pos*3+c] {
							occupancy[pos][c] = 1
						}
					}
				}
				for i, city := range cities {
					travel += tc.matrix[tc.start][city]*occupancy[0][i] + tc.matrix[city][tc.start]*occupancy[2][i]
					for j, next := range cities {
						travel += tc.matrix[city][next] * (occupancy[0][i]*occupancy[1][j] + occupancy[1][i]*occupancy[2][j])
					}
				}
				for i := range 3 {
					row := occupancy[i][0] + occupancy[i][1] + occupancy[i][2] - 1
					col := occupancy[0][i] + occupancy[1][i] + occupancy[2][i] - 1
					posCost += penalty * row * row
					cityCost += penalty * col * col
				}
				energy, err := q.Evaluate(bits)
				must(t, err)
				spin, err := h.Evaluate(ising.Spins(bits))
				must(t, err)
				if !close(energy, travel+posCost+cityCost) || !close(energy, spin) {
					t.Fatalf("mask %d QUBO=%g Ising=%g travel=%g penalties=%g,%g", mask, energy, spin, travel, posCost, cityCost)
				}
				decoded, err := p.Decode(bits)
				must(t, err)
				if decoded.Feasible {
					feasible++
					direct, err := p.RouteDistance(decoded.Route)
					must(t, err)
					if posCost != 0 || cityCost != 0 || direct != decoded.Distance || !close(direct, energy) {
						t.Fatal(decoded)
					}
					if energy == ground.Energy {
						optimal++
					}
				} else if energy <= ground.Energy || posCost+cityCost <= 0 || decoded.Route != nil {
					t.Fatalf("bad invalid assignment %d: %+v energy %g", mask, decoded, energy)
				}
			}
			if feasible != 6 || optimal != tc.ties {
				t.Fatalf("feasible=%d optimal=%d", feasible, optimal)
			}
		})
	}
}

func TestDecoderAndClassicalLimits(t *testing.T) {
	p := symmetric(t)
	for _, tc := range []struct {
		bits  string
		route []int
	}{
		{"100001010", []int{0, 1, 3, 2, 0}},
		{"010001100", []int{0, 2, 3, 1, 0}},
	} {
		d, err := p.DecodeBitstring(tc.bits)
		must(t, err)
		if !d.Feasible || d.Distance != 80 || !reflect.DeepEqual(d.Route, tc.route) {
			t.Fatalf("decoded %+v", d)
		}
	}
	for _, bits := range []string{"0", "000000000", "110001000", "100100001"} {
		d, err := p.DecodeBitstring(bits)
		must(t, err)
		if d.Feasible || len(d.Violations) == 0 || d.Route != nil {
			t.Fatalf("accepted %s: %+v", bits, d)
		}
	}
	if _, err := p.DecodeBitstring("1x0"); err == nil {
		t.Fatal("nonbinary accepted")
	}
	if _, err := p.RouteDistance([]int{0, 1, 1, 2, 0}); err == nil {
		t.Fatal("invalid route accepted")
	}
	best, err := tsp.SolveClassical(context.Background(), p, 0)
	must(t, err)
	if !reflect.DeepEqual(best.Route, []int{0, 1, 3, 2, 0}) {
		t.Fatalf("tie break %+v", best)
	}
	if _, err := tsp.SolveClassical(context.Background(), p, 3); err == nil {
		t.Fatal("city limit ignored")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := tsp.SolveClassical(ctx, p, 4); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation %v", err)
	}
}

func TestPenaltyOverridesAndRange(t *testing.T) {
	p := symmetric(t)
	for _, v := range []float64{0, -1, math.NaN(), math.Inf(1), math.MaxFloat64} {
		if _, _, err := tsp.ToQUBO(p, tsp.EncodeOptions{Penalty: &v}); err == nil {
			t.Fatalf("accepted penalty %g", v)
		}
	}
	for _, scale := range []float64{1e16, math.MaxFloat64} {
		p, err := tsp.New([][]float64{{0, scale, 0}, {0, 0, 0}, {0, 0, 0}}, 0)
		must(t, err)
		value, err := p.DefaultPenalty()
		if scale == math.MaxFloat64 {
			if err == nil || !strings.Contains(err.Error(), "rescale") {
				t.Fatalf("overflow diagnostic %v", err)
			}
		} else if err != nil || value <= 3*scale {
			t.Fatalf("unsafe %g: %v", value, err)
		}
	}
	if _, _, err := tsp.ToQUBO(p, tsp.EncodeOptions{MaxVariables: 8}); err == nil {
		t.Fatal("encoding budget ignored")
	}
	custom := 1.0
	a, err := tsp.Prepare(p, tsp.EncodeOptions{Penalty: &custom})
	must(t, err)
	ground, err := bruteforce.Solve(a.QUBO())
	must(t, err)
	d, err := p.Decode(ground.Assignment)
	must(t, err)
	if d.Feasible || ground.Energy >= 80 {
		t.Fatalf("low penalty unexpectedly guaranteed feasibility: %+v", ground)
	}
	samples := qaoa.Evaluation{Mode: qaoa.Sampled, QubitCount: 9, LayerCount: 1, Shots: 7, Counts: map[string]int{"000000000": 2, "100001010": 3, "010001100": 2}}
	result, err := a.Report(context.Background(), samples, tsp.ReferenceOptions{Enabled: true})
	must(t, err)
	if result.BestObserved.Decoded.Feasible || result.BestObserved.ConstraintPenalty != 6 ||
		result.BestFeasible.Decoded.Distance != 80 || result.BestFeasible.ConstraintPenalty != 0 || result.OptimalTourSamples != 5 ||
		*result.OptimalSampleRate != 5.0/7 || result.FeasibleSamples != 5 || result.InfeasibleSamples != 2 {
		t.Fatalf("separate metrics %+v", result)
	}
	samples.Shots, samples.Counts = 1, map[string]int{"000000000": 1}
	result, err = a.Report(context.Background(), samples, tsp.ReferenceOptions{Enabled: true, MaxCities: 3})
	must(t, err)
	if result.BestFeasible != nil || result.Classical != nil || result.OptimalSampleRate != nil || result.Raw.Counts["000000000"] != 1 {
		t.Fatalf("no feasible samples %+v", result)
	}
	samples.Counts = map[string]int{"00000000x": 1}
	if _, err := a.Report(context.Background(), samples, tsp.ReferenceOptions{}); err == nil {
		t.Fatal("invalid count data accepted")
	}
}
