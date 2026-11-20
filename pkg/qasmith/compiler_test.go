package qasmith_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/quvly/Qasmith/pkg/ir"
	"github.com/quvly/Qasmith/pkg/qasmith"
)

func fixture(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for path, source := range files {
		path = filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(source), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return filepath.Join(root, "main.smith")
}

func TestImportedClassicalFunctions(t *testing.T) {
	for _, imports := range []string{
		`import { total as sum, adjustment } from "./math/operations.smith"; import { constant } from "./constant.smith";`,
		`import { constant } from "./constant.smith"; import { adjustment, total as sum } from "./math/operations.smith";`,
	} {
		path := fixture(t, map[string]string{
			"main.smith": imports + `fn main() -> Int { let b = 5; return sum(constant(), b) * adjustment(); }`,
			"math/operations.smith": `import { twice } from "./helper.smith";
				fn add(a: Int, b: Int) -> Int { return a + b; }
				pub fn total(a: Int, b: Int) -> Int { return add(a,b); }
				pub fn adjustment() -> Int { let b = 1; return twice(b); }`,
			"math/helper.smith": `pub fn twice(a: Int) -> Int { return a * 2; }`,
			"constant.smith":    `pub fn constant() -> Int { return 3; }`,
		})
		value, err := (qasmith.Compiler{}).Eval(path)
		if err != nil || value != 16 {
			t.Fatalf("eval = %d, %v", value, err)
		}
	}
}

func TestSemanticDiagnostics(t *testing.T) {
	for _, tc := range []struct{ name, source, lib, code, message string }{
		{"private", `import { hidden } from "./lib.smith"; fn main() -> Int { return 1; }`, `fn hidden() -> Int { return 2; }`, "E2001", "private"},
		{"missing symbol", `import { absent } from "./lib.smith"; fn main() -> Int { return 1; }`, `pub fn foo() -> Int { return 2; }`, "E2001", "no exported symbol"},
		{"duplicate import", `import { foo, foo } from "./lib.smith"; fn main() -> Int { return 1; }`, `pub fn foo() -> Int { return 2; }`, "E2003", "conflicts"},
		{"shadow import", `import { foo } from "./lib.smith"; fn main() -> Int { let foo = 1; return foo; }`, `pub fn foo() -> Int { return 2; }`, "E3001", "conflicts"},
		{"declare import", `import { foo } from "./lib.smith"; fn foo() -> Int { return 1; } fn main() -> Int { return 1; }`, `pub fn foo() -> Int { return 2; }`, "E2003", "conflicts"},
		{"import not reexported", `import { foo } from "./lib.smith"; fn main() -> Int { return 1; }`, `import { foo } from "./other.smith";`, "E2001", "no exported symbol"},
		{"imported signature", `import { foo } from "./lib.smith"; fn main() -> Int { return foo(1); }`, `pub fn foo() -> Int { return 2; }`, "E3004", "expects 0 arguments"},
		{"unused module body", `import { foo } from "./lib.smith"; fn main() -> Int { return 1; }`, `pub fn foo() -> Int { return nope; }`, "E3003", "undefined variable"},
		{"unknown function", `fn main() -> Int { return unknown(); }`, "", "E3003", "undefined function"},
		{"local scope", `fn f() -> Int { let hidden = 7; return hidden; } fn main() -> Int { return hidden; }`, "", "E3003", "undefined variable"},
		{"duplicate declarations", `fn main() -> Int { return 1; } fn main() -> Int { return 2; }`, "", "E3001", "duplicate"},
		{"duplicate parameter", `fn f(a: Int, a: Int) -> Int { return a; } fn main() -> Int { return 1; }`, "", "E3001", "local name"},
		{"assignment", `fn main() -> Int { let a = 1; a = 2; return a; }`, "", "E1002", "expected ;"},
		{"argument type", `fn f(a: Int) -> Void {} fn main() -> MeasuredResult { let q = qubit(); f(q); return measure(q); }`, "", "E3004", "expects Int"},
		{"bad gate", `fn main() -> MeasuredResult { let q = qubit(); x(1); return measure(q); }`, "", "E3004", "expects Qubit"},
		{"bad return", `fn main() -> MeasuredResult { return 1; }`, "", "E3002", "expected return type"},
		{"missing return", `fn main() -> Int {}`, "", "E3002", "missing return"},
		{"direct recursion", `fn main() -> Int { return main(); }`, "", "E3005", "unsupported recursion"},
		{"indirect recursion", `fn a() -> Int { return b(); } fn b() -> Int { return a(); } fn main() -> Int { return 1; }`, "", "E3005", "unsupported recursion"},
		{"helper allocation", `fn helper() -> Void { let q = qubit(); } fn main() -> Int { return 1; }`, "", "E3100", "let binding in the quantum entry point"},
		{"helper measurement", `fn helper(q: Qubit) -> Void { measure(q); } fn main() -> Int { return 1; }`, "", "E3100", "final returned expression"},
		{"qubit copy", `fn main() -> MeasuredResult { let q = qubit(); let copy = q; return measure(copy); }`, "", "E3100", "new qubit"},
		{"double measurement", `fn main() -> MeasuredResult { let q = qubit(); return measure(q, q); }`, "", "E3100", "distinct qubit"},
		{"after measurement", `fn main() -> MeasuredResult { let q = qubit(); return measure(q); h(q); }`, "", "E3002", "unreachable"},
		{"duplicate gate operands", `fn main() -> MeasuredResult { let q = qubit(); cx(q, q); return measure(q); }`, "", "E3100", "distinct qubit"},
		{"transitive alias", `import { wrap } from "./lib.smith"; fn main() -> MeasuredResult { let q = qubit(); wrap(q, q); return measure(q); }`, `fn entangle(a: Qubit,b: Qubit) -> Void { cx(a,b); } pub fn wrap(a: Qubit,b: Qubit) -> Void { entangle(b,a); }`, "E3100", "distinct qubit"},
		{"pure numeric", `fn helper() -> Void {} fn main() -> Int { helper(); return 1; }`, "", "E3100", "pure numeric"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := fixture(t, map[string]string{"main.smith": tc.source, "lib.smith": tc.lib, "other.smith": `pub fn foo() -> Int { return 1; }`})
			err := (qasmith.Compiler{}).Check(path)
			var detail *qasmith.Diagnostic
			if !errors.As(err, &detail) || detail.Code != tc.code || detail.Position.Line == 0 || !strings.Contains(detail.Message, tc.message) {
				t.Fatalf("wanted %s with %q, got %v", tc.code, tc.message, err)
			}
		})
	}
}

func TestClassicalArithmetic(t *testing.T) {
	tests := []struct {
		expr    string
		want    int64
		problem string
	}{
		{"3 + 5", 8, ""}, {"(3 + 5) * 2 - 1", 15, ""}, {"9 - 4 - 1", 4, ""},
		{"-11 % 4", -3, ""}, {"-9223372036854775808", -9223372036854775808, ""},
		{"1 / (4 - 4)", 0, "division by zero"}, {"1 % 0", 0, "division by zero"},
		{"9223372036854775807 + 1", 0, "overflow"}, {"-9223372036854775808 - 1", 0, "overflow"},
		{"3037000500 * 3037000500", 0, "overflow"}, {"-9223372036854775808 / -1", 0, "overflow"},
	}
	compiler := qasmith.Compiler{}
	for _, tc := range tests {
		t.Run(tc.expr, func(t *testing.T) {
			path := fixture(t, map[string]string{"main.smith": fmt.Sprintf("fn main() -> Int { return %s; }", tc.expr)})
			if err := compiler.Check(path); err != nil {
				t.Fatalf("check evaluated arithmetic: %v", err)
			}
			got, err := compiler.Eval(path)
			if tc.problem != "" {
				if err == nil || !strings.Contains(err.Error(), tc.problem) {
					t.Fatalf("expected %q, got %v", tc.problem, err)
				}
			} else if err != nil || got != tc.want {
				t.Fatalf("eval = %d, %v", got, err)
			}
		})
	}
}

func TestNestedQuantumBorrows(t *testing.T) {
	path := fixture(t, map[string]string{
		"main.smith":      `import { wrap } from "./lib.smith"; fn main() -> MeasuredResult { let q = qubit(); let r = qubit(); wrap(r, q); return measure(q,r); }`,
		"lib.smith":       `import { flip } from "./deep/flip.smith"; pub fn wrap(a: Qubit, b: Qubit) -> Void { flip(a); cx(a,b); }`,
		"deep/flip.smith": `pub fn flip(a: Qubit) -> Void { x(a); }`,
	})
	compiler := qasmith.Compiler{}
	circuit, err := compiler.Lower(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(circuit.Operations) != 6 {
		t.Fatalf("unexpected circuit %v", circuit.Operations)
	}
	x := circuit.Operations[2].(ir.ApplyGate)
	cx := circuit.Operations[3].(ir.ApplyGate)
	if x.Gate != ir.X || !reflect.DeepEqual(x.Qubits, []ir.QubitID{1}) || !reflect.DeepEqual(cx.Qubits, []ir.QubitID{1, 0}) {
		t.Fatalf("unexpected gate operands: %+v, %+v", x, cx)
	}
	if filepath.Base(x.Origin.File) != "flip.smith" || len(x.Origin.CallSites) != 2 || filepath.Base(x.Origin.CallSites[0].File) != "lib.smith" || filepath.Base(x.Origin.CallSites[1].File) != "main.smith" {
		t.Fatalf("unexpected origin %+v", x.Origin)
	}
	result, err := compiler.Run(path, 100, 7)
	if err != nil || !reflect.DeepEqual(result.Counts, map[string]int{"11": 100}) {
		t.Fatalf("simulation: %+v, %v", result, err)
	}
}
