package qasmith_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/cmplx"
	"math/rand"
	"reflect"
	"strings"
	"testing"

	"github.com/quvly/Qasmith/pkg/extension"
	"github.com/quvly/Qasmith/pkg/optimization/plan"
	"github.com/quvly/Qasmith/pkg/optimization/qaoa"
	"github.com/quvly/Qasmith/pkg/optimization/qubo"
	"github.com/quvly/Qasmith/pkg/qasmith"
)

const quboImports = `import { qubo, fromMatrix, addLinear, addQuadratic, addExactlyOne, addSquaredLinear, scale, combine } from "std/optimization/qubo";
import { qaoa } from "std/optimization/qaoa";
`

const quboExamples = "../../examples/optimization/qubo-language/"

var testMatrix = [][]float64{{-2, 4, 0}, {0, 3, -1}, {0, 0, -.5}}

func compileQUBO(t *testing.T, path string) (qasmith.Prepared, plan.Plan) {
	t.Helper()
	c := qasmith.Compiler{}
	requireOK(t, c.Check(path))
	prepared, err := c.Prepare(path)
	requireOK(t, err)
	p, ok := prepared.Plan.(plan.Plan)
	if !ok {
		t.Fatal("expected variational plan")
	}
	return prepared, p
}

func compareModels(t *testing.T, got, want qubo.Model) {
	t.Helper()
	left, err := json.Marshal(got)
	requireOK(t, err)
	right, err := json.Marshal(want)
	requireOK(t, err)
	if string(left) != string(right) {
		t.Fatalf("Smith model %s\nGo model %s", left, right)
	}
	for mask := range 1 << want.Variables() {
		bits := assignment(mask, want.Variables())
		a, err := got.Evaluate(bits)
		requireOK(t, err)
		b, err := want.Evaluate(bits)
		requireOK(t, err)
		if a != b {
			t.Fatalf("mask %d energy %g, want %g", mask, a, b)
		}
	}
}

func assignment(mask, n int) []bool {
	bits := make([]bool, n)
	for i := range bits {
		bits[i] = mask&(1<<i) != 0
	}
	return bits
}

func directMatrix(bits []bool, matrix [][]float64, offset float64) float64 {
	for i, row := range matrix {
		for j, x := range row {
			if bits[i] && bits[j] {
				offset += x
			}
		}
	}
	return offset
}

func closeEnergy(a, b float64) bool {
	return math.Abs(a-b) <= 2e-12*(1+math.Max(math.Abs(a), math.Abs(b)))
}

func compareQuantum(t *testing.T, path string, model qubo.Model) {
	t.Helper()
	ctx := context.Background()
	prepared, p := compileQUBO(t, path)
	if p.Problem().HasDecoder() {
		t.Fatal("generic model gained a decoder")
	}
	compareModels(t, p.Problem().QUBO(), model)
	reference, err := qaoa.FromQUBO(model)
	requireOK(t, err)
	wantIsing, err := json.Marshal(reference.Ising())
	requireOK(t, err)
	gotIsing, err := json.Marshal(p.Problem().QAOA().Ising())
	requireOK(t, err)
	if string(gotIsing) != string(wantIsing) {
		t.Fatalf("Ising differs: %s vs %s", gotIsing, wantIsing)
	}
	params := p.Parameters()
	want, err := reference.Build(params, qaoa.BuildOptions{Measure: true})
	requireOK(t, err)
	if !reflect.DeepEqual(normalized(prepared.Circuit), normalized(want)) {
		t.Fatal("different normalized circuit")
	}
	a, err := prepared.Export()
	requireOK(t, err)
	b, err := reference.Export(params, qaoa.BuildOptions{Measure: true}, nil)
	requireOK(t, err)
	if string(a) != string(b) {
		t.Fatal("different OpenQASM")
	}
	state, _, err := qasmith.Statevector(ctx, prepared.Circuit)
	requireOK(t, err)
	expected, _, err := qasmith.Statevector(ctx, want)
	requireOK(t, err)
	var exact float64
	for i, amp := range state {
		if !(cmplx.Abs(amp-expected[i]) <= 2e-12) {
			t.Fatalf("different complex amplitude %d", i)
		}
		energy, err := model.Evaluate(assignment(i, model.Variables()))
		requireOK(t, err)
		exact += cmplx.Abs(amp) * cmplx.Abs(amp) * energy
	}
	obj, err := qaoa.EvaluateExact(ctx, reference, params)
	requireOK(t, err)
	if !closeEnergy(exact, obj.ExpectedEnergy) {
		t.Fatalf("exact objective %g vs %g", exact, obj.ExpectedEnergy)
	}
	raw, err := prepared.Execute(ctx, 200, 42)
	requireOK(t, err)
	samples, err := qaoa.EvaluateSampled(ctx, reference, params, 200, 42)
	requireOK(t, err)
	if !reflect.DeepEqual(raw.Raw.Counts, samples.Counts) || raw.Optimization.Domain != nil ||
		!closeEnergy(raw.Optimization.ExpectedQUBOEnergy, samples.ExpectedEnergy) {
		t.Fatal("different sampled objective")
	}
}

func TestSmithQUBOModelAndCircuitEquivalence(t *testing.T) {
	base, err := qubo.FromMatrix(testMatrix, 1)
	requireOK(t, err)
	for mask := range 8 {
		bits := assignment(mask, 3)
		measured, err := base.Evaluate(bits)
		requireOK(t, err)
		if measured != directMatrix(bits, testMatrix, 1) || mask == 5 && measured != -1.5 {
			t.Fatalf("mask %d energy %g", mask, measured)
		}
	}
	for _, file := range []string{"matrix.smith", "builder.smith", "packaged.smith"} {
		compareQuantum(t, quboExamples+file, base)
	}
	b, err := qubo.New(3)
	requireOK(t, err)
	requireOK(t, b.AddModel(base))
	requireOK(t, b.AddExactlyOne([]int{0, 1, 2}, 10))
	constrained, err := b.Build()
	requireOK(t, err)
	compareQuantum(t, "../../program.smith", constrained)
	compareQuantum(t, quboExamples+"imported.smith", constrained)
	requireOK(t, b.AddSquaredPenalty([]qubo.Weight{{Index: 0, Coefficient: 1.5}, {Index: 1, Coefficient: -2}, {Index: 2, Coefficient: .5}}, 1.5, 2))
	both, err := b.Build()
	requireOK(t, err)
	compareQuantum(t, quboExamples+"constraints.smith", both)
	_, p := compileQUBO(t, quboExamples+"constraints.smith")
	for mask := range 8 {
		bits := assignment(mask, 3)
		var weight, count float64
		for i, w := range []float64{1.5, -2, .5} {
			if bits[i] {
				weight += w
				count++
			}
		}
		actual, err := p.Problem().QUBO().Evaluate(bits)
		requireOK(t, err)
		expected := directMatrix(bits, testMatrix, 1) + 10*(count-1)*(count-1) + 2*(weight-1.5)*(weight-1.5)
		if actual != expected {
			t.Fatalf("constraint energy for %d: %g vs %g", mask, actual, expected)
		}
	}
}

func TestSmithRandomizedCompleteLandscapes(t *testing.T) {
	random := rand.New(rand.NewSource(3108))
	for trial := range 16 {
		n := 1 + trial%4
		offset := float64(random.Intn(13)-6) / 7
		matrix := make([][]float64, n)
		rows := make([]string, n)
		var builder strings.Builder
		fmt.Fprintf(&builder, "let m0 = qubo(%d, %.17f);\n", n, offset)
		for i := range n {
			matrix[i] = make([]float64, n)
			row := make([]string, n)
			for j := range n {
				if trial%2 == 0 || random.Intn(3) == 0 {
					matrix[i][j] = float64(random.Intn(15)-7) / 7
				}
				row[j] = fmt.Sprintf("%.17f", matrix[i][j])
			}
			rows[i] = "[" + strings.Join(row, ",") + "]"
		}
		step := 0
		for i := range n {
			fmt.Fprintf(&builder, "let m%d = addLinear(m%d, %d, %.17f);\n", step+1, step, i, matrix[i][i])
			step++
			for j := i + 1; j < n; j++ {
				fmt.Fprintf(&builder, "let m%d = addQuadratic(m%d, %d, %d, %.17f);\n", step+1, step, i, j, matrix[i][j])
				step++
				fmt.Fprintf(&builder, "let m%d = addQuadratic(m%d, %d, %d, %.17f);\n", step+1, step, j, i, matrix[j][i])
				step++
			}
		}
		expected, err := qubo.FromMatrix(matrix, offset)
		requireOK(t, err)
		for _, body := range []string{
			fmt.Sprintf("let model = fromMatrix([%s],%.17f); return qaoa(model,[0.4],[0.3]);", strings.Join(rows, ","), offset),
			builder.String() + fmt.Sprintf("return qaoa(m%d,[0.4],[0.3]);", step),
		} {
			source := fixture(t, map[string]string{"main.smith": quboImports + "fn main() -> MeasuredResult {\n" + body + "\n}"})
			_, p := compileQUBO(t, source)
			compareModels(t, p.Problem().QUBO(), expected)
			for mask := range 1 << n {
				bits := assignment(mask, n)
				want := directMatrix(bits, matrix, offset)
				got, err := p.Problem().QUBO().Evaluate(bits)
				requireOK(t, err)
				if !closeEnergy(got, want) {
					t.Fatalf("trial %d mask %d energy %g, want %g", trial, mask, got, want)
				}
			}
		}
	}
}

func TestSmithCompositionAndSharedInputs(t *testing.T) {
	left, err := qubo.FromMatrix([][]float64{{-2, 0}, {0, 3}}, 0)
	requireOK(t, err)
	right, err := qubo.FromMatrix([][]float64{{0, 2}, {0, 1}}, 1.25)
	requireOK(t, err)
	scaled, err := right.Scale(.5)
	requireOK(t, err)
	builder, err := qubo.New(2)
	requireOK(t, err)
	requireOK(t, builder.AddModel(left))
	requireOK(t, builder.AddModel(scaled))
	expected, err := builder.Build()
	requireOK(t, err)
	compareQuantum(t, quboExamples+"composition.smith", expected)
	_, p := compileQUBO(t, quboExamples+"composition.smith")
	for mask := range 4 {
		bits := assignment(mask, 2)
		want := directMatrix(bits, [][]float64{{-2, 0}, {0, 3}}, 0) + .5*directMatrix(bits, [][]float64{{0, 2}, {0, 1}}, 1.25)
		got, err := p.Problem().QUBO().Evaluate(bits)
		requireOK(t, err)
		if got != want {
			t.Fatalf("mask %d composition %g, want %g", mask, got, want)
		}
	}
	path := fixture(t, map[string]string{
		"main.smith": quboImports + `import { edit } from "./worker.smith";
fn main() -> MeasuredResult {
    let base = fromMatrix([[-2.0,4.0,0.0],[0.0,3.0,-1.0],[0.0,0.0,-0.5]],1);
    let first = edit(base, 2.0);
    let second = edit(base, 3.0);
    let model = combine(base, combine(first, second));
    return qaoa(model, [0.4], [0.3]);
}`,
		"worker.smith": quboImports + `pub fn edit(base: OptimizationProblem, weight: Float) -> OptimizationProblem {
    let a = addLinear(base, 0, 5);
    let b = addQuadratic(a, 1, 1, -0.25);
    return addSquaredLinear(b, [0,1,2], [1.5,-2.0,0.5], 1.5, weight);
}`,
	})
	_, p = compileQUBO(t, path)
	for mask := range 8 {
		bits := assignment(mask, 3)
		x := make([]float64, 3)
		for i, b := range bits {
			if b {
				x[i] = 1
			}
		}
		residual := 1.5*x[0] - 2*x[1] + .5*x[2] - 1.5
		want := 3*directMatrix(bits, testMatrix, 1) + 10*x[0] - .5*x[1] + 5*residual*residual
		got, err := p.Problem().QUBO().Evaluate(bits)
		requireOK(t, err)
		if got != want {
			t.Fatalf("shared inputs changed at mask %d: %g vs %g", mask, got, want)
		}
	}
}

func TestSmithQUBODiagnostics(t *testing.T) {
	for _, tc := range []struct {
		name, body, want string
		check            bool
		limits           qasmith.Limits
	}{
		{"model type", `return qaoa(addLinear(1.0,0,2),[1],[1]);`, "expects OptimizationProblem", true, qasmith.Limits{}},
		{"square", `return qaoa(fromMatrix([[0,1],[3,4],[1,2]],0),[1],[1]);`, "matrix must be square", true, qasmith.Limits{}},
		{"ragged", `return qaoa(fromMatrix([[0,1],[2]],0),[1],[1]);`, "consistent dimensions", true, qasmith.Limits{}},
		{"matrix precision", `return qaoa(fromMatrix([[9007199254740993]],0),[1],[1]);`, "cannot be represented exactly", false, qasmith.Limits{}},
		{"matrix dimensions", `return qaoa(fromMatrix([[0,1],[3,4]],0),[1],[1]);`, "matrix element limit", true, qasmith.Limits{MaxMatrixElements: 3}},
		{"size", `return qaoa(qubo(0,0),[1],[1]);`, "variable count must be positive", false, qasmith.Limits{}},
		{"variable limit", `return qaoa(qubo(3,0),[1],[1]);`, "variable limit", false, qasmith.Limits{MaxModelVariables: 2}},
		{"index", `return qaoa(addLinear(qubo(3,0),5,1),[1],[1]);`, "index 5 is outside model range 0..2", false, qasmith.Limits{}},
		{"invalid penalty", `return qaoa(addExactlyOne(qubo(3,0),[0,1],-1),[1],[1]);`, "finite and nonnegative", false, qasmith.Limits{}},
		{"index array type", `return qaoa(addExactlyOne(qubo(3,0),[0.0,1.0],1),[1],[1]);`, "rank-1 Int array", true, qasmith.Limits{}},
		{"duplicate index", `return qaoa(addExactlyOne(qubo(3,0),[0,0],1),[1],[1]);`, "duplicate penalty variable", false, qasmith.Limits{}},
		{"weight lengths", `return qaoa(addSquaredLinear(qubo(3,0),[0,1],[1.0],0,1),[1],[1]);`, "equal lengths", true, qasmith.Limits{}},
		{"combine size", `return qaoa(combine(qubo(2,0),qubo(3,0)),[1],[1]);`, "different variable counts", false, qasmith.Limits{}},
		{"scale overflow", `return qaoa(scale(qubo(1,1e308),2),[1],[1]);`, "finite", false, qasmith.Limits{}},
		{"nonfinite coefficient", `return qaoa(addLinear(qubo(1,0),0,1e308*2),[1],[1]);`, "non-finite Float", true, qasmith.Limits{}},
		{"coefficient limit", `return qaoa(addLinear(qubo(1,0),0,2),[1],[1]);`, "model coefficient limit", false, qasmith.Limits{MaxModelCoefficients: 1}},
		{"work limit", `return qaoa(addExactlyOne(qubo(4,0),[0,1,2,3],2),[1],[1]);`, "structured evaluation work limit", false, qasmith.Limits{MaxStructuredWork: 8}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := fixture(t, map[string]string{"main.smith": quboImports + "fn main() -> MeasuredResult { " + tc.body + " }"})
			c := qasmith.Compiler{Limits: tc.limits}
			err := c.Check(path)
			if !tc.check {
				requireOK(t, err)
				_, err = c.Compile(path)
			}
			var detail *qasmith.Diagnostic
			if !errors.As(err, &detail) || detail.Position.Line == 0 || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("wanted diagnostic %q, got %v", tc.want, err)
			}
		})
	}
}

func TestDomainDecoderProtectionAndDeferredConversion(t *testing.T) {
	for _, operation := range []string{"addLinear(p,0,1)", "addQuadratic(p,0,1,1)", "addExactlyOne(p,[0,1],2)",
		"addSquaredLinear(p,[0],[1],0,2)", "scale(p,0)", "combine(p,base)", "combine(base,p)"} {
		path := fixture(t, map[string]string{"main.smith": quboImports + `import { tsp } from "std/optimization/tsp";
fn main() -> MeasuredResult {
    let p = tsp([[0,1,2],[1,0,3],[2,3,0]],0);
    let base = qubo(4,0);
    return qaoa(` + operation + `,[1],[1]);
}`})
		c := qasmith.Compiler{}
		requireOK(t, c.Check(path))
		if _, err := c.Compile(path); err == nil || !strings.Contains(err.Error(), "domain decoder") {
			t.Fatalf("%s: %v", operation, err)
		}
	}
	path := fixture(t, map[string]string{"main.smith": quboImports + `fn main() -> MeasuredResult {
    let big = addLinear(qubo(1, 1.7e308), 0, 1.7e308);
    return qaoa(scale(big, 0.0), [0.4], [0.3]);
}`})
	_, p := compileQUBO(t, path)
	if p.Problem().QUBO().Offset() != 0 || len(p.Problem().QUBO().LinearTerms()) != 0 {
		t.Fatal("zero scale incorrect")
	}
}

func TestQUBOModuleDiscoveryAndImportedDiagnostic(t *testing.T) {
	missing := fixture(t, map[string]string{"main.smith": `import { missing } from "std/optimization/qubo"; fn main() -> Int { return 0; }`})
	if err := (qasmith.Compiler{}).Check(missing); err == nil || !strings.Contains(err.Error(), "no exported symbol") {
		t.Fatalf("wrong unknown export diagnostic: %v", err)
	}
	r := &extension.Registry{}
	requireOK(t, r.RegisterModule("std/optimization/qubo", extension.Function{Name: "fake", Return: extension.Int,
		Evaluate: func(extension.StaticContext, []extension.Value) (extension.Value, error) {
			return extension.Value{Type: extension.Int}, nil
		}}))
	if err := (qasmith.Compiler{Registry: r}).Check(missing); err == nil || !strings.Contains(err.Error(), "duplicate trusted module") {
		t.Fatalf("conflicting module accepted: %v", err)
	}
	source := fixture(t, map[string]string{
		"main.smith": `import { build } from "./helper.smith"; import { qaoa } from "std/optimization/qaoa";
fn main() -> MeasuredResult { return qaoa(build(), [0.4], [0.3]); }`,
		"helper.smith": `import { qubo as empty, addLinear } from "std/optimization/qubo";
pub fn build() -> OptimizationProblem { return addLinear(empty(2,0), 10, 3); }`,
	})
	c := qasmith.Compiler{}
	requireOK(t, c.Check(source))
	_, err := c.Compile(source)
	var diagnostic *qasmith.Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.Code != "E4001" || !strings.HasSuffix(diagnostic.Position.File, "helper.smith") ||
		!strings.Contains(err.Error(), "called from ") || !strings.Contains(err.Error(), "main.smith") {
		t.Fatalf("missing call-site context: %v", err)
	}
}
