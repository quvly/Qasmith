package qasmith_test

import (
	"errors"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/quvly/Qasmith/pkg/ir"
	"github.com/quvly/Qasmith/pkg/qasmith"
)

func rzzFixture(t *testing.T, source string) string {
	t.Helper()
	module, err := os.ReadFile("../../examples/modules/rotations.smith")
	if err != nil {
		t.Fatal(err)
	}
	return fixture(t, map[string]string{
		"main.smith":      source,
		"rotations.smith": string(module),
		"wrapper.smith": `import { rzz as apply } from "./rotations.smith";
pub fn turn(theta: Float, a: Qubit, b: Qubit) -> Void {
    apply(theta / 2, b, a);
}
`,
	})
}

func TestRZZRejectsAliasedQubits(t *testing.T) {
	for _, tc := range []struct{ path, function string }{
		{"rotations", "rzz"}, {"wrapper", "turn"}, {"rotations", "rotatePair"},
	} {
		t.Run(tc.function, func(t *testing.T) {
			path := rzzFixture(t, `import { `+tc.function+` as rotate } from "./`+tc.path+`.smith";
fn main() -> MeasuredResult {
    let q = qubit();
    rotate(pi, q, q);
    return measure(q);
}
`)
			err := (qasmith.Compiler{}).Check(path)
			var detail *qasmith.Diagnostic
			if !errors.As(err, &detail) || detail.Code != "E3100" || detail.Position.Line != 4 ||
				!strings.Contains(detail.Message, "distinct qubit operands (parameters 2 and 3)") {
				t.Fatalf("expected distinctness diagnostic at caller, got %v", err)
			}
		})
	}
}

func TestRZZForwardingOriginAndPass(t *testing.T) {
	path := rzzFixture(t, `import { turn } from "./wrapper.smith";
fn main() -> MeasuredResult {
    let a = qubit();
    let b = qubit();
    x(a);
    x(a);
    turn(pi, a, b);
    return measure(a, b);
}
`)
	baseline, err := (qasmith.Compiler{}).Lower(path)
	if err != nil {
		t.Fatal(err)
	}
	compiler := qasmith.Compiler{Passes: []ir.Pass{qasmith.CancelAdjacentX{}}}
	optimized, err := compiler.Lower(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(baseline.Operations) != 9 || len(optimized.Operations) != 7 ||
		!reflect.DeepEqual(baseline.Operations[4:], optimized.Operations[2:]) {
		t.Fatalf("pass changed rotation, measurement or metadata: %v", optimized.Operations)
	}
	for i, want := range []ir.ApplyGate{
		{Gate: ir.CX, Qubits: []ir.QubitID{1, 0}},
		{Gate: ir.RZ, Qubits: []ir.QubitID{0}, Parameters: []float64{math.Pi / 2}},
		{Gate: ir.CX, Qubits: []ir.QubitID{1, 0}},
	} {
		got := optimized.Operations[2+i].(ir.ApplyGate)
		origin := got.Origin
		got.Origin = ir.Origin{}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("gate %d = %+v, want %+v", i, got, want)
		}
		if filepath.Base(origin.File) != "rotations.smith" || origin.Line != i+2 || origin.Column != 5 ||
			len(origin.CallSites) != 2 ||
			filepath.Base(origin.CallSites[0].File) != "wrapper.smith" || origin.CallSites[0].Line != 3 || origin.CallSites[0].Column != 5 ||
			filepath.Base(origin.CallSites[1].File) != "main.smith" || origin.CallSites[1].Line != 7 || origin.CallSites[1].Column != 5 {
			t.Fatalf("unexpected source origin %+v", origin)
		}
	}
}
