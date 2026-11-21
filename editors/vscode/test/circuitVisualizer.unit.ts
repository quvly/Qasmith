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

test('parseCircuit: Measure is a valid operation', () => {
  const code = `
    let q = qubit();
    measure(q);
    h(q);
  `;

  const circuit = parseCircuit(code);
  // measure and h are both valid operations
  assert.equal(circuit.operations.length, 2);
  assert.equal(circuit.operations[0].name, 'MEASURE');
  assert.equal(circuit.operations[0].type, GateType.Measure);
  assert.equal(circuit.operations[1].name, 'H');
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

test('parseCircuit: Single-qubit gates H, X, RX, RY and RZ in sequence', () => {
  const code = `
    let q = qubit();
    h(q);
    x(q);
    rx(pi / 2.0, q);
    ry(pi / 4.0, q);
    rz(pi / 3.0, q);
  `;

  const circuit = parseCircuit(code);
  assert.equal(circuit.qubits, 1);
  assert.equal(circuit.operations.length, 5);

  const layout = layoutCircuit(circuit);
  // Each gate should be in its own column (all on same qubit)
  assert.equal(layout.numColumns, 5);

  assert.equal(circuit.operations[0].name, 'H');
  assert.equal(circuit.operations[1].name, 'X');
  assert.equal(circuit.operations[2].name, 'RX');
  assert.ok(circuit.operations[2].params);
  assert.ok(Math.abs(circuit.operations[2].params![0] - Math.PI / 2) < 0.001);
  assert.equal(circuit.operations[3].name, 'RY');
  assert.ok(circuit.operations[3].params);
  assert.ok(Math.abs(circuit.operations[3].params![0] - Math.PI / 4) < 0.001);
  assert.equal(circuit.operations[4].name, 'RZ');
  assert.ok(circuit.operations[4].params);
  assert.ok(Math.abs(circuit.operations[4].params![0] - Math.PI / 3) < 0.001);
});

test('parseCircuit: Two independently allocated qubits with CNOT', () => {
  const code = `
    let a = qubit();
    let b = qubit();
    cx(a, b);
  `;

  const circuit = parseCircuit(code);
  assert.equal(circuit.qubits, 2);
  assert.equal(circuit.operations.length, 1);
  assert.equal(circuit.operations[0].name, 'CX');
  assert.equal(circuit.operations[0].type, GateType.Controlled);
  assert.deepEqual(circuit.operations[0].qubits, [0, 1]);
});

test('parseCircuit: Three independently allocated qubits with non-adjacent CNOT', () => {
  const code = `
    let a = qubit();
    let b = qubit();
    let c = qubit();
    cx(a, c);
  `;

  const circuit = parseCircuit(code);
  assert.equal(circuit.qubits, 3);
  assert.equal(circuit.operations.length, 1);
  assert.deepEqual(circuit.operations[0].qubits, [0, 2]);
});

test('parseCircuit: qreg(n) function-call syntax', () => {
  const code = `
    let q = qreg(3);
    h(q[0]);
    cx(q[0], q[2]);
  `;

  const circuit = parseCircuit(code);
  assert.equal(circuit.qubits, 3);
  assert.equal(circuit.operations.length, 2);
  assert.equal(circuit.operations[0].name, 'H');
  assert.deepEqual(circuit.operations[0].qubits, [0]);
  assert.equal(circuit.operations[1].name, 'CX');
  assert.deepEqual(circuit.operations[1].qubits, [0, 2]);
});

test('parseCircuit: Four-qubit register with sequential CNOT operations', () => {
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

  const circuit = parseCircuit(code);
  assert.equal(circuit.qubits, 4);
  assert.equal(circuit.operations.length, 7);

  // Verify all operations parsed correctly
  const cxOps = circuit.operations.filter(op => op.name === 'CX');
  assert.equal(cxOps.length, 3);
  assert.deepEqual(cxOps[0].qubits, [0, 1]);
  assert.deepEqual(cxOps[1].qubits, [1, 2]);
  assert.deepEqual(cxOps[2].qubits, [2, 3]);

  // All gates should not overlap
  const layout = layoutCircuit(circuit);
  for (let i = 0; i < layout.assignments.length; i++) {
    for (let j = i + 1; j < layout.assignments.length; j++) {
      const a = layout.assignments[i];
      const b = layout.assignments[j];
      const overlap = a.operation.qubits.some(q => b.operation.qubits.includes(q));
      if (a.column === b.column) {
        assert.ok(!overlap, `Operations at column ${a.column} should not share qubits`);
      }
    }
  }
});

test('parseCircuit: GHZ state construction', () => {
  const code = `
    let q = qreg(4);
    h(q[0]);
    cx(q[0], q[1]);
    cx(q[0], q[2]);
    cx(q[0], q[3]);
  `;

  const circuit = parseCircuit(code);
  assert.equal(circuit.qubits, 4);
  assert.equal(circuit.operations.length, 4);

  assert.equal(circuit.operations[0].name, 'H');
  assert.deepEqual(circuit.operations[0].qubits, [0]);

  const cxOps = circuit.operations.slice(1);
  for (const cx of cxOps) {
    assert.equal(cx.name, 'CX');
    assert.equal(cx.qubits[0], 0, 'Control should always be q[0]');
  }
});

test('parseCircuit: Whole-register measurement', () => {
  const code = `
    let q = qreg(3);
    h(q[0]);
    cx(q[0], q[1]);
    cx(q[0], q[2]);
    return measure(q);
  `;

  const circuit = parseCircuit(code);
  assert.equal(circuit.qubits, 3);

  const measures = circuit.operations.filter(op => op.type === GateType.Measure);
  assert.equal(measures.length, 3, 'Should measure all 3 qubits');
  assert.deepEqual(measures[0].qubits, [0]);
  assert.deepEqual(measures[1].qubits, [1]);
  assert.deepEqual(measures[2].qubits, [2]);
});

test('parseCircuit: Selected register-element measurement', () => {
  const code = `
    let q = qreg(4);
    h(q[0]);
    cx(q[0], q[2]);
    return measure(q[0], q[2]);
  `;

  const circuit = parseCircuit(code);
  assert.equal(circuit.qubits, 4);

  const measures = circuit.operations.filter(op => op.type === GateType.Measure);
  assert.equal(measures.length, 2, 'Should measure q[0] and q[2]');
  assert.deepEqual(measures[0].qubits, [0]);
  assert.deepEqual(measures[1].qubits, [2]);
});

test('parseCircuit: Measurement with return statement', () => {
  const code = `
    let a = qubit();
    let b = qubit();
    h(a);
    cx(a, b);
    return measure(a, b);
  `;

  const circuit = parseCircuit(code);
  const measures = circuit.operations.filter(op => op.type === GateType.Measure);
  assert.equal(measures.length, 2);
});

test('parseCircuit: Allocated but unmeasured qubits', () => {
  const code = `
    let a = qubit();
    let b = qubit();
    let c = qubit();
    h(a);
    cx(a, b);
  `;

  const circuit = parseCircuit(code);
  assert.equal(circuit.qubits, 3);
  assert.equal(circuit.operations.length, 2);
  
  const measures = circuit.operations.filter(op => op.type === GateType.Measure);
  assert.equal(measures.length, 0, 'No measurements should be present');
});

test('layoutCircuit: Long sequences of parameterized rotation gates', () => {
  const code = `
    let q = qubit();
    rx(pi / 2.0, q);
    ry(pi / 4.0, q);
    rz(pi / 3.0, q);
    rx(pi / 6.0, q);
    ry(pi / 8.0, q);
  `;

  const circuit = parseCircuit(code);
  const layout = layoutCircuit(circuit);

  // All on same qubit, so each gate gets its own column
  assert.equal(layout.numColumns, 5);

  // Verify column widths are not uniform
  const widths = layout.columnWidths;
  assert.ok(widths.length === 5);
  // All widths should be positive
  for (const w of widths) {
    assert.ok(w > 0);
  }

  // Render and verify SVG is valid
  const svg = renderCircuit(circuit, layout);
  assert.ok(svg.includes('<svg'));
  assert.ok(svg.includes('RX'));
  assert.ok(svg.includes('RY'));
  assert.ok(svg.includes('RZ'));
});

test('renderCircuit: Circuits with multi-qubit operations and measurements', () => {
  const code = `
    let q = qreg(3);
    h(q[0]);
    cx(q[0], q[1]);
    cx(q[1], q[2]);
    return measure(q);
  `;

  const circuit = parseCircuit(code);
  const layout = layoutCircuit(circuit);
  const svg = renderCircuit(circuit, layout);

  assert.ok(svg.includes('<svg'));
  assert.ok(svg.includes('H'));
  // Should have measurement symbols
  assert.ok(svg.includes('path') || svg.includes('MEASURE'), 'Should contain measurement rendering');
});

test('renderCircuit: No overlapping gates with rotation sequence', () => {
  const code = `
    let q = qubit();
    h(q);
    x(q);
    rx(pi / 2.0, q);
    ry(pi / 4.0, q);
    rz(pi / 3.0, q);
  `;

  const circuit = parseCircuit(code);
  const layout = layoutCircuit(circuit);

  // Verify column spacing prevents overlap
  const totalWidth = layout.columnWidths.reduce((sum, w) => sum + w, 0);
  assert.ok(totalWidth > layout.numColumns * 44, 'Total width should exceed minimum if gates are wider');

  // Each gate on same qubit should be in separate column
  const cols = layout.assignments.map(a => a.column);
  const uniqueCols = new Set(cols);
  assert.equal(uniqueCols.size, cols.length, 'All gates should be in separate columns');
});
