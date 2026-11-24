# Workstream H: Proposed text (needs owner approval)

**Do not merge until the owner has reviewed and approved the wording below.**

This PR contains proposed additions to the README and documentation. Both
sections are drafts and should not be published without explicit owner sign-off.

---

## Proposed: "Status and claims" section

Add this section to the README after "Where to go next":

### Status and claims

The following claims are backed by tests or documentation in this repository:

- **Bell state simulation produces only `00` and `11` with roughly equal counts.**
  Statevector cross-check: `tests/cross-check/cross_check_test.go:TestCorpusStatevectors/bell`.
  Statistical test: `tests/cross-check/statistical_test.go:TestChiSquareBellState`.
- **Classical evaluation of `examples/classical.smith` returns `8`.**
  Integration test: `tests/integration/cli_test.go:TestCLI`.
- **The compiler rejects programs that copy qubit references, reuse a qubit
  after measurement, or pass the same qubit as both CX operands.**
  Semantic tests: `pkg/qasmith/compiler_test.go` and `pkg/qasmith/rzz_test.go`.
- **Generated OpenQASM is syntactically valid per the reference parser.**
  CI job: `openqasm-validation` in `.github/workflows/ci.yml`.
  Historical evidence: `docs/internal/phase-2a-compliance.md`.
- **The simulator statevector matches analytical computation for 20 circuits
  (Bell, GHZ, rotations, registers, controlled gates).**
  Cross-check corpus: `tests/cross-check/corpus/` and
  `tests/cross-check/cross_check_test.go`.

Other claims in the README are documentation of intent and should not be read
as guarantees until corresponding tests are added.

---

## Proposed: "Commercial relationship" paragraph

Add this paragraph at the bottom of the README, after the license line:

### Commercial relationship

Qasmith is an open-source project licensed under the Apache License 2.0. It was
created by [Quvly](https://quvly.eu), which also maintains a separate commercial
quantum experiment execution engine. The commercial product is not part of this
repository and is not covered by the Apache-2.0 license. Contributions to
Qasmith are welcome under the terms of the license; see
[CONTRIBUTING.md](CONTRIBUTING.md).

---

## Owner action required

Before merging, the owner should:

1. Review both drafts above.
2. Confirm or revise the "Status and claims" list.
3. Confirm or revise the "Commercial relationship" paragraph.
4. Reply to the PR with approval or requested changes.

**Do not merge this PR until the owner has approved the wording.**
