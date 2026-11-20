package qasmith_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/quvly/Qasmith/pkg/extension"
	"github.com/quvly/Qasmith/pkg/ir"
	"github.com/quvly/Qasmith/pkg/qasmith"
)

func TestStaticClassicalProgram(t *testing.T) {
	path := fixture(t, map[string]string{
		"main.smith": `import { choose } from "./lib.smith";
fn main() -> Int {
    let matrix = [[10, 20], [30, 40], [50, 60]];
    let copy = matrix;
    let allowed = 1 + 2 * 3 >= 7 && 2 < 2.5 && !false || 1 / 0 == 5;
    let blocked = false && 1 / 0 == 5;
    for i in 2..2 { return 0; }
    if allowed != blocked && true == true {
        return choose(copy, len(matrix) - 1);
    } else {
        return 0;
    }
}`,
		"lib.smith": `pub fn choose(values: Int[2][3], index: Int) -> Int {
    let row = values[index];
    return row[len(row) - 1];
}`,
	})
	compiler := qasmith.Compiler{}
	if err := compiler.Check(path); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if got, err := compiler.Eval(path); err != nil || got != 60 {
			t.Fatalf("eval = %d, %v", got, err)
		}
	}
}

func TestRegisterMappingAndMeasurementOrder(t *testing.T) {
	path := fixture(t, map[string]string{"main.smith": `fn size() -> Int { return 1 + 2; }
fn main() -> MeasuredResult {
    let x0 = qubit();
    let q = qreg(size());
    let r = qreg(2);
    for i in 0..len(q) {
        for j in i..len(q) {
            if i < j && j <= 2 {
                cx(q[i], q[j]);
            } else {
                x(r[i % len(r)]);
            }
        }
    }
    x(q[2]);
    return measure(q[2], x0, r[1]);
}`})
	c := qasmith.Compiler{}
	if err := c.Check(path); err != nil {
		t.Fatal(err)
	}
	p, err := c.Lower(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := p.Validate(); err != nil || got != (ir.Layout{Qubits: 6, Bits: 3}) {
		t.Fatalf("layout %+v, %v", got, err)
	}
	gates, measurements := 0, 0
	var controlled [][2]ir.QubitID
	for _, op := range p.Operations {
		switch op := op.(type) {
		case ir.ApplyGate:
			gates++
			if op.Gate == ir.CX {
				controlled = append(controlled, [2]ir.QubitID{op.Qubits[0], op.Qubits[1]})
			}
		case ir.Measure:
			if op.Qubit != []ir.QubitID{3, 0, 5}[measurements] || int(op.Bit) != measurements {
				t.Fatalf("measurement %+v", op)
			}
			measurements++
		}
	}
	if gates != 7 || measurements != 3 || !reflect.DeepEqual(controlled, [][2]ir.QubitID{{1, 2}, {1, 3}, {2, 3}}) {
		t.Fatalf("gates=%d measurements=%d CX=%v", gates, measurements, controlled)
	}
	result, err := c.Run(path, 40, 17)
	if err != nil || !reflect.DeepEqual(result.Counts, map[string]int{"101": 40}) {
		t.Fatalf("sampled %+v, %v", result, err)
	}
}

func TestStaticDiagnostics(t *testing.T) {
	for _, tc := range []struct{ name, body, want string }{
		{"zero register", "let q = qreg(0); return measure(q);", "positive Int"},
		{"negative register", "let q = qreg(3-4); return measure(q);", "positive Int"},
		{"register float", "let q = qreg(2.0); return measure(q);", "expects Int"},
		{"register index", "let q = qreg(2); let i = len(q); x(q[i]); return measure(q);", "index 2 out of bounds"},
		{"negative index", "let q = qreg(2); x(q[-1]); return measure(q);", "index -1 out of bounds"},
		{"array index", "let q = qubit(); let a = [1, 2]; x(a[2]); return measure(q);", "index 2 out of bounds"},
		{"index type", "let q = qreg(2); x(q[0.3]); return measure(q);", "index requires Int"},
		{"invalid indexing", "let q = qubit(); x(q[0]); return measure(q);", "cannot index Qubit"},
		{"ragged", "let q = qubit(); let a = [[1], [2, 3]]; return measure(q);", "consistent dimensions"},
		{"mixed", "let q = qubit(); let a = [1, 2.5]; return measure(q);", "one classical type"},
		{"qubit array", "let q = qubit(); let a = [q]; return measure(q);", "one classical type"},
		{"array size", "let q = qreg(2); layer([1.0]); return measure(q);", "expects Float[2]"},
		{"register size", "let q = qreg(1+2); prepare(q); return measure(q);", "expects QReg<2>"},
		{"nested invalid type", "let q = qreg(2); if false { h(1); } return measure(q);", "expects Qubit"},
		{"bool", "let q = qreg(2); if true && 0 { h(q[0]); } return measure(q);", "requires Bool"},
		{"condition", "let q = qreg(2); if q[0] { h(q[1]); } return measure(q);", "condition requires Bool"},
		{"dynamic", "let q = qreg(2); if measure(q) == true { h(q[1]); } return measure(q);", "final returned"},
		{"duplicate CX", "let q = qreg(2); for i in 0..2 { cx(q[i], q[i]); } return measure(q);", "distinct qubit"},
		{"duplicate helper", "let q = qreg(2); for i in 0..2 { pair(q[i], q[i]); } return measure(q);", "distinct qubit"},
		{"aliased borrow", "let q = qreg(2); join(q, q); return measure(q);", "distinct qubit"},
		{"duplicate measure", "let q = qreg(2); return measure(q[0], q[1-1]);", "distinct qubit"},
		{"mixed measure", "let q = qreg(2); return measure(q, q[0]);", "single register"},
		{"element copy", "let q = qreg(2); let a = q[0]; return measure(q);", "pass borrowed elements directly"},
		{"register copy", "let q = qreg(2); let a = q; return measure(q);", "new qubit or register"},
		{"loop float", "let q = qreg(2); for i in 0..2.0 { h(q[i]); } return measure(q);", "loop bound requires Int"},
		{"descending", "let q = qreg(2); for i in 2..1 { h(q[i]); } return measure(q);", "descending loop ranges"},
		{"loop assignment", "let q = qreg(2); for i in 0..2 { i = 0; } return measure(q);", "expected ;"},
		{"loop scope", "let q = qreg(2); for i in 0..2 { h(q[i]); } x(q[i]); return measure(q);", "undefined variable"},
		{"dynamic failure", "let q = qreg(2); for i in 0..(1/0) { h(q[i]); } return measure(q);", "division by zero"},
		{"helper allocation", "let q = qreg(2); bad(); return measure(q);", "let binding in the quantum entry point"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			helper := `fn pair(a: Qubit, b: Qubit) -> Void { cx(a, b); }
fn join(a: QReg<2>, b: QReg<2>) -> Void { cx(a[0], b[0]); }
fn prepare(q: QReg<2>) -> Void { h(q[1]); }
fn layer(a: Float[2]) -> Void {}
`
			if tc.name == "helper allocation" {
				helper += "fn bad() -> Void { let r = qreg(1); }\n"
			}
			path := fixture(t, map[string]string{"main.smith": helper + "fn main() -> MeasuredResult {\n" + tc.body + "\n}"})
			err := (qasmith.Compiler{}).Check(path)
			var detail *qasmith.Diagnostic
			if !errors.As(err, &detail) || detail.Position.Line == 0 || !strings.Contains(detail.Message, tc.want) {
				t.Fatalf("want %q, got %v", tc.want, err)
			}
		})
	}
}

func TestCompilerLimits(t *testing.T) {
	for _, tc := range []struct {
		name, source, want string
		limits             qasmith.Limits
	}{
		{"loop", "let q = qreg(2); for i in 0..2 { for j in 0..2 { h(q[i]); } } return measure(q);", "loop expansion", qasmith.Limits{MaxLoopIterations: 5}},
		{"huge loop", "let q = qreg(2); for i in -9223372036854775808..9223372036854775807 {} return measure(q);", "loop expansion", qasmith.Limits{}},
		{"gates", "let q = qreg(2); h(q[0]); h(q[1]); return measure(q);", "generated gates", qasmith.Limits{MaxGates: 1}},
		{"qubits", "let q = qreg(2); let a = qubit(); return measure(q);", "allocated qubits", qasmith.Limits{MaxQubits: 2}},
		{"arrays", "let q = qreg(2); for i in 0..2 { let a = [1,2]; } return measure(q);", "array elements", qasmith.Limits{MaxArrayElements: 3}},
		{"depth", "let q = qreg(2); for i in 0..2 { if true { h(q[i]); } } return measure(q);", "evaluation depth", qasmith.Limits{MaxDepth: 5}},
		{"steps", "let q = qreg(2); for i in 0..2 { h(q[i]); } return measure(q);", "evaluation steps", qasmith.Limits{MaxSteps: 6}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := fixture(t, map[string]string{"main.smith": "fn main() -> MeasuredResult {" + tc.source + "}"})
			c := qasmith.Compiler{Limits: tc.limits}
			ir, err := c.Lower(p)
			var detail *qasmith.Diagnostic
			if ir != nil || !errors.As(err, &detail) || detail.Code != "E4100" || !strings.Contains(detail.Message, tc.want) {
				t.Fatalf("wanted %q without partial circuit, got %v, %v", tc.want, ir, err)
			}
			if err := c.Check(p); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("check did not enforce limit: %v", err)
			}
		})
	}
}

func TestLargeRegisterCompilesButDoesNotSimulate(t *testing.T) {
	path := fixture(t, map[string]string{"main.smith": "fn main() -> MeasuredResult { let q = qreg(17); return measure(q); }"})
	c := qasmith.Compiler{}
	out, err := c.Compile(path)
	if err != nil || !strings.Contains(string(out), "qubit[17] q;") {
		t.Fatalf("compile: %s, %v", out, err)
	}
	if _, err := c.Run(path, 1, 17); err == nil || !strings.Contains(err.Error(), "simulator limit is 16") {
		t.Fatalf("run: %v", err)
	}
}

func TestHostArrayIsolationAndCheck(t *testing.T) {
	registry := &extension.Registry{}
	arrayType := extension.ArrayType(extension.Int, 2)
	calls := 0
	if err := registry.Register(extension.Intrinsic{
		Namespace: "test", Name: "if",
		Signature: extension.Signature{Parameters: []extension.Type{arrayType}, Return: extension.Bool},
		Validate: func(args []extension.Value) error {
			calls++
			args[0].Array.Elements[0].Integer = 99
			return nil
		},
		Lower: func(_ extension.LoweringContext, args []extension.Value) (extension.Value, error) {
			calls++
			ok := args[0].Array.Elements[0].Integer == 1
			args[0].Array.Elements[1].Integer = 50
			return extension.Value{Type: extension.Bool, Bool: ok}, nil
		},
	}); err != nil {
		t.Fatal(err)
	}
	path := fixture(t, map[string]string{"main.smith": `import { if as test } from "host:test";
fn main() -> Int {
    let a = [1, 2];
    if test(a) { return a[1]; } else { return -1; }
}`})
	c := qasmith.Compiler{Registry: registry}
	if err := c.Check(path); err != nil || calls != 0 {
		t.Fatalf("check invoked callbacks or failed: %d, %v", calls, err)
	}
	if result, err := c.Eval(path); err != nil || calls != 2 || result != 2 {
		t.Fatalf("eval=%d calls=%d err=%v", result, calls, err)
	}
}

func TestHostArrayResultsAndGateBudget(t *testing.T) {
	arrayType := extension.ArrayType(extension.Float, 2)
	for _, mode := range []string{"valid", "wrong shape", "wrong element", "gate budget"} {
		t.Run(mode, func(t *testing.T) {
			r := &extension.Registry{}
			if err := r.Register(extension.Intrinsic{
				Namespace: "test", Name: "array",
				Signature: extension.Signature{Return: arrayType},
				Lower: func(extension.LoweringContext, []extension.Value) (extension.Value, error) {
					elements := []extension.Value{{Type: extension.Float, Float: .2}, {Type: extension.Float, Float: .4}}
					if mode == "wrong shape" {
						elements = elements[:1]
					}
					if mode == "wrong element" {
						elements[0].Type = extension.Qubit
					}
					return extension.Value{Type: arrayType, Array: &extension.ArrayValue{Elements: elements}}, nil
				},
			}); err != nil {
				t.Fatal(err)
			}
			if err := r.Register(flipTwice()); err != nil {
				t.Fatal(err)
			}
			path := fixture(t, map[string]string{"main.smith": `import { array } from "host:test";
import { flipTwice } from "host:testgates";
fn main() -> MeasuredResult {
    let a = array(); let q = qreg(2); rz(a[1], q[0]); flipTwice(q[1]); return measure(q);
}`})
			c := qasmith.Compiler{Registry: r}
			if mode == "gate budget" {
				c.Limits.MaxGates = 2
			}
			circuit, err := c.Lower(path)
			switch mode {
			case "valid":
				if err != nil || circuit.Operations[2].(ir.ApplyGate).Parameters[0] != .4 {
					t.Fatalf("%v, %v", circuit, err)
				}
			case "wrong shape", "wrong element":
				var d *qasmith.Diagnostic
				if circuit != nil || !errors.As(err, &d) || d.Code != "E4001" {
					t.Fatalf("%v, %v", circuit, err)
				}
			case "gate budget":
				var d *qasmith.Diagnostic
				if circuit != nil || !errors.As(err, &d) || d.Code != "E4100" {
					t.Fatalf("%v, %v", circuit, err)
				}
			}
		})
	}
	registry := &extension.Registry{}
	for _, typ := range []extension.Type{"QReg<2>", "Qubit[2]"} {
		err := registry.Register(extension.Intrinsic{
			Namespace: "test", Name: "unsupported",
			Signature: extension.Signature{Parameters: []extension.Type{typ}, Return: extension.Void},
			Lower: func(extension.LoweringContext, []extension.Value) (extension.Value, error) {
				return extension.Value{Type: extension.Void}, nil
			},
		})
		if err == nil {
			t.Fatalf("registry accepted resource type %s", typ)
		}
	}
}

func TestNewBuiltinNamesRemainAvailableToExistingFunctions(t *testing.T) {
	path := fixture(t, map[string]string{"main.smith": `
fn len() -> Int { return 3; }
fn qreg() -> Int { return 5; }
fn main() -> Int { return len() + qreg(); }`})
	if result, err := (qasmith.Compiler{}).Eval(path); err != nil || result != 8 {
		t.Fatalf("eval=%d, %v", result, err)
	}
}
