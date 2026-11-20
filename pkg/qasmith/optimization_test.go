package qasmith_test

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"math/cmplx"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/quvly/Qasmith/pkg/extension"
	"github.com/quvly/Qasmith/pkg/ir"
	"github.com/quvly/Qasmith/pkg/optimization/plan"
	"github.com/quvly/Qasmith/pkg/optimization/qaoa"
	"github.com/quvly/Qasmith/pkg/optimization/qubo"
	"github.com/quvly/Qasmith/pkg/optimization/tsp"
	"github.com/quvly/Qasmith/pkg/qasmith"
)

func requireOK(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func normalized(circuit *ir.Program) []ir.Operation {
	out := make([]ir.Operation, 0, len(circuit.Operations))
	for _, op := range circuit.Operations {
		switch op := op.(type) {
		case ir.AllocateQubit:
			op.Origin = ir.Origin{}
			out = append(out, op)
		case ir.ApplyGate:
			op.Origin = ir.Origin{}
			out = append(out, op)
		case ir.Measure:
			op.Origin = ir.Origin{}
			out = append(out, op)
		}
	}
	return out
}

func TestSmithOptimizationEquivalence(t *testing.T) {
	ctx := context.Background()
	data, err := os.ReadFile("../../examples/optimization/tsp/problem.json")
	requireOK(t, err)
	var model tsp.Problem
	requireOK(t, json.Unmarshal(data, &model))
	app, err := tsp.Prepare(model, tsp.EncodeOptions{})
	requireOK(t, err)
	params := qaoa.Parameters{Gammas: []float64{.4}, Betas: []float64{.3}}
	fromGo, err := app.QAOA().Build(params, qaoa.BuildOptions{Measure: true})
	requireOK(t, err)
	wantState, wantMeasured, err := qasmith.Statevector(ctx, fromGo)
	requireOK(t, err)
	wantExact, err := qaoa.EvaluateExact(ctx, app.QAOA(), params)
	requireOK(t, err)
	wantSample, err := app.Run(ctx, params, tsp.RunOptions{Shots: 1000, Seed: 42, Reference: tsp.ReferenceOptions{Enabled: true}})
	requireOK(t, err)
	compiler := qasmith.Compiler{}
	for _, path := range []string{"../../tsp.smith", "../../program-imported.smith", "../../examples/optimization/language/imported.smith"} {
		t.Run(path, func(t *testing.T) {
			requireOK(t, compiler.Check(path))
			p, err := compiler.Prepare(path)
			requireOK(t, err)
			recipe, ok := p.Plan.(plan.Plan)
			if !ok || !reflect.DeepEqual(recipe.Parameters(), params) || recipe.Variables() != 9 || recipe.Layers() != 1 {
				t.Fatalf("unexpected plan %+v", p.Plan)
			}
			expectedModel, err := json.Marshal(app.QUBO())
			requireOK(t, err)
			model, err := json.Marshal(recipe.Problem().QUBO())
			requireOK(t, err)
			if string(model) != string(expectedModel) {
				t.Fatalf("QUBO changed: %s", model)
			}
			wantIsing, err := json.Marshal(app.QAOA().Ising())
			requireOK(t, err)
			gotIsing, err := json.Marshal(recipe.Problem().QAOA().Ising())
			requireOK(t, err)
			if string(gotIsing) != string(wantIsing) {
				t.Fatalf("Ising changed: %s", gotIsing)
			}
			layout, err := p.Circuit.Validate()
			requireOK(t, err)
			if layout != (ir.Layout{Qubits: 9, Bits: 9}) || !reflect.DeepEqual(normalized(fromGo), normalized(p.Circuit)) {
				t.Fatal("Smith and Go circuit IR differ")
			}
			if p.Origin.File == "" || p.Origin.Line == 0 || p.Circuit.Operations[0].(ir.AllocateQubit).Origin.Line != p.Origin.Line {
				t.Fatalf("missing source origin %+v", p.Origin)
			}
			got, measured, err := qasmith.Statevector(ctx, p.Circuit)
			requireOK(t, err)
			if !reflect.DeepEqual(measured, wantMeasured) {
				t.Fatalf("measurement mismatch %v", measured)
			}
			var expected float64
			for i, amp := range got {
				if !(cmplx.Abs(amp-wantState[i]) <= 2e-12) {
					t.Fatalf("amplitude %d differs", i)
				}
				bits := make([]bool, 9)
				for j := range bits {
					bits[j] = i&(1<<j) != 0
				}
				e, err := recipe.Problem().QUBO().Evaluate(bits)
				requireOK(t, err)
				expected += cmplx.Abs(amp) * cmplx.Abs(amp) * e
			}
			if math.Abs(expected-wantExact.ExpectedEnergy) > 1e-9 {
				t.Fatalf("expected energy %g vs %g", expected, wantExact.ExpectedEnergy)
			}
			out, err := p.Execute(ctx, 1000, 42)
			requireOK(t, err)
			domain, ok := out.Optimization.Domain.(tsp.ReportMetrics)
			if !ok || domain.Classical.Distance != 80 || domain.FeasibleSamples != wantSample.FeasibleSamples ||
				!reflect.DeepEqual(out.Raw.Counts, wantSample.Raw.Counts) ||
				out.Optimization.ExpectedQUBOEnergy != wantSample.ExpectedQUBOEnergy {
				t.Fatalf("unexpected decoded output %+v", out)
			}
		})
	}
}

func customRegistry(t *testing.T, decode plan.Decoder) *extension.Registry {
	t.Helper()
	r := &extension.Registry{}
	opaque := plan.OptimizationProblem()
	requireOK(t, r.RegisterType(opaque.Descriptor()))
	fn := extension.Function{Name: "createProblem", Parameters: []extension.Parameter{{Type: extension.Int}},
		Return: opaque.Name(),
		Evaluate: func(ctx extension.StaticContext, args []extension.Value) (extension.Value, error) {
			if err := ctx.Spend(2); err != nil {
				return extension.Value{}, err
			}
			objective, err := qubo.FromMatrix([][]float64{{float64(args[0].Integer)}}, 2)
			if err != nil {
				return extension.Value{}, err
			}
			model, err := plan.FromQUBO(objective, decode)
			if err != nil {
				return extension.Value{}, err
			}
			return opaque.Pack(model)
		}}
	requireOK(t, r.RegisterModule("host:custom-optimization", fn))
	if err := r.RegisterModule("host:custom-optimization", fn); err == nil {
		t.Fatal("duplicate module accepted")
	}
	return r
}

func TestExternalModuleAndDecoderFailure(t *testing.T) {
	path := fixture(t, map[string]string{
		"main.smith": `import { create as pb } from "./outer.smith";
import { qaoa as generate } from "std/optimization/qaoa";
fn main() -> MeasuredResult { return generate(pb(1), [1.5707963267948966], [-0.7853981633974483]); }`,
		"outer.smith": `import { make } from "./inner.smith";
pub fn create(cost: Int) -> OptimizationProblem { return make(cost); }`,
		"inner.smith": `import { createProblem as makeHost } from "host:custom-optimization";
fn identity(p: OptimizationProblem) -> OptimizationProblem { return p; }
pub fn make(cost: Int) -> OptimizationProblem { return identity(makeHost(cost)); }`,
	})
	ctx := context.Background()
	compiler := qasmith.Compiler{Registry: customRegistry(t, nil)}
	requireOK(t, compiler.Check(path))
	p, err := compiler.Prepare(path)
	requireOK(t, err)
	code, err := p.Export()
	requireOK(t, err)
	if !strings.Contains(string(code), "qubit[1] q;") {
		t.Fatalf("missing custom circuit: %s", code)
	}
	out, err := p.Execute(ctx, 100, 42)
	requireOK(t, err)
	if out.Optimization.ExpectedQUBOEnergy != 2 || !reflect.DeepEqual(out.Raw.Counts, map[string]int{"0": 100}) || out.Optimization.Domain != nil {
		t.Fatalf("unexpected custom result %+v", out)
	}
	sentinel := errors.New("decoder failed")
	compiler.Registry = customRegistry(t, func(_ context.Context, raw qaoa.Evaluation) (plan.DomainResult, error) {
		delete(raw.Counts, "0")
		return nil, sentinel
	})
	p, err = compiler.Prepare(path)
	requireOK(t, err)
	out, err = p.Execute(ctx, 100, 42)
	if !errors.Is(err, sentinel) || out.Raw.Counts["0"] != 100 || !strings.Contains(err.Error(), "main.smith") {
		t.Fatalf("error did not preserve raw data: %+v, %v", out, err)
	}
}

type testCircuit struct{ invalid bool }

func (c testCircuit) Build() (*ir.Program, error) {
	if c.invalid {
		return &ir.Program{Operations: []ir.Operation{ir.Measure{Qubit: 2}}}, nil
	}
	return &ir.Program{Operations: []ir.Operation{ir.AllocateQubit{Qubit: 0}, ir.Measure{Qubit: 0}}}, nil
}
func TestGenericProducerAndCheck(t *testing.T) {
	const source = `import { prepare } from "host:test-plan";
fn main() -> MeasuredResult { return prepare(); }`
	path := fixture(t, map[string]string{"main.smith": source})
	for _, bad := range []bool{false, true} {
		r := &extension.Registry{}
		called := 0
		requireOK(t, r.RegisterModule("host:test-plan", extension.Function{Name: "prepare", Return: extension.MeasuredResult,
			Validate: func(extension.StaticContext, []extension.Value) error { called++; return nil },
			Produce: func(extension.StaticContext, []extension.Value) (extension.CircuitPlan, error) {
				called++
				return testCircuit{invalid: bad}, nil
			}}))
		compiler := qasmith.Compiler{Registry: r}
		requireOK(t, compiler.Check(path))
		if called != 0 {
			t.Fatal("check invoked host callbacks")
		}
		p, err := compiler.Prepare(path)
		if bad {
			if err == nil || !strings.Contains(err.Error(), "invalid measurement") {
				t.Fatalf("accepted invalid producer IR: %v", err)
			}
		} else {
			requireOK(t, err)
			if _, err := p.Optimize(context.Background(), qaoa.OptimizeOptions{}, false); err == nil || !strings.Contains(err.Error(), "no supported variational") {
				t.Fatalf("unexpected error %v", err)
			}
		}
		if called != 2 {
			t.Fatalf("callbacks %d", called)
		}
	}
}

func TestSmithOptimizationDiagnostics(t *testing.T) {
	const imports = `import { tsp } from "std/optimization/tsp";
import { qaoa } from "std/optimization/qaoa";
`
	const body = `let p = tsp([[0, 1, 2], [1, 0, 3], [2, 3, 0]], 0);`
	for _, tc := range []struct {
		name, source, want string
		compile            bool
		limits             qasmith.Limits
	}{
		{"module", `import { tsp } from "std/optimization/missing"; fn main() -> Int { return 0; }`, "unregistered standard-library", false, qasmith.Limits{}},
		{"export", `import { wrong } from "std/optimization/tsp"; fn main() -> Int { return 0; }`, "no exported symbol", false, qasmith.Limits{}},
		{"wrong type", imports + `fn main() -> MeasuredResult { return qaoa(1, [0.4], [0.3]); }`, "expects OptimizationProblem", false, qasmith.Limits{}},
		{"scalar conversion", imports + `fn main() -> Int { return tsp([[0,1,2],[1,0,3],[2,3,0]],0); }`, "expected return type Int", false, qasmith.Limits{}},
		{"shape", imports + `fn main() -> MeasuredResult { let p = tsp([[0,1],[1,0],[2,3]],0); return qaoa(p,[1],[1]); }`, "square distance matrix", false, qasmith.Limits{}},
		{"negative", imports + `fn main() -> MeasuredResult { let p = tsp([[0,-1,2],[1,0,3],[2,3,0]],0); return qaoa(p,[1],[1]); }`, "distances[0][1]", true, qasmith.Limits{}},
		{"precision", imports + `fn main() -> MeasuredResult { let p = tsp([[0,9007199254740993,2],[1,0,3],[2,3,0]],0); return qaoa(p,[1],[1]); }`, "cannot be represented exactly", true, qasmith.Limits{}},
		{"start", imports + `fn main() -> MeasuredResult { let p = tsp([[0,1,2],[1,0,3],[2,3,0]],9); return qaoa(p,[1],[1]); }`, "starting city 9", true, qasmith.Limits{}},
		{"parameter lengths", imports + `fn main() -> MeasuredResult { ` + body + ` return qaoa(p,[1,2],[1]); }`, "equal, nonzero lengths", false, qasmith.Limits{}},
		{"empty", imports + `fn main() -> MeasuredResult { ` + body + ` return qaoa(p,[],[]); }`, "empty array", false, qasmith.Limits{}},
		{"non-finite", imports + `fn main() -> MeasuredResult { ` + body + ` return qaoa(p,[1e308*1e308],[1]); }`, "non-finite Float", true, qasmith.Limits{}},
		{"let producer", imports + `fn main() -> MeasuredResult { ` + body + ` let a = qaoa(p,[1],[1]); return qaoa(p,[1],[1]); }`, "final returned expression", false, qasmith.Limits{}},
		{"mixed effects", imports + `fn hGate(q: Qubit) -> Void { h(q); } fn main() -> MeasuredResult { ` + body + ` let q = qubit(); hGate(q); return qaoa(p,[1],[1]); }`, "independent quantum operations", false, qasmith.Limits{}},
		{"matrix budget", imports + `fn main() -> MeasuredResult { ` + body + ` return qaoa(p,[1],[1]); }`, "matrix element limit", false, qasmith.Limits{MaxMatrixElements: 8}},
		{"model budget", imports + `fn main() -> MeasuredResult { ` + body + ` return qaoa(p,[1],[1]); }`, "variable limit", true, qasmith.Limits{MaxModelVariables: 3}},
		{"coefficient budget", imports + `fn main() -> MeasuredResult { ` + body + ` return qaoa(p,[1],[1]); }`, "coefficient limit", true, qasmith.Limits{MaxModelCoefficients: 8}},
		{"work budget", imports + `fn main() -> MeasuredResult { ` + body + ` return qaoa(p,[1],[1]); }`, "structured evaluation work", true, qasmith.Limits{MaxStructuredWork: 1}},
		{"layer budget", imports + `fn main() -> MeasuredResult { ` + body + ` return qaoa(p,[1,2],[1,2]); }`, "layers limit", false, qasmith.Limits{MaxCircuitLayers: 1}},
		{"gate budget", imports + `fn main() -> MeasuredResult { ` + body + ` return qaoa(p,[1],[1]); }`, "gates limit", true, qasmith.Limits{MaxGates: 2}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := fixture(t, map[string]string{"main.smith": tc.source})
			compiler := qasmith.Compiler{Limits: tc.limits}
			var err error
			if tc.compile {
				_, err = compiler.Compile(path)
			} else {
				err = compiler.Check(path)
			}
			var detail *qasmith.Diagnostic
			if !errors.As(err, &detail) || detail.Position.Line == 0 || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected source-aware %q, got %v", tc.want, err)
			}
		})
	}
}

func TestStructuredIdentityAndSourceTrace(t *testing.T) {
	authentic := plan.OptimizationProblem()
	foreign, err := extension.NewOpaqueType(authentic.Name(), func(v int) (int, error) { return v, nil })
	requireOK(t, err)
	r := &extension.Registry{}
	requireOK(t, r.RegisterType(authentic.Descriptor()))
	requireOK(t, r.RegisterModule("host:foreign", extension.Function{Name: "foreign", Return: authentic.Name(),
		Evaluate: func(extension.StaticContext, []extension.Value) (extension.Value, error) { return foreign.Pack(1) }}))
	path := fixture(t, map[string]string{
		"main.smith": `import { helper } from "./library.smith";
import { qaoa } from "std/optimization/qaoa";
fn main() -> MeasuredResult { return qaoa(helper(),[1],[1]); }`,
		"library.smith": `import { foreign } from "host:foreign";
pub fn helper() -> OptimizationProblem { return foreign(); }`,
	})
	_, err = (qasmith.Compiler{Registry: r}).Compile(path)
	if err == nil || !strings.Contains(err.Error(), "invalid or foreign") || !strings.Contains(err.Error(), "library.smith") ||
		!strings.Contains(err.Error(), "called from ") || !strings.Contains(err.Error(), "main.smith") {
		t.Fatalf("unexpected error %v", err)
	}
}

func TestLargeAndLocallyPackagedSmithOptimization(t *testing.T) {
	path := fixture(t, map[string]string{
		"qasmith.json": `{"name":"main","version":"0.1.0","dependencies":{"travel":{"path":"./packages/travel"}}}`,
		"main.smith": `import { make } from "travel/model";
import { qaoa } from "std/optimization/qaoa";
fn main() -> MeasuredResult { return qaoa(make(), [0.4,0.2], [0.3,0.6]); }`,
		"packages/travel/model.smith": `import { tsp } from "std/optimization/tsp";
pub fn make() -> OptimizationProblem {
    let row = [0.0,0.0,0.0,0.0,0.0,0.0];
    return tsp([row,row,row,row,row,row], 5);
}`,
	})
	c := qasmith.Compiler{}
	requireOK(t, c.Check(path))
	p, err := c.Prepare(path)
	requireOK(t, err)
	code, err := p.Export()
	requireOK(t, err)
	if !strings.Contains(string(code), "qubit[25] q;") {
		t.Fatal("large circuit did not compile")
	}
	if _, err := p.Execute(context.Background(), 1, 42); err == nil || !strings.Contains(err.Error(), "simulator limit") {
		t.Fatalf("wrong runtime limit: %v", err)
	}
}

func TestSmithNoFeasibleSamplePreservesRaw(t *testing.T) {
	p, err := (qasmith.Compiler{}).Prepare("../../tsp.smith")
	requireOK(t, err)
	out, err := p.Execute(context.Background(), 1, 42)
	requireOK(t, err)
	report := out.Optimization.Domain.(tsp.ReportMetrics)
	if report.BestFeasible != nil || report.InfeasibleSamples != 1 || len(out.Raw.Counts) != 1 || len(out.Optimization.Observations) != 1 {
		t.Fatalf("unexpected result %+v", out)
	}
}
