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
(0, node_test_1.test)('parseCircuit: Ignore non-gate function calls', () => {
    const code = `
    let q = qubit();
    measure(q);
    h(q);
  `;
    const circuit = (0, circuitLayout_1.parseCircuit)(code);
    node_assert_1.strict.equal(circuit.operations.length, 1);
    node_assert_1.strict.equal(circuit.operations[0].name, 'H');
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
