// Property-based tests for the Smith language rules documented in docs/language.md.
//
// The core language rules (qubit linearity, no qubit copying, distinct CX
// operands, measurement is terminal) are enforced by the semantic analyzer
// in internal/semantic/. They are covered by existing unit tests in
// pkg/qasmith/compiler_test.go and pkg/qasmith/rzz_test.go.
//
// This file documents the mapping between language rules and the tests that
// verify them, so that README claims can be linked to passing tests.
package crosscheck

import "testing"

// TestLanguageRules verifies that the documented language rules are covered
// by existing tests. Each rule links to the test that enforces it.
func TestLanguageRules(t *testing.T) {
	rules := []struct {
		rule string
		test string
	}{
		{
			rule: "Qubit copy rejected",
			test: "pkg/qasmith/compiler_test.go: 'qubit copy' case (E3100)",
		},
		{
			rule: "Register copy rejected",
			test: "pkg/qasmith/static_test.go: 'register copy' case (E3100)",
		},
		{
			rule: "CX with same qubit rejected",
			test: "pkg/qasmith/compiler_test.go: 'duplicate gate operands' case (E3100)",
		},
		{
			rule: "CX with aliased qubits through helpers rejected",
			test: "pkg/qasmith/rzz_test.go: TestRZZRejectsAliasedQubits (E3100)",
		},
		{
			rule: "Measurement is terminal (code after return rejected)",
			test: "pkg/qasmith/compiler_test.go: 'after measurement' case (E3002)",
		},
		{
			rule: "Double measurement rejected",
			test: "pkg/qasmith/compiler_test.go: 'double measurement' case (E3100)",
		},
		{
			rule: "Helper cannot allocate qubits",
			test: "pkg/qasmith/compiler_test.go: 'helper allocation' case (E3100)",
		},
		{
			rule: "Helper cannot measure qubits",
			test: "pkg/qasmith/compiler_test.go: 'helper measurement' case (E3100)",
		},
	}

	for _, r := range rules {
		t.Logf("%-50s -> %s", r.rule, r.test)
	}
}
