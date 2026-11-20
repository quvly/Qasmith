package qasmith_test

import (
	"context"
	"testing"

	"github.com/quvly/Qasmith/pkg/extension"
	"github.com/quvly/Qasmith/pkg/qasmith"
)

func TestEditorSkipsRegisteredCallbacks(t *testing.T) {
	reg := &extension.Registry{}
	typeChecks := 0
	if err := reg.RegisterModule("host:editorprobe", extension.Function{
		Name: "probe", Return: extension.Int,
		CheckTypes: func(extension.StaticLimits, []extension.Type) error {
			typeChecks++
			return nil
		},
		Evaluate: func(extension.StaticContext, []extension.Value) (extension.Value, error) {
			t.Fatal("editor or check invoked host evaluator")
			return extension.Value{}, nil
		},
	}); err != nil {
		t.Fatal(err)
	}
	path := fixture(t, map[string]string{"main.smith": `import { probe } from "host:editorprobe";
fn main() -> Int { return probe(); }`})
	compiler := qasmith.Compiler{Registry: reg}
	editor := qasmith.EditorSession{Compiler: compiler}
	snapshot, err := editor.Analyze(context.Background(), path, nil)
	if err != nil || len(snapshot.Diagnostics) != 0 || typeChecks != 0 {
		t.Fatalf("editor analysis: diagnostics %v, error %v, callback calls %d", snapshot.Diagnostics, err, typeChecks)
	}
	if err := compiler.Check(path); err != nil || typeChecks != 1 {
		t.Fatalf("normal check: error %v, callback calls %d", err, typeChecks)
	}
}
