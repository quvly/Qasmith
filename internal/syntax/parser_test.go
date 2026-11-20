package syntax

import (
	"strings"
	"testing"
)

func TestTokens(t *testing.T) {
	tokens, err := lex("test.smith", "/* comment */ pub fn import from as let return _a12 Int 23 -> {}(),:;=+-*/% \"./x.smith\" // last")
	if err != nil {
		t.Fatal(err)
	}
	var kinds []string
	for _, tok := range tokens {
		kinds = append(kinds, tok.kind)
	}
	want := "pub fn import from as let return identifier identifier integer -> { } ( ) , : ; = + - * / % string end of file"
	if strings.Join(kinds, " ") != want {
		t.Fatalf("got %q", strings.Join(kinds, " "))
	}
}

func TestParseModule(t *testing.T) {
	m, err := Parse("module.smith", `// module
import { inc as increment, twice } from "./other.smith";
pub fn calculate(a: Int, b: Int) -> Int {
    let result = increment(a) + b * (2 - 1);
    return result;
}
fn hidden() -> Void { return; }`)
	if err != nil {
		t.Fatal(err)
	}
	imp := m.Imports[0]
	if imp.Path != "./other.smith" || imp.Names[0].Alias != "increment" || imp.Names[1].Name != "twice" || imp.Names[0].At.Line != 2 || imp.Names[0].At.Column != 10 {
		t.Fatalf("unexpected import: %+v", imp)
	}
	f := m.Functions[0]
	if !f.Public || len(f.Params) != 2 || f.Params[0].Type != "Int" || len(m.Functions) != 2 || m.Functions[1].Public {
		t.Fatalf("unexpected functions: %+v", m.Functions)
	}
	sum := f.Body.Statements[0].(*VariableDeclaration).Value.(*BinaryExpression)
	product := sum.Right.(*BinaryExpression)
	difference := product.Right.(*BinaryExpression)
	if sum.Op != "+" || product.Op != "*" || difference.Op != "-" || sum.Left.(*CallExpression).Name != "increment" {
		t.Fatalf("unexpected expression: %+v", sum)
	}
}

func TestParseErrors(t *testing.T) {
	for _, source := range []string{
		"/*", "import { foo } from \"\n", "fn main() -> Int { return @; }",
		"fn main() -> Int { return 1", "fn main(a Int) -> Int {}",
		"fn main() -> Int { return 9223372036854775808; }", "import {} from \"a\";",
		"fn main() -> Int { return " + strings.Repeat("(", 513) + "1;",
	} {
		if _, err := Parse("broken.smith", source); err == nil || !strings.Contains(err.Error(), "broken.smith:") {
			t.Errorf("expected source-aware parse error for %q, got %v", source, err)
		}
	}
}

func FuzzParse(f *testing.F) {
	f.Add("fn main() -> Int { return 8; }")
	f.Add("import { invalid")
	f.Fuzz(func(t *testing.T, source string) { _, _ = Parse("fuzz.smith", source) })
}
