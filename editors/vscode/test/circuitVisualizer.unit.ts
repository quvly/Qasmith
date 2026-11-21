import { strict as assert } from 'node:assert';
import { test } from 'node:test';
import { parseCircuit, layoutCircuit, renderCircuit, GateType } from '../src/circuitLayout';

test('parseCircuit: Hadamard followed by CNOT', () => {
  const code = `
    let q0 = qubit();
    let q1 = qubit();
    h(q0);
    cx(q0, q1);
  `;

  const circuit = parseCircuit(code);

  assert.equal(circuit.qubits, 2, 'Should have 2 qubits');
  assert.equal(circuit.operations.length, 2, 'Should have 2 operations');

  // First operation: H on q0
  const h = circuit.operations[0];
  assert.equal(h.name, 'H');
  assert.equal(h.type, GateType.SingleQubit);
  assert.deepEqual(h.qubits, [0]);

  // Second operation: CX with control=q0, target=q1
  const cx = circuit.operations[1];
  assert.equal(cx.name, 'CX');
  assert.equal(cx.type, GateType.Controlled);
  assert.deepEqual(cx.qubits, [0, 1]);
});

test('parseCircuit: CNOT with reversed control and target', () => {
  const code = `
    let a = qubit();
    let b = qubit();
    cx(b, a);
  `;

  const circuit = parseCircuit(code);
  const cx = circuit.operations[0];

  assert.equal(cx.name, 'CX');
  assert.equal(cx.type, GateType.Controlled);
  // Control should be b (q1), target should be a (q0)
  assert.deepEqual(cx.qubits, [1, 0]);
});

test('parseCircuit: Non-adjacent qubits', () => {
  const code = `
    let q0 = qubit();
    let q1 = qubit();
    let q2 = qubit();
    cx(q0, q2);
  `;

  const circuit = parseCircuit(code);
  const cx = circuit.operations[0];

  assert.equal(circuit.qubits, 3);
  assert.equal(cx.type, GateType.Controlled);
  // Control=q0 (0), target=q2 (2)
  assert.deepEqual(cx.qubits, [0, 2]);
});

test('layoutCircuit: H + CX column assignment', () => {
  const code = `
    let q0 = qubit();
    let q1 = qubit();
    h(q0);
    cx(q0, q1);
  `;

  const circuit = parseCircuit(code);
  const layout = layoutCircuit(circuit);

  assert.equal(layout.numColumns, 2, 'Should have 2 columns');
  assert.equal(layout.assignments.length, 2);

  // H should be in column 0
  assert.equal(layout.assignments[0].column, 0);
  assert.equal(layout.assignments[0].operation.name, 'H');

  // CX should be in column 1
  assert.equal(layout.assignments[1].column, 1);
  assert.equal(layout.assignments[1].operation.name, 'CX');
});

test('layoutCircuit: Multiple sequential CX gates', () => {
  const code = `
    let q0 = qubit();
    let q1 = qubit();
    cx(q0, q1);
    cx(q1, q0);
    cx(q0, q1);
  `;

  const circuit = parseCircuit(code);
  const layout = layoutCircuit(circuit);

  // Each CX should get its own column
  assert.equal(layout.numColumns, 3);
  assert.equal(layout.assignments[0].column, 0);
  assert.equal(layout.assignments[1].column, 1);
  assert.equal(layout.assignments[2].column, 2);
});

test('layoutCircuit: Interleaved single and multi-qubit gates', () => {
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

  const circuit = parseCircuit(code);
  const layout = layoutCircuit(circuit);

  // H(q0), H(q1), H(q2) can all be in col 0 (different qubits)
  // CX(q0, q1) must be in col 1 (conflicts with H on q0 and q1)
  // CX(q1, q2) must be in col 2 (conflicts with CX on q1)

  const h0 = layout.assignments.find(a => a.operation.name === 'H' && a.operation.qubits[0] === 0);
  const h1 = layout.assignments.find(a => a.operation.name === 'H' && a.operation.qubits[0] === 1);
  const cx01 = layout.assignments.find(a => a.operation.name === 'CX' && a.operation.qubits[1] === 1);
  const h2 = layout.assignments.find(a => a.operation.name === 'H' && a.operation.qubits[0] === 2);
  const cx12 = layout.assignments.find(a => a.operation.name === 'CX' && a.operation.qubits[1] === 2);

  assert.ok(h0 && h1);
  assert.equal(h0.column, 0, 'H(q0) in col 0');
  assert.equal(h1.column, 0, 'H(q1) in col 0');

  assert.ok(cx01);
  assert.equal(cx01.column, 1, 'CX(q0,q1) in col 1');

  assert.ok(h2);
  assert.equal(h2.column, 0, 'H(q2) in col 0 (parallel with H gates)');

  assert.ok(cx12);
  assert.equal(cx12.column, 2, 'CX(q1,q2) in col 2');
});

test('parseCircuit: Parametric gates', () => {
  const code = `
    let q = qubit();
    rx(pi/2, q);
    ry(3.14, q);
  `;

  const circuit = parseCircuit(code);
  assert.equal(circuit.operations.length, 2);

  const rx = circuit.operations[0];
  assert.equal(rx.name, 'RX');
  assert.ok(rx.params);
  assert.ok(Math.abs(rx.params[0] - Math.PI / 2) < 0.001);

  const ry = circuit.operations[1];
  assert.equal(ry.name, 'RY');
  assert.ok(ry.params);
  assert.ok(Math.abs(ry.params[0] - 3.14) < 0.001);
});

test('parseCircuit: Qubit register', () => {
  const code = `
    let q = qreg<3>();
    h(q[0]);
    cx(q[0], q[2]);
  `;

  const circuit = parseCircuit(code);
  assert.equal(circuit.qubits, 3);
  assert.equal(circuit.operations.length, 2);

  const cx = circuit.operations[1];
  assert.deepEqual(cx.qubits, [0, 2]);
});

test('parseCircuit: CZ and SWAP gates', () => {
  const code = `
    let q0 = qubit();
    let q1 = qubit();
    cz(q0, q1);
    swap(q0, q1);
  `;

  const circuit = parseCircuit(code);
  assert.equal(circuit.operations.length, 2);

  assert.equal(circuit.operations[0].type, GateType.CZ);
  assert.equal(circuit.operations[1].type, GateType.Swap);
});

test('renderCircuit: Generates valid SVG', () => {
  const code = `
    let q0 = qubit();
    let q1 = qubit();
    h(q0);
    cx(q0, q1);
  `;

  const circuit = parseCircuit(code);
  const layout = layoutCircuit(circuit);
  const svg = renderCircuit(circuit, layout);

  assert.ok(svg.includes('<svg'), 'Should contain SVG tag');
  assert.ok(svg.includes('</svg>'), 'Should have closing SVG tag');
  assert.ok(svg.includes('q0'), 'Should contain qubit labels');
  assert.ok(svg.includes('q1'), 'Should contain qubit labels');
  assert.ok(svg.includes('>H<'), 'Should contain H gate label');
  // CX should render as control dot + target circle, not as text
  assert.ok(svg.includes('circle'), 'Should contain circle elements for CNOT');
});

test('renderCircuit: Empty circuit', () => {
  const code = `
    let q0 = qubit();
  `;

  const circuit = parseCircuit(code);
  const layout = layoutCircuit(circuit);
  const svg = renderCircuit(circuit, layout);

  assert.equal(svg, '', 'Empty circuit should return empty string');
});

test('layoutCircuit: Non-adjacent qubit operations', () => {
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

  const circuit = parseCircuit(code);
  const layout = layoutCircuit(circuit);

  // H(q0), H(q1), H(q2) can all be in col 0
  // CX(q0, q3) must be in col 1
  const hOps = layout.assignments.filter(a => a.operation.name === 'H');
  const cx = layout.assignments.find(a => a.operation.name === 'CX');

  assert.equal(hOps.length, 3);
  for (const h of hOps) {
    assert.equal(h.column, 0, 'All H gates in col 0');
  }

  assert.ok(cx);
  assert.equal(cx.column, 1, 'CX in col 1');
  assert.deepEqual(cx.operation.qubits, [0, 3], 'CX spans non-adjacent qubits');
});

test('parseCircuit: Ignore non-gate function calls', () => {
  const code = `
    let q = qubit();
    measure(q);
    h(q);
  `;

  const circuit = parseCircuit(code);
  assert.equal(circuit.operations.length, 1);
  assert.equal(circuit.operations[0].name, 'H');
});

test('parseCircuit: CNOT alias', () => {
  const code = `
    let q0 = qubit();
    let q1 = qubit();
    cnot(q0, q1);
  `;

  const circuit = parseCircuit(code);
  const gate = circuit.operations[0];

  assert.equal(gate.name, 'CNOT');
  assert.equal(gate.type, GateType.Controlled);
});
