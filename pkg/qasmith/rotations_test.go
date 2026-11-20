package qasmith_test

import (
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/quvly/Qasmith/pkg/extension"
	"github.com/quvly/Qasmith/pkg/ir"
	"github.com/quvly/Qasmith/pkg/qasmith"
)

func TestFloatFunctionsAndConversions(t *testing.T) {
	source := fixture(t, map[string]string{
		"main.smith": `import { half, turn, one } from "./angles.smith";
			fn main() -> MeasuredResult {
				let q = qubit();
				let offset = -(0.5 + -2.75) * 2e2 / 1.0E+2;
				rx(half(3), q);
				ry(turn() + offset - 1.0e-3, q);
				rz(one(), q);
				rx(3 / 2, q);
				return measure(q);
			}`,
		"angles.smith": `pub fn half(value: Float) -> Float { return value / 2; }
			pub fn turn() -> Float { return pi; }
			pub fn one() -> Float { return 1; }`,
	})
	program, err := (qasmith.Compiler{}).Lower(source)
	if err != nil {
		t.Fatal(err)
	}
	pi := float64(math.Pi)
	for i, want := range []float64{1.5, float64(pi+4.5) - .001, 1, 1} {
		got := program.Operations[i+1].(ir.ApplyGate)
		if len(got.Parameters) != 1 || got.Parameters[0] != want || got.Qubits[0] != 0 || got.Origin.Line != i+5 {
			t.Errorf("gate %d: %+v, want %v", i, got, want)
		}
	}
}

func TestFloatDiagnostics(t *testing.T) {
	for _, tc := range []struct {
		expression, problem string
		evaluate            bool
	}{
		{`rx("hello", q)`, "expected expression", false},
		{"rx(q, 3.14)", "expects Float", false},
		{"rx(3.14)", "expects 2 arguments", false},
		{"rx(3.14, 5)", "expects Qubit", false},
		{"rx(1.0 % 2, q)", "requires Int", false},
		{"rx(-q, q)", "requires a numeric operand", false},
		{"rx(1.0e-, q)", "exponent requires digits", false},
		{"rx(1.2.3, q)", "malformed numeric literal", false},
		{"rx(1e999, q)", "invalid Float literal", false},
		{"rx(1e308 * 1e308, q)", "non-finite Float", true},
		{"ry(1 / -0.0, q)", "division by zero", true},
		{"rz(0.0 / 0.0, q)", "division by zero", true},
		{"rx(-1e308 * 1e308, q)", "non-finite Float", true},
		{"rz(-(-9223372036854775808), q)", "integer overflow", true},
	} {
		t.Run(tc.expression, func(t *testing.T) {
			path := fixture(t, map[string]string{"main.smith": "fn main() -> MeasuredResult {\nlet q = qubit();\n" + tc.expression + ";\nreturn measure(q);\n}"})
			compiler := qasmith.Compiler{}
			err := compiler.Check(path)
			if tc.evaluate && err == nil {
				_, err = compiler.Lower(path)
			}
			var detail *qasmith.Diagnostic
			if !errors.As(err, &detail) || detail.Position.Line != 3 || !strings.Contains(detail.Message, tc.problem) {
				t.Fatalf("want %q on line 3, got %v", tc.problem, err)
			}
		})
	}
}

func TestFloatNarrowingAndReservedPi(t *testing.T) {
	for _, source := range []string{
		`fn main() -> Int { return 1.0; }`,
		`fn f(a: Int) -> Int { return a; } fn main() -> Int { return f(1.0); }`,
		`fn main() -> Int { let pi = 4; return pi; }`,
		`fn pi() -> Int { return 4; } fn main() -> Int { return 1; }`,
		`fn f(pi: Float) -> Float { return pi; } fn main() -> Int { return 1; }`,
		`import { f as pi } from "./f.smith"; fn main() -> Int { return 1; }`,
		`fn f(q: Qubit) -> Float { return 1.0; } fn main() -> Int { return 1; }`,
	} {
		path := fixture(t, map[string]string{"main.smith": source, "f.smith": "pub fn f() -> Int { return 1; }"})
		if err := (qasmith.Compiler{}).Check(path); err == nil {
			t.Errorf("accepted %s", source)
		}
	}
}

func TestHostRotation(t *testing.T) {
	registry := &extension.Registry{}
	if err := registry.Register(extension.Intrinsic{
		Namespace: "angles", Name: "turn",
		Signature: extension.Signature{Parameters: []extension.Type{extension.Float, extension.Qubit}, Return: extension.Void},
		Lower: func(ctx extension.LoweringContext, args []extension.Value) (extension.Value, error) {
			parameterized := ctx.(extension.ParameterizedLoweringContext)
			return extension.Value{Type: extension.Void}, parameterized.ApplyParameterizedGate(ir.RX, []float64{args[0].Float}, args[1].Qubit)
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(extension.Intrinsic{
		Namespace: "angles", Name: "invalid",
		Signature: extension.Signature{Return: extension.Float},
		Lower: func(extension.LoweringContext, []extension.Value) (extension.Value, error) {
			return extension.Value{Type: extension.Float, Float: math.NaN()}, nil
		},
	}); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ angle, expected string }{{"2", "rx(2.0)"}, {"invalid()", "non-finite Float result"}} {
		path := fixture(t, map[string]string{"main.smith": `import { turn, invalid } from "host:angles";
			fn main() -> MeasuredResult { let q = qubit(); turn(` + tc.angle + `, q); return measure(q); }`})
		got, err := (qasmith.Compiler{Registry: registry}).Compile(path)
		if err != nil {
			got = []byte(err.Error())
		}
		if !strings.Contains(string(got), tc.expected) {
			t.Fatalf("wanted %q, got %s", tc.expected, got)
		}
	}
}

func TestFloatEvaluationPrecision(t *testing.T) {
	for _, tc := range []struct {
		source string
		want   float64
	}{
		{"(9007199254740993 + 0.0) - 9007199254740992.0", 0},
		{"(9007199254740993 - 9007199254740992) + 0.0", 1},
		{"-0.0", math.Copysign(0, -1)},
		{"5e-324", math.SmallestNonzeroFloat64},
		{"-5e-324 / 2.0", math.Copysign(0, -1)},
		{"1.7976931348623157e308", math.MaxFloat64},
		{"1.0000000074505806 * 0.9999999925494194 - 1.0", 0},
	} {
		t.Run(tc.source, func(t *testing.T) {
			path := fixture(t, map[string]string{"main.smith": `fn main() -> MeasuredResult {
				let q = qubit(); rz(` + tc.source + `, q); return measure(q);
			}`})
			program, err := (qasmith.Compiler{}).Lower(path)
			if err != nil {
				t.Fatal(err)
			}
			got := program.Operations[1].(ir.ApplyGate).Parameters[0]
			if math.Float64bits(got) != math.Float64bits(tc.want) {
				t.Fatalf("got %g (%016x), want %g (%016x)", got, math.Float64bits(got), tc.want, math.Float64bits(tc.want))
			}
		})
	}
}

func TestInvalidHostRotation(t *testing.T) {
	for _, tc := range []struct {
		name    string
		params  []float64
		qubit   ir.QubitID
		message string
	}{
		{"missing angle", nil, 0, "invalid operands"},
		{"NaN angle", []float64{math.NaN()}, 0, "finite parameters"},
		{"infinite angle", []float64{math.Inf(1)}, 0, "finite parameters"},
		{"foreign qubit", []float64{1}, 1, "outside intrinsic arguments"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			registry := &extension.Registry{}
			err := registry.Register(extension.Intrinsic{
				Namespace: "angles", Name: "turn",
				Signature: extension.Signature{Parameters: []extension.Type{extension.Qubit}, Return: extension.Void},
				Lower: func(ctx extension.LoweringContext, _ []extension.Value) (extension.Value, error) {
					_ = ctx.(extension.ParameterizedLoweringContext).ApplyParameterizedGate(ir.RZ, tc.params, tc.qubit)
					return extension.Value{Type: extension.Void}, nil
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			path := fixture(t, map[string]string{"main.smith": `import { turn } from "host:angles";
fn main() -> MeasuredResult {
    let a = qubit();
    let b = qubit();
    turn(a);
    return measure(a, b);
}
`})
			var detail *qasmith.Diagnostic
			program, err := (qasmith.Compiler{Registry: registry}).Lower(path)
			if program != nil || !errors.As(err, &detail) || detail.Code != "E4001" ||
				detail.Position.Line != 5 || !strings.Contains(detail.Message, tc.message) {
				t.Fatalf("got program %v and error %v, wanted %q", program, err, tc.message)
			}
		})
	}
}
