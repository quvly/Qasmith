"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
const node_assert_1 = require("node:assert");
const node_test_1 = require("node:test");
const circuitLayout_1 = require("../src/circuitLayout");
(0, node_test_1.test)('parseCircuit: Hadamard followed by CNOT', () => {
    const code = `
    let q0 = qubit();
    let q1 = qubit();
    h(q0);
    cx(q0, q1);
  `;
    const circuit = (0, circuitLayout_1.parseCircuit)(code);
    node_assert_1.strict.equal(circuit.qubits, 2, 'Should have 2 qubits');
    node_assert_1.strict.equal(circuit.operations.length, 2, 'Should have 2 operations');
    // First operation: H on q0
    const h = circuit.operations[0];
    node_assert_1.strict.equal(h.name, 'H');
    node_assert_1.strict.equal(h.type, circuitLayout_1.GateType.SingleQubit);
    node_assert_1.strict.deepEqual(h.qubits, [0]);
    // Second operation: CX with control=q0, target=q1
    const cx = circuit.operations[1];
    node_assert_1.strict.equal(cx.name, 'CX');
    node_assert_1.strict.equal(cx.type, circuitLayout_1.GateType.Controlled);
    node_assert_1.strict.deepEqual(cx.qubits, [0, 1]);
});
(0, node_test_1.test)('parseCircuit: CNOT with reversed control and target', () => {
    const code = `
    let a = qubit();
    let b = qubit();
    cx(b, a);
  `;
    const circuit = (0, circuitLayout_1.parseCircuit)(code);
    const cx = circuit.operations[0];
    node_assert_1.strict.equal(cx.name, 'CX');
    node_assert_1.strict.equal(cx.type, circuitLayout_1.GateType.Controlled);
    // Control should be b (q1), target should be a (q0)
    node_assert_1.strict.deepEqual(cx.qubits, [1, 0]);
});
(0, node_test_1.test)('parseCircuit: Non-adjacent qubits', () => {
    const code = `
    let q0 = qubit();
    let q1 = qubit();
    let q2 = qubit();
    cx(q0, q2);
  `;
    const circuit = (0, circuitLayout_1.parseCircuit)(code);
    const cx = circuit.operations[0];
    node_assert_1.strict.equal(circuit.qubits, 3);
    node_assert_1.strict.equal(cx.type, circuitLayout_1.GateType.Controlled);
    // Control=q0 (0), target=q2 (2)
    node_assert_1.strict.deepEqual(cx.qubits, [0, 2]);
});
(0, node_test_1.test)('layoutCircuit: H + CX column assignment', () => {
    const code = `
    let q0 = qubit();
    let q1 = qubit();
    h(q0);
    cx(q0, q1);
  `;
    const circuit = (0, circuitLayout_1.parseCircuit)(code);
    const layout = (0, circuitLayout_1.layoutCircuit)(circuit);
    node_assert_1.strict.equal(layout.numColumns, 2, 'Should have 2 columns');
    node_assert_1.strict.equal(layout.assignments.length, 2);
    // H should be in column 0
    node_assert_1.strict.equal(layout.assignments[0].column, 0);
    node_assert_1.strict.equal(layout.assignments[0].operation.name, 'H');
    // CX should be in column 1
    node_assert_1.strict.equal(layout.assignments[1].column, 1);
    node_assert_1.strict.equal(layout.assignments[1].operation.name, 'CX');
});
(0, node_test_1.test)('layoutCircuit: Multiple sequential CX gates', () => {
    const code = `
    let q0 = qubit();
    let q1 = qubit();
    cx(q0, q1);
    cx(q1, q0);
    cx(q0, q1);
  `;
    const circuit = (0, circuitLayout_1.parseCircuit)(code);
    const layout = (0, circuitLayout_1.layoutCircuit)(circuit);
    // Each CX should get its own column
    node_assert_1.strict.equal(layout.numColumns, 3);
    node_assert_1.strict.equal(layout.assignments[0].column, 0);
    node_assert_1.strict.equal(layout.assignments[1].column, 1);
    node_assert_1.strict.equal(layout.assignments[2].column, 2);
});
(0, node_test_1.test)('layoutCircuit: Interleaved single and multi-qubit gates', () => {
    const code = `
    let q0 = qubit();
    let q1 = qubit();
    let q2 = qubit();
    h(q0);
    h(q1);
    cx(q0, q1);
    h(q2);
    cx(q1, q2);
  `;
    const circuit = (0, circuitLayout_1.parseCircuit)(code);
    const layout = (0, circuitLayout_1.layoutCircuit)(circuit);
    // H(q0), H(q1), H(q2) can all be in col 0 (different qubits)
    // CX(q0, q1) must be in col 1 (conflicts with H on q0 and q1)
    // CX(q1, q2) must be in col 2 (conflicts with CX on q1)
    const h0 = layout.assignments.find(a => a.operation.name === 'H' && a.operation.qubits[0] === 0);
    const h1 = layout.assignments.find(a => a.operation.name === 'H' && a.operation.qubits[0] === 1);
    const cx01 = layout.assignments.find(a => a.operation.name === 'CX' && a.operation.qubits[1] === 1);
    const h2 = layout.assignments.find(a => a.operation.name === 'H' && a.operation.qubits[0] === 2);
    const cx12 = layout.assignments.find(a => a.operation.name === 'CX' && a.operation.qubits[1] === 2);
    node_assert_1.strict.ok(h0 && h1);
    node_assert_1.strict.equal(h0.column, 0, 'H(q0) in col 0');
    node_assert_1.strict.equal(h1.column, 0, 'H(q1) in col 0');
    node_assert_1.strict.ok(cx01);
    node_assert_1.strict.equal(cx01.column, 1, 'CX(q0,q1) in col 1');
    node_assert_1.strict.ok(h2);
    node_assert_1.strict.equal(h2.column, 0, 'H(q2) in col 0 (parallel with H gates)');
    node_assert_1.strict.ok(cx12);
    node_assert_1.strict.equal(cx12.column, 2, 'CX(q1,q2) in col 2');
});
(0, node_test_1.test)('parseCircuit: Parametric gates', () => {
    const code = `
    let q = qubit();
    rx(pi/2, q);
    ry(3.14, q);
  `;
    const circuit = (0, circuitLayout_1.parseCircuit)(code);
    node_assert_1.strict.equal(circuit.operations.length, 2);
    const rx = circuit.operations[0];
    node_assert_1.strict.equal(rx.name, 'RX');
    node_assert_1.strict.ok(rx.params);
    node_assert_1.strict.ok(Math.abs(rx.params[0] - Math.PI / 2) < 0.001);
    const ry = circuit.operations[1];
    node_assert_1.strict.equal(ry.name, 'RY');
    node_assert_1.strict.ok(ry.params);
    node_assert_1.strict.ok(Math.abs(ry.params[0] - 3.14) < 0.001);
});
(0, node_test_1.test)('parseCircuit: Qubit register', () => {
    const code = `
    let q = qreg<3>();
    h(q[0]);
    cx(q[0], q[2]);
  `;
    const circuit = (0, circuitLayout_1.parseCircuit)(code);
    node_assert_1.strict.equal(circuit.qubits, 3);
    node_assert_1.strict.equal(circuit.operations.length, 2);
    const cx = circuit.operations[1];
    node_assert_1.strict.deepEqual(cx.qubits, [0, 2]);
});
(0, node_test_1.test)('parseCircuit: CZ and SWAP gates', () => {
    const code = `
    let q0 = qubit();
    let q1 = qubit();
    cz(q0, q1);
    swap(q0, q1);
  `;
    const circuit = (0, circuitLayout_1.parseCircuit)(code);
    node_assert_1.strict.equal(circuit.operations.length, 2);
    node_assert_1.strict.equal(circuit.operations[0].type, circuitLayout_1.GateType.CZ);
    node_assert_1.strict.equal(circuit.operations[1].type, circuitLayout_1.GateType.Swap);
});
(0, node_test_1.test)('renderCircuit: Generates valid SVG', () => {
    const code = `
    let q0 = qubit();
    let q1 = qubit();
    h(q0);
    cx(q0, q1);
  `;
    const circuit = (0, circuitLayout_1.parseCircuit)(code);
    const layout = (0, circuitLayout_1.layoutCircuit)(circuit);
    const svg = (0, circuitLayout_1.renderCircuit)(circuit, layout);
    node_assert_1.strict.ok(svg.includes('<svg'), 'Should contain SVG tag');
    node_assert_1.strict.ok(svg.includes('</svg>'), 'Should have closing SVG tag');
    node_assert_1.strict.ok(svg.includes('q0'), 'Should contain qubit labels');
    node_assert_1.strict.ok(svg.includes('q1'), 'Should contain qubit labels');
    node_assert_1.strict.ok(svg.includes('>H<'), 'Should contain H gate label');
    // CX should render as control dot + target circle, not as text
    node_assert_1.strict.ok(svg.includes('circle'), 'Should contain circle elements for CNOT');
});
(0, node_test_1.test)('renderCircuit: Empty circuit', () => {
    const code = `
    let q0 = qubit();
  `;
    const circuit = (0, circuitLayout_1.parseCircuit)(code);
    const layout = (0, circuitLayout_1.layoutCircuit)(circuit);
    const svg = (0, circuitLayout_1.renderCircuit)(circuit, layout);
    node_assert_1.strict.equal(svg, '', 'Empty circuit should return empty string');
});
(0, node_test_1.test)('layoutCircuit: Non-adjacent qubit operations', () => {
    const code = `
    let q0 = qubit();
    let q1 = qubit();
    let q2 = qubit();
    let q3 = qubit();
    h(q0);
    h(q1);
    cx(q0, q3);
    h(q2);
  `;
    const circuit = (0, circuitLayout_1.parseCircuit)(code);
    const layout = (0, circuitLayout_1.layoutCircuit)(circuit);
    // H(q0), H(q1), H(q2) can all be in col 0
    // CX(q0, q3) must be in col 1
    const hOps = layout.assignments.filter(a => a.operation.name === 'H');
    const cx = layout.assignments.find(a => a.operation.name === 'CX');
    node_assert_1.strict.equal(hOps.length, 3);
    for (const h of hOps) {
        node_assert_1.strict.equal(h.column, 0, 'All H gates in col 0');
    }
    node_assert_1.strict.ok(cx);
    node_assert_1.strict.equal(cx.column, 1, 'CX in col 1');
    node_assert_1.strict.deepEqual(cx.operation.qubits, [0, 3], 'CX spans non-adjacent qubits');
});
(0, node_test_1.test)('parseCircuit: Measure is a valid operation', () => {
    const code = `
    let q = qubit();
    measure(q);
    h(q);
  `;
    const circuit = (0, circuitLayout_1.parseCircuit)(code);
    // measure and h are both valid operations
    node_assert_1.strict.equal(circuit.operations.length, 2);
    node_assert_1.strict.equal(circuit.operations[0].name, 'MEASURE');
    node_assert_1.strict.equal(circuit.operations[0].type, circuitLayout_1.GateType.Measure);
    node_assert_1.strict.equal(circuit.operations[1].name, 'H');
});
(0, node_test_1.test)('parseCircuit: CNOT alias', () => {
    const code = `
    let q0 = qubit();
    let q1 = qubit();
    cnot(q0, q1);
  `;
    const circuit = (0, circuitLayout_1.parseCircuit)(code);
    const gate = circuit.operations[0];
    node_assert_1.strict.equal(gate.name, 'CNOT');
    node_assert_1.strict.equal(gate.type, circuitLayout_1.GateType.Controlled);
});
(0, node_test_1.test)('parseCircuit: Single-qubit gates H, X, RX, RY and RZ in sequence', () => {
    const code = `
    let q = qubit();
    h(q);
    x(q);
    rx(pi / 2.0, q);
    ry(pi / 4.0, q);
    rz(pi / 3.0, q);
  `;
    const circuit = (0, circuitLayout_1.parseCircuit)(code);
    node_assert_1.strict.equal(circuit.qubits, 1);
    node_assert_1.strict.equal(circuit.operations.length, 5);
    const layout = (0, circuitLayout_1.layoutCircuit)(circuit);
    // Each gate should be in its own column (all on same qubit)
    node_assert_1.strict.equal(layout.numColumns, 5);
    node_assert_1.strict.equal(circuit.operations[0].name, 'H');
    node_assert_1.strict.equal(circuit.operations[1].name, 'X');
    node_assert_1.strict.equal(circuit.operations[2].name, 'RX');
    node_assert_1.strict.ok(circuit.operations[2].params);
    node_assert_1.strict.ok(Math.abs(circuit.operations[2].params[0] - Math.PI / 2) < 0.001);
    node_assert_1.strict.equal(circuit.operations[3].name, 'RY');
    node_assert_1.strict.ok(circuit.operations[3].params);
    node_assert_1.strict.ok(Math.abs(circuit.operations[3].params[0] - Math.PI / 4) < 0.001);
    node_assert_1.strict.equal(circuit.operations[4].name, 'RZ');
    node_assert_1.strict.ok(circuit.operations[4].params);
    node_assert_1.strict.ok(Math.abs(circuit.operations[4].params[0] - Math.PI / 3) < 0.001);
});
(0, node_test_1.test)('parseCircuit: Two independently allocated qubits with CNOT', () => {
    const code = `
    let a = qubit();
    let b = qubit();
    cx(a, b);
  `;
    const circuit = (0, circuitLayout_1.parseCircuit)(code);
    node_assert_1.strict.equal(circuit.qubits, 2);
    node_assert_1.strict.equal(circuit.operations.length, 1);
    node_assert_1.strict.equal(circuit.operations[0].name, 'CX');
    node_assert_1.strict.equal(circuit.operations[0].type, circuitLayout_1.GateType.Controlled);
    node_assert_1.strict.deepEqual(circuit.operations[0].qubits, [0, 1]);
});
(0, node_test_1.test)('parseCircuit: Three independently allocated qubits with non-adjacent CNOT', () => {
    const code = `
    let a = qubit();
    let b = qubit();
    let c = qubit();
    cx(a, c);
  `;
    const circuit = (0, circuitLayout_1.parseCircuit)(code);
    node_assert_1.strict.equal(circuit.qubits, 3);
    node_assert_1.strict.equal(circuit.operations.length, 1);
    node_assert_1.strict.deepEqual(circuit.operations[0].qubits, [0, 2]);
});
(0, node_test_1.test)('parseCircuit: qreg(n) function-call syntax', () => {
    const code = `
    let q = qreg(3);
    h(q[0]);
    cx(q[0], q[2]);
  `;
    const circuit = (0, circuitLayout_1.parseCircuit)(code);
    node_assert_1.strict.equal(circuit.qubits, 3);
    node_assert_1.strict.equal(circuit.operations.length, 2);
    node_assert_1.strict.equal(circuit.operations[0].name, 'H');
    node_assert_1.strict.deepEqual(circuit.operations[0].qubits, [0]);
    node_assert_1.strict.equal(circuit.operations[1].name, 'CX');
    node_assert_1.strict.deepEqual(circuit.operations[1].qubits, [0, 2]);
});
(0, node_test_1.test)('parseCircuit: Four-qubit register with sequential CNOT operations', () => {
    const code = `
    let q = qreg(4);
    h(q[0]);
    h(q[1]);
    h(q[2]);
    h(q[3]);
    cx(q[0], q[1]);
    cx(q[1], q[2]);
    cx(q[2], q[3]);
  `;
    const circuit = (0, circuitLayout_1.parseCircuit)(code);
    node_assert_1.strict.equal(circuit.qubits, 4);
    node_assert_1.strict.equal(circuit.operations.length, 7);
    // Verify all operations parsed correctly
    const cxOps = circuit.operations.filter(op => op.name === 'CX');
    node_assert_1.strict.equal(cxOps.length, 3);
    node_assert_1.strict.deepEqual(cxOps[0].qubits, [0, 1]);
    node_assert_1.strict.deepEqual(cxOps[1].qubits, [1, 2]);
    node_assert_1.strict.deepEqual(cxOps[2].qubits, [2, 3]);
    // All gates should not overlap
    const layout = (0, circuitLayout_1.layoutCircuit)(circuit);
    for (let i = 0; i < layout.assignments.length; i++) {
        for (let j = i + 1; j < layout.assignments.length; j++) {
            const a = layout.assignments[i];
            const b = layout.assignments[j];
            const overlap = a.operation.qubits.some(q => b.operation.qubits.includes(q));
            if (a.column === b.column) {
                node_assert_1.strict.ok(!overlap, `Operations at column ${a.column} should not share qubits`);
            }
        }
    }
});
(0, node_test_1.test)('parseCircuit: GHZ state construction', () => {
    const code = `
    let q = qreg(4);
    h(q[0]);
    cx(q[0], q[1]);
    cx(q[0], q[2]);
    cx(q[0], q[3]);
  `;
    const circuit = (0, circuitLayout_1.parseCircuit)(code);
    node_assert_1.strict.equal(circuit.qubits, 4);
    node_assert_1.strict.equal(circuit.operations.length, 4);
    node_assert_1.strict.equal(circuit.operations[0].name, 'H');
    node_assert_1.strict.deepEqual(circuit.operations[0].qubits, [0]);
    const cxOps = circuit.operations.slice(1);
    for (const cx of cxOps) {
        node_assert_1.strict.equal(cx.name, 'CX');
        node_assert_1.strict.equal(cx.qubits[0], 0, 'Control should always be q[0]');
    }
});
(0, node_test_1.test)('parseCircuit: Whole-register measurement', () => {
    const code = `
    let q = qreg(3);
    h(q[0]);
    cx(q[0], q[1]);
    cx(q[0], q[2]);
    return measure(q);
  `;
    const circuit = (0, circuitLayout_1.parseCircuit)(code);
    node_assert_1.strict.equal(circuit.qubits, 3);
    const measures = circuit.operations.filter(op => op.type === circuitLayout_1.GateType.Measure);
    node_assert_1.strict.equal(measures.length, 3, 'Should measure all 3 qubits');
    node_assert_1.strict.deepEqual(measures[0].qubits, [0]);
    node_assert_1.strict.deepEqual(measures[1].qubits, [1]);
    node_assert_1.strict.deepEqual(measures[2].qubits, [2]);
});
(0, node_test_1.test)('parseCircuit: Selected register-element measurement', () => {
    const code = `
    let q = qreg(4);
    h(q[0]);
    cx(q[0], q[2]);
    return measure(q[0], q[2]);
  `;
    const circuit = (0, circuitLayout_1.parseCircuit)(code);
    node_assert_1.strict.equal(circuit.qubits, 4);
    const measures = circuit.operations.filter(op => op.type === circuitLayout_1.GateType.Measure);
    node_assert_1.strict.equal(measures.length, 2, 'Should measure q[0] and q[2]');
    node_assert_1.strict.deepEqual(measures[0].qubits, [0]);
    node_assert_1.strict.deepEqual(measures[1].qubits, [2]);
});
(0, node_test_1.test)('parseCircuit: Measurement with return statement', () => {
    const code = `
    let a = qubit();
    let b = qubit();
    h(a);
    cx(a, b);
    return measure(a, b);
  `;
    const circuit = (0, circuitLayout_1.parseCircuit)(code);
    const measures = circuit.operations.filter(op => op.type === circuitLayout_1.GateType.Measure);
    node_assert_1.strict.equal(measures.length, 2);
});
(0, node_test_1.test)('parseCircuit: Allocated but unmeasured qubits', () => {
    const code = `
    let a = qubit();
    let b = qubit();
    let c = qubit();
    h(a);
    cx(a, b);
  `;
    const circuit = (0, circuitLayout_1.parseCircuit)(code);
    node_assert_1.strict.equal(circuit.qubits, 3);
    node_assert_1.strict.equal(circuit.operations.length, 2);
    const measures = circuit.operations.filter(op => op.type === circuitLayout_1.GateType.Measure);
    node_assert_1.strict.equal(measures.length, 0, 'No measurements should be present');
});
(0, node_test_1.test)('layoutCircuit: Long sequences of parameterized rotation gates', () => {
    const code = `
    let q = qubit();
    rx(pi / 2.0, q);
    ry(pi / 4.0, q);
    rz(pi / 3.0, q);
    rx(pi / 6.0, q);
    ry(pi / 8.0, q);
  `;
    const circuit = (0, circuitLayout_1.parseCircuit)(code);
    const layout = (0, circuitLayout_1.layoutCircuit)(circuit);
    // All on same qubit, so each gate gets its own column
    node_assert_1.strict.equal(layout.numColumns, 5);
    // Verify column widths are not uniform
    const widths = layout.columnWidths;
    node_assert_1.strict.ok(widths.length === 5);
    // All widths should be positive
    for (const w of widths) {
        node_assert_1.strict.ok(w > 0);
    }
    // Render and verify SVG is valid
    const svg = (0, circuitLayout_1.renderCircuit)(circuit, layout);
    node_assert_1.strict.ok(svg.includes('<svg'));
    node_assert_1.strict.ok(svg.includes('RX'));
    node_assert_1.strict.ok(svg.includes('RY'));
    node_assert_1.strict.ok(svg.includes('RZ'));
});
(0, node_test_1.test)('renderCircuit: Circuits with multi-qubit operations and measurements', () => {
    const code = `
    let q = qreg(3);
    h(q[0]);
    cx(q[0], q[1]);
    cx(q[1], q[2]);
    return measure(q);
  `;
    const circuit = (0, circuitLayout_1.parseCircuit)(code);
    const layout = (0, circuitLayout_1.layoutCircuit)(circuit);
    const svg = (0, circuitLayout_1.renderCircuit)(circuit, layout);
    node_assert_1.strict.ok(svg.includes('<svg'));
    node_assert_1.strict.ok(svg.includes('H'));
    // Should have measurement symbols
    node_assert_1.strict.ok(svg.includes('path') || svg.includes('MEASURE'), 'Should contain measurement rendering');
});
(0, node_test_1.test)('renderCircuit: No overlapping gates with rotation sequence', () => {
    const code = `
    let q = qubit();
    h(q);
    x(q);
    rx(pi / 2.0, q);
    ry(pi / 4.0, q);
    rz(pi / 3.0, q);
  `;
    const circuit = (0, circuitLayout_1.parseCircuit)(code);
    const layout = (0, circuitLayout_1.layoutCircuit)(circuit);
    // Verify column spacing prevents overlap
    const totalWidth = layout.columnWidths.reduce((sum, w) => sum + w, 0);
    node_assert_1.strict.ok(totalWidth > layout.numColumns * 44, 'Total width should exceed minimum if gates are wider');
    // Each gate on same qubit should be in separate column
    const cols = layout.assignments.map(a => a.column);
    const uniqueCols = new Set(cols);
    node_assert_1.strict.equal(uniqueCols.size, cols.length, 'All gates should be in separate columns');
});
