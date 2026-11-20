package syntax

import "testing"

func TestStaticSyntax(t *testing.T) {
	module, err := Parse("static.smith", `fn setup(q: QReg<4>, a: Float[2][3]) -> Void {
    let matrix = [[0, 1], [2, 3]];
    for i in 0..4 {
        for j in i..4 {
            if !false && i + 1 < j || matrix[0][1] == 0 {
                rz(a[1][0], q[i]);
            } else {
                h(q[j]);
            }
        }
    }
}`)
	if err != nil {
		t.Fatal(err)
	}
	f := module.Functions[0]
	if f.Params[0].Type != "QReg<4>" || f.Params[1].Type != "Float[2][3]" {
		t.Fatalf("parameters %+v", f.Params)
	}
	matrix := f.Body.Statements[0].(*VariableDeclaration).Value.(*ArrayLiteral)
	if len(matrix.Elements) != 2 || len(matrix.Elements[0].(*ArrayLiteral).Elements) != 2 {
		t.Fatal("matrix shape")
	}
	outer := f.Body.Statements[1].(*ForStatement)
	inner := outer.Body.Statements[0].(*ForStatement)
	condition := inner.Body.Statements[0].(*IfStatement)
	root := condition.Condition.(*BinaryExpression)
	and := root.Left.(*BinaryExpression)
	not := and.Left.(*UnaryExpression)
	comparison := and.Right.(*BinaryExpression)
	index := root.Right.(*BinaryExpression).Left.(*IndexExpression)
	if root.Op != "||" || and.Op != "&&" || not.Op != "!" || comparison.Op != "<" || comparison.Left.(*BinaryExpression).Op != "+" ||
		index.Target.(*IndexExpression).Index.(*IntegerLiteral).Value != 0 || index.Index.(*IntegerLiteral).Value != 1 ||
		condition.Else == nil || index.At.Line != 5 {
		t.Fatalf("unexpected condition %+v", condition)
	}
}

func TestMalformedStaticSyntax(t *testing.T) {
	for _, body := range []string{
		"for i in 1.2.3 {}", "for i in 0...", "for i of 0..4 {}", "for i in 0.. {}",
		"if true { else {}", "if true {} else",
		"let a = [1, ;", "let a = [1]; a[];",
	} {
		if _, err := Parse("broken.smith", "fn main() -> Int { "+body+" return 0; }"); err == nil {
			t.Errorf("accepted %s", body)
		}
	}
}
