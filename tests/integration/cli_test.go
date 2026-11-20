package integration

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestCLI(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	scratch := t.TempDir()
	binary := filepath.Join(scratch, "qasmith")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command("go", "build", "-o", binary, "./cmd/qasmith")
	build.Dir = root
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, output)
	}

	call := func(dir string, code int, args ...string) string {
		t.Helper()
		command := exec.Command(binary, args...)
		command.Dir = dir
		output, err := command.CombinedOutput()
		gotCode := 0
		if err != nil {
			var exit *exec.ExitError
			if !errors.As(err, &exit) {
				t.Fatalf("launch CLI: %v", err)
			}
			gotCode = exit.ExitCode()
		}
		if gotCode != code {
			t.Fatalf("%v exited %d, want %d\n%s", args, gotCode, code, output)
		}
		return string(output)
	}
	read := func(path string) []byte {
		t.Helper()
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return content
	}
	put := func(name, source string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(scratch, name), []byte(source), 0644); err != nil {
			t.Fatal(err)
		}
	}
	bell := read(filepath.Join(root, "tests/golden/bell.qasm"))

	t.Run("package and inline circuits match the golden output", func(t *testing.T) {
		for idx, source := range []string{"bell.smith", "examples/program.smith", "bell.smith"} {
			output := filepath.Join(scratch, fmt.Sprintf("bell%d.qasm", idx))
			call(root, 0, "compile", source, "-o", output)
			if got := read(output); !bytes.Equal(got, bell) {
				t.Fatalf("%s output:\n%s", source, got)
			}
		}
	})
	t.Run("relative import and default output path", func(t *testing.T) {
		put("gates.smith", `pub fn flip(q: Qubit) -> Void { x(q); }`)
		put("local.smith", `import { flip } from "./gates.smith";
			fn main() -> MeasuredResult { let q = qubit(); flip(q); return measure(q); }`)
		call(scratch, 0, "compile", "local.smith")
		if got := string(read(filepath.Join(scratch, "local.qasm"))); !strings.Contains(got, "x q[0];\n\nc[0] = measure q[0];") {
			t.Fatalf("invalid output:\n%s", got)
		}
	})
	t.Run("Bell simulation", func(t *testing.T) {
		args := []string{"run", "bell.smith", "--shots", "1000", "--seed", "17"}
		got := call(root, 0, args...)
		if same := call(root, 0, args...); same != got {
			t.Fatal("seeded results differ")
		}
		pattern := regexp.MustCompile(`(?m)^(00|11): ([0-9]+) \([0-9.]+%\)$`)
		matches := pattern.FindAllStringSubmatch(got, -1)
		if len(matches) != 2 || strings.Count(got, " (") != 2 {
			t.Fatalf("unexpected simulation:\n%s", got)
		}
		total := 0
		for _, match := range matches {
			count, _ := strconv.Atoi(match[2])
			if count < 450 || count > 550 {
				t.Fatalf("unexpected distribution:\n%s", got)
			}
			total += count
		}
		if total != 1000 {
			t.Fatalf("result counts sum to %d", total)
		}
	})
	t.Run("imported rotation circuit", func(t *testing.T) {
		output := filepath.Join(scratch, "rotations.qasm")
		call(root, 0, "compile", "examples/rotations.smith", "-o", output)
		if got := read(output); !bytes.Equal(got, read(filepath.Join(root, "tests/golden/rotations.qasm"))) {
			t.Fatalf("rotation output:\n%s", got)
		}
		if result := call(root, 0, "run", "examples/rotations.smith", "--shots", "100", "--seed", "17"); !strings.HasSuffix(result, "11: 100 (100.0%)\n") {
			t.Fatalf("rotation simulation:\n%s", result)
		}
	})
	t.Run("static circuit goldens", func(t *testing.T) {
		for _, name := range []string{"register", "register_layer", "circuit_builder"} {
			source := filepath.Join("examples", name+".smith")
			output := filepath.Join(scratch, name+".qasm")
			call(root, 0, "check", source)
			call(root, 0, "compile", source, "-o", output)
			if got := read(output); !bytes.Equal(got, read(filepath.Join(root, "tests/golden", name+".qasm"))) {
				t.Fatalf("%s output:\n%s", name, got)
			}
		}
		put("inline_layer.smith", `fn prepare(q: QReg<4>) -> Void { for i in 0..4 { h(q[i]); } }
fn applyLayer(q: QReg<4>, angles: Float[4]) -> Void { for i in 0..4 { rz(angles[i], q[i]); } }
fn main() -> MeasuredResult {
    let q = qreg(4); let angles = [0.1, 0.2, 0.3, 0.4]; prepare(q); applyLayer(q, angles); return measure(q);
}`)
		call(scratch, 0, "compile", "inline_layer.smith")
		if !bytes.Equal(read(filepath.Join(scratch, "inline_layer.qasm")), read(filepath.Join(scratch, "register_layer.qasm"))) {
			t.Fatal("inline and imported register helper generated different OpenQASM")
		}
		flags := []string{"--shots", "256", "--seed", "17"}
		imported := call(root, 0, append([]string{"run", "examples/register_layer.smith"}, flags...)...)
		local := call(scratch, 0, append([]string{"run", "inline_layer.smith"}, flags...)...)
		if local != imported {
			t.Fatalf("local and imported results differ: %s / %s", local, imported)
		}
		limited := filepath.Join(scratch, "limited.qasm")
		if got := call(root, 1, "compile", "examples/circuit_builder.smith", "-o", limited, "--max-gates", "21"); !strings.Contains(got, "generated gates limit exceeded") {
			t.Fatalf("missing limit diagnostic: %s", got)
		}
		if _, err := os.Stat(limited); !os.IsNotExist(err) {
			t.Fatalf("limited compilation produced output: %v", err)
		}
	})
	t.Run("QAOA compile run and optimize", func(t *testing.T) {
		put("problem.json", `{"schema":"qasmith.qubo.v1","variables":2,"offset":1,"linear":[-2,3],"quadratic":[{"i":0,"j":1,"coefficient":4}]}`)
		call(scratch, 0, "qaoa", "compile", "problem.json", "--layers", "2", "--gammas", "0.4,0.2", "--betas", "0.3,0.6", "-o", "qaoa.qasm")
		if got := read(filepath.Join(scratch, "qaoa.qasm")); !bytes.Equal(got, read(filepath.Join(root, "tests/golden/qaoa.qasm"))) {
			t.Fatalf("QAOA output:\n%s", got)
		}
		args := []string{"qaoa", "run", "problem.json", "--gammas", "0.4", "--betas", "0.3", "--shots", "80", "--seed", "42"}
		first := call(scratch, 0, args...)
		if first != call(scratch, 0, args...) || !strings.Contains(first, "Estimated expected energy:") || !strings.Contains(first, "Best observed assignment:") {
			t.Fatalf("sampled output:\n%s", first)
		}
		got := call(scratch, 0, "qaoa", "optimize", "problem.json", "--max-evals", "40", "--restarts", "2", "--eval-mode", "sampled", "--seed", "42", "-o", "optimized.qasm")
		if !strings.Contains(got, "Best expected energy:") || !strings.Contains(got, "Classical minimum energy: -1\n") ||
			!strings.Contains(got, "Optimal sample fraction:") || !strings.Contains(string(read(filepath.Join(scratch, "optimized.qasm"))), " = measure ") {
			t.Fatalf("optimized output:\n%s", got)
		}
		call(scratch, 2, "qaoa", "compile", "problem.json", "--layers", "2", "--gammas", "0.4", "--betas", "0.3,0.6")
		call(scratch, 2, "qaoa", "run", "problem.json", "--gammas", "0.4", "--betas", "NaN")
		call(scratch, 2, "qaoa", "run", "problem.json", "--gammas", "0.4", "--betas", "0.3", "--shots", "0")
		call(scratch, 1, "qaoa", "compile", "problem.json", "--gammas", "0.4", "--betas", "0.3", "--max-gates", "1")
		put("bad.json", `{"schema":"qasmith.qubo.v9","variables":2,"offset":1,"linear":[-2,3],"quadratic":[]}`)
		call(scratch, 1, "qaoa", "optimize", "bad.json")
	})
	t.Run("TSP workflow and QUBO interoperability", func(t *testing.T) {
		source := filepath.Join(root, "examples/optimization/tsp/problem.json")
		if got := call(scratch, 0, "tsp", "inspect", source); !strings.Contains(got, "QUBO variables: 9") ||
			!strings.Contains(got, "Uniform feasibility: 1.171875%") {
			t.Fatalf("inspect %s", got)
		}
		if got := call(scratch, 0, "tsp", "solve-classical", source); !strings.Contains(got, "Classical optimal distance: 80") ||
			!strings.Contains(got, "Optimal tours: 2") {
			t.Fatalf("classical %s", got)
		}
		call(scratch, 0, "tsp", "qubo", source, "-o", "tsp.qubo.json")
		flags := []string{"--gammas", "0.4", "--betas", "0.3"}
		call(scratch, 0, append([]string{"tsp", "compile", source, "-o", "tsp.qasm"}, flags...)...)
		call(scratch, 0, append([]string{"qaoa", "compile", "tsp.qubo.json", "-o", "generic.qasm"}, flags...)...)
		if got := read(filepath.Join(scratch, "tsp.qasm")); !bytes.Equal(got, read(filepath.Join(scratch, "generic.qasm"))) ||
			!bytes.Contains(got, []byte("qubit[9] q;")) {
			t.Fatalf("interoperability: %s", got)
		}
		got := call(scratch, 0, "tsp", "run", source, "--gammas", "0", "--betas", "0", "--shots", "1000", "--seed", "42", "--exact")
		if !strings.Contains(got, "Best feasible route:") || !strings.Contains(got, "Exact expected QUBO energy:") ||
			!strings.Contains(got, "Infeasible measurement counts:") {
			t.Fatalf("sample output %s", got)
		}
		got = call(scratch, 0, "tsp", "optimize", source, "--max-evals", "8", "--restarts", "1", "--shots", "10", "--seed", "42", "-o", "tsp-optimized.qasm")
		if !strings.Contains(got, "Optimized expected QUBO energy:") || !strings.Contains(got, "Classical optimal distance: 80") ||
			!bytes.Contains(read(filepath.Join(scratch, "tsp-optimized.qasm")), []byte("qubit[9] q;")) {
			t.Fatalf("optimize %s", got)
		}
		call(scratch, 1, "tsp", "qubo", source, "--penalty", "0")
		call(scratch, 2, "tsp", "compile", source, "--layers", "2", "--gammas", "0.4", "--betas", "0.3")
		call(scratch, 1, "tsp", "solve-classical", source, "--max-cities", "3")
	})
	t.Run("Smith optimization plan end to end", func(t *testing.T) {
		source := filepath.Join(root, "tsp.smith")
		original := read(source)
		fixture := filepath.Join(root, "examples/optimization/tsp/problem.json")
		call(root, 0, "check", "tsp.smith")
		if got := call(root, 0, "deps", "tsp.smith"); !strings.Contains(got, "std/optimization/tsp") || !strings.Contains(got, "std/optimization/qaoa") {
			t.Fatalf("dependency tree %s", got)
		}
		paths := []string{"smith.qasm", "imported.qasm", "tsp-reference.qasm", "generic-reference.qasm"}
		call(scratch, 0, "compile", source, "-o", paths[0])
		call(scratch, 0, "compile", filepath.Join(root, "examples/optimization/language/imported.smith"), "-o", paths[1])
		call(scratch, 0, "tsp", "compile", fixture, "--gammas", "0.4", "--betas", "0.3", "-o", paths[2])
		call(scratch, 0, "tsp", "qubo", fixture, "-o", "smith-reference.qubo.json")
		call(scratch, 0, "qaoa", "compile", "smith-reference.qubo.json", "--gammas", "0.4", "--betas", "0.3", "-o", paths[3])
		for _, path := range paths[1:] {
			if !bytes.Equal(read(filepath.Join(scratch, paths[0])), read(filepath.Join(scratch, path))) {
				t.Fatalf("Smith OpenQASM differs from %s", path)
			}
		}
		got := call(root, 0, "run", "tsp.smith", "--shots", "1000", "--seed", "42")
		if !strings.Contains(got, "Qubits: 9") || !strings.Contains(got, "Layers: 1") || !strings.Contains(got, "QUBO energy") ||
			!strings.Contains(got, "Best feasible distance: 80") || !strings.Contains(got, "Infeasible samples:") {
			t.Fatalf("results %s", got)
		}
		got = call(scratch, 0, "optimize", source, "--max-evals", "12", "--restarts", "1", "--shots", "100", "--seed", "42", "--output-qasm", "smith-optimized.qasm")
		if !strings.Contains(got, "Objective mode: exact") || !strings.Contains(got, "Evaluations: 12") || !strings.Contains(got, "Measurement counts:") ||
			!bytes.Contains(read(filepath.Join(scratch, "smith-optimized.qasm")), []byte("qubit[9] q;")) {
			t.Fatalf("results %s", got)
		}
		got = call(scratch, 0, "optimize", source, "--max-evals", "4", "--eval-mode", "sampled", "--no-sample", "--shots", "20", "--seed", "42")
		if !strings.Contains(got, "Objective mode: sampled") || strings.Contains(got, "Measurement counts:") {
			t.Fatalf("results %s", got)
		}
		if !bytes.Equal(original, read(source)) {
			t.Fatal("optimization modified source")
		}
		if got := call(root, 1, "optimize", "bell.smith"); !strings.Contains(got, "no supported variational") {
			t.Fatalf("error %s", got)
		}
		if got := call(root, 1, "optimize", "examples/classical.smith"); !strings.Contains(got, "requires main returning MeasuredResult") {
			t.Fatalf("error %s", got)
		}
		call(root, 0, "check", "examples/optimization/language/invalid.smith")
		if got := call(root, 1, "compile", "examples/optimization/language/invalid.smith"); !strings.Contains(got, "distances[1][2]") {
			t.Fatalf("error %s", got)
		}
	})
	t.Run("Smith-defined QUBOs", func(t *testing.T) {
		source := filepath.Join(root, "program.smith")
		before := read(source)
		call(root, 0, "check", "program.smith")
		if got := call(root, 0, "deps", "program.smith"); !strings.Contains(got, "std/optimization/qubo") {
			t.Fatalf("missing dependency: %s", got)
		}
		path := func(name string) string {
			return filepath.Join(root, "examples/optimization/qubo-language", name+".smith")
		}
		for _, name := range []string{"matrix", "builder", "constraints", "composition", "imported", "packaged"} {
			call(scratch, 0, "compile", path(name), "-o", name+".qasm")
		}
		for _, name := range []string{"builder", "packaged"} {
			if !bytes.Equal(read(filepath.Join(scratch, "matrix.qasm")), read(filepath.Join(scratch, name+".qasm"))) {
				t.Fatalf("matrix and %s are not equivalent", name)
			}
		}
		call(scratch, 0, "compile", source, "-o", "reference.qasm")
		if !bytes.Equal(read(filepath.Join(scratch, "reference.qasm")), read(filepath.Join(scratch, "imported.qasm"))) {
			t.Fatal("imported Smith constraints differ from root example")
		}
		for _, src := range []string{source, path("imported")} {
			got := call(scratch, 0, "run", src, "--shots", "1000", "--seed", "42")
			if !strings.Contains(got, "Measurement counts:") || !strings.Contains(got, "Best observed QUBO energy:") ||
				strings.Contains(got, "Feasibility rate:") {
				t.Fatalf("unexpected generic run: %s", got)
			}
		}
		got := call(scratch, 0, "optimize", path("imported"), "--max-evals", "20", "--restarts", "2", "--shots", "100", "--seed", "42", "--output-qasm", "custom-optimized.qasm")
		if !strings.Contains(got, "Evaluations: 20") || !strings.Contains(got, "Measurement counts:") ||
			!bytes.Contains(read(filepath.Join(scratch, "custom-optimized.qasm")), []byte("qubit[3] q;")) {
			t.Fatalf("unexpected optimization: %s", got)
		}
		call(scratch, 0, "optimize", source, "--max-evals", "8", "--eval-mode", "sampled", "--seed", "42", "--shots", "100")
		if !bytes.Equal(before, read(source)) {
			t.Fatal("optimization modified the Smith source")
		}
	})
	t.Run("check eval dependencies and version", func(t *testing.T) {
		if got := call(root, 0, "check", "bell.smith"); got != "" {
			t.Fatalf("check output %q", got)
		}
		if got := call(root, 0, "eval", "examples/classical.smith"); got != "8\n" {
			t.Fatalf("eval output %q", got)
		}
		if got := call(root, 0, "deps", "bell.smith"); got != "bell.smith\n└── packages/entangle/bell.smith\n" {
			t.Fatalf("deps output %q", got)
		}
		if got := call(root, 0, "version"); got != "Qasmith v0.9.1\n" {
			t.Fatalf("version output %q", got)
		}
	})
	t.Run("diagnostics and exit status", func(t *testing.T) {
		put("malformed.smith", "fn main() -> Int {\n    return @;\n}")
		put("missing.smith", `import { x } from "./absent.smith"; fn main() -> Int { return 3; }`)
		if got := call(scratch, 1, "check", "malformed.smith"); !strings.Contains(got, "malformed.smith:2:12: error[E1001]") {
			t.Fatalf("lexer diagnostic %s", got)
		}
		if got := call(scratch, 1, "compile", "missing.smith"); !strings.Contains(got, "missing.smith:1:1: error[E2000]") {
			t.Fatalf("import diagnostic %s", got)
		}
		if got := call(root, 1, "compile", "examples/classical.smith"); !strings.Contains(got, "requires main returning MeasuredResult") {
			t.Fatalf("entry diagnostic %s", got)
		}
		call(root, 2, "run", "bell.smith", "--unknown")
		call(root, 2, "unknown")
	})
}
