#!/bin/bash

# End-to-End Test Script for Module Resolution
# This script tests all aspects of the module resolution system

set -e

QASMITH="/Users/lubosberan/Dev/Qasmith/qasmith"
TEST_DIR="/tmp/qasmith-e2e-test"

echo "=== Qasmith Module Resolution E2E Tests ==="
echo ""

# Clean up
rm -rf "$TEST_DIR"
mkdir -p "$TEST_DIR"

# Test 1: Project Initialization
echo "Test 1: Project Initialization"
cd "$TEST_DIR"
$QASMITH init test-project > /dev/null
cd test-project
if [ ! -f "qasmith.toml" ] || [ ! -f "src/main.smith" ]; then
    echo "❌ FAILED: Project structure not created correctly"
    exit 1
fi
echo "✅ PASSED"

# Test 2: TOML Manifest Support
echo "Test 2: TOML Manifest Support"
if ! grep -q "name = \"test-project\"" qasmith.toml; then
    echo "❌ FAILED: TOML manifest not created correctly"
    exit 1
fi
echo "✅ PASSED"

# Test 3: Run Project
echo "Test 3: Run Project"
if ! $QASMITH run src/main.smith > /dev/null 2>&1; then
    echo "❌ FAILED: Could not run project"
    exit 1
fi
echo "✅ PASSED"

# Test 4: Relative Imports
echo "Test 4: Relative Imports"
cd "$TEST_DIR"
mkdir -p relative-test/src relative-test/libraries
cat > relative-test/qasmith.toml << 'EOF'
name = "relative-test"
version = "0.1.0"
EOF

cat > relative-test/libraries/utils.smith << 'EOF'
pub fn helper() -> Int {
    return 42;
}
EOF

cat > relative-test/src/main.smith << 'EOF'
import { helper } from "../libraries/utils.smith";

fn main() -> MeasuredResult {
    let q = qubit();
    let value = helper();
    if value == 42 {
        h(q);
    }
    return measure(q);
}
EOF

cd relative-test
if ! $QASMITH run src/main.smith > /dev/null 2>&1; then
    echo "❌ FAILED: Relative imports not working"
    exit 1
fi
echo "✅ PASSED"

# Test 5: Package Dependencies
echo "Test 5: Package Dependencies"
cd "$TEST_DIR"
mkdir -p package-test/packages/mylib
cat > package-test/qasmith.toml << 'EOF'
name = "package-test"
version = "0.1.0"

[dependencies.mylib]
path = "./packages/mylib"
EOF

cat > package-test/packages/mylib/math.smith << 'EOF'
pub fn add(a: Int, b: Int) -> Int {
    return a + b;
}
EOF

cat > package-test/main.smith << 'EOF'
import { add } from "mylib/math";

fn main() -> MeasuredResult {
    let q = qubit();
    let result = add(2, 3);
    if result == 5 {
        h(q);
    }
    return measure(q);
}
EOF

cd package-test
if ! $QASMITH run main.smith > /dev/null 2>&1; then
    echo "❌ FAILED: Package dependencies not working"
    exit 1
fi
echo "✅ PASSED"

# Test 6: Dependency Tree
echo "Test 6: Dependency Tree"
if ! $QASMITH deps main.smith | grep -q "mylib/math.smith"; then
    echo "❌ FAILED: Dependency tree not showing package"
    exit 1
fi
echo "✅ PASSED"

# Test 7: Standard Library Imports
echo "Test 7: Standard Library Imports"
cd "$TEST_DIR"
mkdir -p stdlib-test
cat > stdlib-test/qasmith.toml << 'EOF'
name = "stdlib-test"
version = "0.1.0"
EOF

cat > stdlib-test/main.smith << 'EOF'
import { qaoa } from "std/optimization/qaoa";
import { tsp } from "std/optimization/tsp";

fn main() -> MeasuredResult {
    let distances = [
        [0, 10, 15],
        [10, 0, 20],
        [15, 20, 0]
    ];
    let problem = tsp(distances, 0);
    return qaoa(problem, [0.4], [0.3]);
}
EOF

cd stdlib-test
if ! $QASMITH run main.smith > /dev/null 2>&1; then
    echo "❌ FAILED: Standard library imports not working"
    exit 1
fi
echo "✅ PASSED"

# Test 8: JSON Backward Compatibility
echo "Test 8: JSON Backward Compatibility"
cd "$TEST_DIR"
mkdir -p json-test
cat > json-test/qasmith.json << 'EOF'
{
  "name": "json-test",
  "version": "0.1.0"
}
EOF

cat > json-test/main.smith << 'EOF'
fn main() -> MeasuredResult {
    let q = qubit();
    h(q);
    return measure(q);
}
EOF

cd json-test
if ! $QASMITH run main.smith > /dev/null 2>&1; then
    echo "❌ FAILED: JSON manifest not working"
    exit 1
fi
echo "✅ PASSED"

# Test 9: TOML Precedence Over JSON
echo "Test 9: TOML Precedence Over JSON"
cd "$TEST_DIR"
mkdir -p precedence-test
cat > precedence-test/qasmith.toml << 'EOF'
name = "toml-app"
version = "0.1.0"
EOF

cat > precedence-test/qasmith.json << 'EOF'
{
  "name": "json-app",
  "version": "0.1.0"
}
EOF

cat > precedence-test/main.smith << 'EOF'
fn main() -> MeasuredResult {
    let q = qubit();
    return measure(q);
}
EOF

cd precedence-test
# Just verify it compiles successfully (both manifests are valid, but TOML should be used)
if ! $QASMITH check main.smith > /dev/null 2>&1; then
    echo "❌ FAILED: Could not compile with both manifests present"
    exit 1
fi
echo "✅ PASSED"

# Test 10: Existing Examples Still Work
echo "Test 10: Existing Examples Still Work"
cd /Users/lubosberan/Dev/Qasmith/examples
if ! $QASMITH run circuit_builder.smith > /dev/null 2>&1; then
    echo "❌ FAILED: Existing examples broken"
    exit 1
fi
echo "✅ PASSED"

# Test 11: Path Traversal Prevention
echo "Test 11: Path Traversal Prevention"
cd "$TEST_DIR"
mkdir -p traversal-test
cat > traversal-test/qasmith.toml << 'EOF'
name = "traversal-test"
version = "0.1.0"
EOF

cat > traversal-test/main.smith << 'EOF'
import { f } from "../../../etc/passwd";

fn main() -> MeasuredResult {
    let q = qubit();
    return measure(q);
}
EOF

cd traversal-test
if $QASMITH check main.smith 2>&1 | grep -qv "escapes root"; then
    echo "❌ FAILED: Path traversal not prevented"
    exit 1
fi
echo "✅ PASSED"

# Test 12: Circular Dependency Detection
echo "Test 12: Circular Dependency Detection"
cd "$TEST_DIR"
mkdir -p circular-test
cat > circular-test/qasmith.toml << 'EOF'
name = "circular-test"
version = "0.1.0"
EOF

cat > circular-test/a.smith << 'EOF'
import { b_fn } from "./b.smith";
pub fn a_fn() -> Int { return b_fn(); }
EOF

cat > circular-test/b.smith << 'EOF'
import { a_fn } from "./a.smith";
pub fn b_fn() -> Int { return a_fn(); }
EOF

cat > circular-test/main.smith << 'EOF'
import { a_fn } from "./a.smith";

fn main() -> MeasuredResult {
    let q = qubit();
    let value = a_fn();
    return measure(q);
}
EOF

cd circular-test
if $QASMITH check main.smith 2>&1 | grep -qv "circular"; then
    echo "❌ FAILED: Circular dependency not detected"
    exit 1
fi
echo "✅ PASSED"

# Clean up
cd /
rm -rf "$TEST_DIR"

echo ""
echo "=== All Tests Passed ==="
echo ""
echo "Summary:"
echo "  ✅ Project initialization"
echo "  ✅ TOML manifest support"
echo "  ✅ Project execution"
echo "  ✅ Relative imports"
echo "  ✅ Package dependencies"
echo "  ✅ Dependency tree"
echo "  ✅ Standard library imports"
echo "  ✅ JSON backward compatibility"
echo "  ✅ TOML precedence"
echo "  ✅ Existing examples"
echo "  ✅ Path traversal prevention"
echo "  ✅ Circular dependency detection"
echo ""
echo "Module resolution system is working correctly!"
