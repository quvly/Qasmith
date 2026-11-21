"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.GateType = void 0;
exports.classifyGate = classifyGate;
exports.parseCircuit = parseCircuit;
exports.layoutCircuit = layoutCircuit;
exports.renderCircuit = renderCircuit;
var GateType;
(function (GateType) {
    GateType["SingleQubit"] = "single";
    GateType["Controlled"] = "controlled";
    GateType["CZ"] = "cz";
    GateType["Swap"] = "swap";
    GateType["MultiQubit"] = "multi";
    GateType["Measure"] = "measure";
})(GateType || (exports.GateType = GateType = {}));
function classifyGate(name, qubitCount) {
    if (qubitCount === 1)
        return GateType.SingleQubit;
    const lower = name.toLowerCase();
    if (lower === 'cx' || lower === 'cnot')
        return GateType.Controlled;
    if (lower === 'cz')
        return GateType.CZ;
    if (lower === 'swap')
        return GateType.Swap;
    return GateType.MultiQubit;
}
function evaluateMath(expr) {
    try {
        const cleaned = expr.trim().replace(/\bpi\b/g, String(Math.PI));
        if (!/^[\d+\-*/().\s eE]+$/.test(cleaned))
            return undefined;
        const result = Function(`"use strict"; return (${cleaned})`)();
        return typeof result === 'number' && isFinite(result) ? result : undefined;
    }
    catch {
        return undefined;
    }
}
function parseCircuit(code) {
    const operations = [];
    const qubitMap = new Map();
    const registerMap = new Map(); // track registers
    let nextQubit = 0;
    const lines = code.split('\n');
    const skipNames = new Set(['qubit', 'let', 'const', 'var', 'return', 'if', 'for', 'while', 'fn', 'pub', 'import']);
    for (const line of lines) {
        // Parse: let q = qubit() or q = qubit()
        const qubitAlloc = line.match(/(\w+)\s*=\s*qubit\s*\(\s*\)/);
        if (qubitAlloc) {
            qubitMap.set(qubitAlloc[1], nextQubit++);
            continue;
        }
        // Parse: let q = qreg<N>() or q = qreg<N>()
        const qregGeneric = line.match(/(\w+)\s*=\s*qreg\s*<\s*(\d+)\s*>\s*\(\s*\)/);
        if (qregGeneric) {
            const base = qregGeneric[1];
            const size = parseInt(qregGeneric[2]);
            const indices = [];
            for (let j = 0; j < size; j++) {
                const idx = nextQubit++;
                qubitMap.set(`${base}[${j}]`, idx);
                indices.push(idx);
            }
            registerMap.set(base, indices);
            continue;
        }
        // Parse: let q = qreg(n) - function call syntax
        const qregFunc = line.match(/(\w+)\s*=\s*qreg\s*\(\s*(\d+)\s*\)/);
        if (qregFunc) {
            const base = qregFunc[1];
            const size = parseInt(qregFunc[2]);
            const indices = [];
            for (let j = 0; j < size; j++) {
                const idx = nextQubit++;
                qubitMap.set(`${base}[${j}]`, idx);
                indices.push(idx);
            }
            registerMap.set(base, indices);
            continue;
        }
        // Parse measurement: measure(q) or measure(q[0]) or measure(q[0], q[1]) or return measure(...)
        const measureMatch = line.match(/(?:return\s+)?measure\s*\(\s*([^)]+)\s*\)/);
        if (measureMatch) {
            const argsStr = measureMatch[1].trim();
            // Split by comma to handle multiple arguments
            const args = argsStr.split(',').map(arg => arg.trim());
            for (const arg of args) {
                // Check if it's a register
                if (registerMap.has(arg)) {
                    const indices = registerMap.get(arg);
                    for (const idx of indices) {
                        operations.push({
                            name: 'MEASURE',
                            qubits: [idx],
                            type: GateType.Measure,
                        });
                    }
                }
                else if (qubitMap.has(arg)) {
                    // Single qubit measurement
                    operations.push({
                        name: 'MEASURE',
                        qubits: [qubitMap.get(arg)],
                        type: GateType.Measure,
                    });
                }
            }
            continue;
        }
        // Parse gate call: name(arg1) or name(arg1, arg2)
        const callMatch = line.match(/(\w+)\s*\(\s*([^()]+?)\s*(?:,\s*([^()]+?))?\s*\)/);
        if (!callMatch)
            continue;
        const gateName = callMatch[1];
        if (skipNames.has(gateName.toLowerCase()) || gateName.toLowerCase() === 'qreg')
            continue;
        const arg1 = callMatch[2].trim();
        const arg2 = callMatch[3]?.trim();
        const isQubitRef = (s) => qubitMap.has(s);
        const isNumeric = (s) => /^[\d+\-*/().\s eE]+$/.test(s.replace(/\bpi\b/gi, '3.14'));
        if (arg2 && isQubitRef(arg1) && isQubitRef(arg2)) {
            // Two-qubit gate: name(q1, q2)
            const q1 = qubitMap.get(arg1);
            const q2 = qubitMap.get(arg2);
            operations.push({
                name: gateName.toUpperCase(),
                qubits: [q1, q2],
                type: classifyGate(gateName, 2),
            });
        }
        else if (!arg2 && isQubitRef(arg1)) {
            // Single-qubit gate: name(q)
            operations.push({
                name: gateName.toUpperCase(),
                qubits: [qubitMap.get(arg1)],
                type: GateType.SingleQubit,
            });
        }
        else if (arg2 && isNumeric(arg1) && isQubitRef(arg2)) {
            // Parametric gate: name(param, q)
            const paramVal = evaluateMath(arg1);
            const op = {
                name: gateName.toUpperCase(),
                qubits: [qubitMap.get(arg2)],
                type: GateType.SingleQubit,
            };
            if (paramVal !== undefined)
                op.params = [paramVal];
            operations.push(op);
        }
    }
    return { qubits: nextQubit, operations };
}
function layoutCircuit(circuit) {
    if (circuit.operations.length === 0) {
        return { assignments: [], numColumns: 0, columnWidths: [], columnCenters: [] };
    }
    const nextAvailable = new Array(circuit.qubits).fill(0);
    const assignments = [];
    const columnGates = new Map();
    let maxCol = 0;
    for (const op of circuit.operations) {
        // Place in earliest column where all target qubits are free
        const col = Math.max(...op.qubits.map(q => nextAvailable[q]));
        assignments.push({ operation: op, column: col });
        // Track gates in each column
        if (!columnGates.has(col)) {
            columnGates.set(col, []);
        }
        columnGates.get(col).push(op);
        for (const q of op.qubits) {
            nextAvailable[q] = col + 1;
        }
        if (col > maxCol)
            maxCol = col;
    }
    const numColumns = maxCol + 1;
    const columnWidths = [];
    const minGap = 20; // minimum gap between columns
    // Calculate width for each column based on widest gate
    for (let col = 0; col < numColumns; col++) {
        const gates = columnGates.get(col) || [];
        let maxWidth = 44; // minimum gate width
        for (const op of gates) {
            const width = calculateGateWidth(op);
            if (width > maxWidth)
                maxWidth = width;
        }
        columnWidths.push(maxWidth + minGap);
    }
    // Calculate center positions for each column
    const columnCenters = [];
    let currentX = 0;
    for (let col = 0; col < numColumns; col++) {
        columnCenters.push(currentX + columnWidths[col] / 2);
        currentX += columnWidths[col];
    }
    return { assignments, numColumns, columnWidths, columnCenters };
}
function calculateGateWidth(op) {
    const minWidth = 44;
    if (op.type === GateType.Measure) {
        return 40; // measurement symbol width
    }
    let label = op.name;
    if (op.params && op.params.length > 0) {
        const formatted = formatAngle(op.params[0]);
        label += `(${formatted})`;
    }
    // Estimate width based on label length
    const estimatedWidth = label.length * 9 + 12;
    return Math.max(minWidth, estimatedWidth);
}
const DEFAULT_OPTIONS = {
    qubitSpacing: 60,
    leftMargin: 50,
    topMargin: 20,
    rightMargin: 20,
    bottomMargin: 20,
};
function renderCircuit(circuit, layout, opts) {
    const o = { ...DEFAULT_OPTIONS, ...opts };
    if (circuit.qubits === 0 || layout.numColumns === 0) {
        return '';
    }
    // Calculate total width based on all column widths
    const totalWidth = layout.columnWidths.reduce((sum, w) => sum + w, 0);
    const width = o.leftMargin + totalWidth + o.rightMargin;
    const height = o.topMargin + circuit.qubits * o.qubitSpacing + o.bottomMargin;
    const colX = (col) => o.leftMargin + layout.columnCenters[col];
    const qubitY = (q) => o.topMargin + q * o.qubitSpacing + o.qubitSpacing / 2;
    let svg = '';
    // Background
    svg += `<rect width="${width}" height="${height}" fill="var(--vscode-editor-background, #1e1e1e)" rx="4"/>`;
    // Qubit wires
    for (let q = 0; q < circuit.qubits; q++) {
        const y = qubitY(q);
        svg += `<line x1="${o.leftMargin - 10}" y1="${y}" x2="${width - o.rightMargin}" y2="${y}" stroke="var(--vscode-editor-foreground, #aaa)" stroke-width="1.5" stroke-opacity="0.4"/>`;
        svg += `<text x="${o.leftMargin - 16}" y="${y + 4}" fill="var(--vscode-editor-foreground, #aaa)" font-family="monospace" font-size="11" text-anchor="end">q${q}</text>`;
    }
    // Group assignments by column
    const columns = new Map();
    for (const a of layout.assignments) {
        const list = columns.get(a.column) ?? [];
        list.push(a);
        columns.set(a.column, list);
    }
    // Render each column
    for (const [col, colAssignments] of columns) {
        const x = colX(col);
        for (const { operation } of colAssignments) {
            svg += renderOperation(operation, x, qubitY);
        }
    }
    return `<svg width="${width}" height="${height}" xmlns="http://www.w3.org/2000/svg">${svg}</svg>`;
}
function renderOperation(op, x, qubitY) {
    switch (op.type) {
        case GateType.SingleQubit:
            return renderSingleQubitGate(op, x, qubitY(op.qubits[0]));
        case GateType.Controlled:
            return renderControlledGate(op, x, qubitY);
        case GateType.CZ:
            return renderCZGate(op, x, qubitY);
        case GateType.Swap:
            return renderSwapGate(op, x, qubitY);
        case GateType.MultiQubit:
            return renderMultiQubitGate(op, x, qubitY);
        case GateType.Measure:
            return renderMeasurement(op, x, qubitY(op.qubits[0]));
    }
}
function renderSingleQubitGate(op, x, y) {
    let label = op.name;
    if (op.params && op.params.length > 0) {
        const formatted = formatAngle(op.params[0]);
        label += `(${formatted})`;
    }
    const size = calculateGateWidth(op);
    const h = size / 2;
    const fontSize = label.length > 6 ? 10 : 13;
    return [
        `<rect x="${x - h}" y="${y - h}" width="${size}" height="${size}" rx="4" fill="var(--vscode-charts-blue, #4a9eff)" stroke="var(--vscode-charts-blue, #2563eb)" stroke-width="1.5"/>`,
        `<text x="${x}" y="${y + 4}" fill="white" font-family="monospace" font-size="${fontSize}" font-weight="600" text-anchor="middle">${escapeXml(label)}</text>`,
    ].join('');
}
function formatAngle(value) {
    const piRatio = value / Math.PI;
    if (Math.abs(piRatio - Math.round(piRatio * 4) / 4) < 0.001) {
        const n = Math.round(piRatio * 4);
        if (n === 0)
            return '0';
        if (n === 4)
            return 'π';
        if (n === -4)
            return '-π';
        if (n === 2)
            return 'π/2';
        if (n === -2)
            return '-π/2';
        if (n === 1)
            return 'π/4';
        if (n === -1)
            return '-π/4';
        if (n % 4 === 0)
            return `${n / 4}π`;
        return `${n}π/4`;
    }
    return value.toFixed(2);
}
function renderControlledGate(op, x, qubitY) {
    // CX/CNOT: first qubit = control (●), second qubit = target (⊕)
    const controlY = qubitY(op.qubits[0]);
    const targetY = qubitY(op.qubits[1]);
    const minY = Math.min(controlY, targetY);
    const maxY = Math.max(controlY, targetY);
    const targetR = 14;
    return [
        // Vertical connection line
        `<line x1="${x}" y1="${minY}" x2="${x}" y2="${maxY}" stroke="var(--vscode-editor-foreground, #aaa)" stroke-width="2"/>`,
        // Control dot
        `<circle cx="${x}" cy="${controlY}" r="5" fill="var(--vscode-editor-foreground, #aaa)"/>`,
        // Target circle (⊕)
        `<circle cx="${x}" cy="${targetY}" r="${targetR}" fill="var(--vscode-editor-background, #1e1e1e)" stroke="var(--vscode-editor-foreground, #aaa)" stroke-width="2"/>`,
        // Target vertical line (⊕ cross)
        `<line x1="${x}" y1="${targetY - targetR}" x2="${x}" y2="${targetY + targetR}" stroke="var(--vscode-editor-foreground, #aaa)" stroke-width="2"/>`,
        // Target horizontal line (⊕ cross)
        `<line x1="${x - targetR}" y1="${targetY}" x2="${x + targetR}" y2="${targetY}" stroke="var(--vscode-editor-foreground, #aaa)" stroke-width="2"/>`,
    ].join('');
}
function renderCZGate(op, x, qubitY) {
    const y1 = qubitY(op.qubits[0]);
    const y2 = qubitY(op.qubits[1]);
    const minY = Math.min(y1, y2);
    const maxY = Math.max(y1, y2);
    return [
        `<line x1="${x}" y1="${minY}" x2="${x}" y2="${maxY}" stroke="var(--vscode-editor-foreground, #aaa)" stroke-width="2"/>`,
        `<circle cx="${x}" cy="${y1}" r="5" fill="var(--vscode-editor-foreground, #aaa)"/>`,
        `<circle cx="${x}" cy="${y2}" r="5" fill="var(--vscode-editor-foreground, #aaa)"/>`,
    ].join('');
}
function renderSwapGate(op, x, qubitY) {
    const y1 = qubitY(op.qubits[0]);
    const y2 = qubitY(op.qubits[1]);
    const minY = Math.min(y1, y2);
    const maxY = Math.max(y1, y2);
    const s = 7; // half-size of ×
    function crossMark(cx, cy) {
        return [
            `<line x1="${cx - s}" y1="${cy - s}" x2="${cx + s}" y2="${cy + s}" stroke="var(--vscode-editor-foreground, #aaa)" stroke-width="2"/>`,
            `<line x1="${cx - s}" y1="${cy + s}" x2="${cx + s}" y2="${cy - s}" stroke="var(--vscode-editor-foreground, #aaa)" stroke-width="2"/>`,
        ].join('');
    }
    return [
        `<line x1="${x}" y1="${minY}" x2="${x}" y2="${maxY}" stroke="var(--vscode-editor-foreground, #aaa)" stroke-width="2"/>`,
        crossMark(x, y1),
        crossMark(x, y2),
    ].join('');
}
function renderMultiQubitGate(op, x, qubitY) {
    const ys = op.qubits.map(q => qubitY(q));
    const minY = Math.min(...ys);
    const maxY = Math.max(...ys);
    const size = calculateGateWidth(op);
    const halfSize = size / 2;
    let svg = '';
    // Vertical connection line
    svg += `<line x1="${x}" y1="${minY}" x2="${x}" y2="${maxY}" stroke="var(--vscode-charts-blue, #4a9eff)" stroke-width="2"/>`;
    // Rectangle on each qubit
    for (const y of ys) {
        svg += `<rect x="${x - halfSize}" y="${y - halfSize}" width="${size}" height="${size}" rx="4" fill="var(--vscode-charts-blue, #4a9eff)" stroke="var(--vscode-charts-blue, #2563eb)" stroke-width="1.5"/>`;
        svg += `<text x="${x}" y="${y + 4}" fill="white" font-family="monospace" font-size="13" font-weight="600" text-anchor="middle">${escapeXml(op.name)}</text>`;
    }
    return svg;
}
function renderMeasurement(op, x, y) {
    // Standard quantum measurement symbol: a meter with an arc and diagonal line
    const width = 36;
    const height = 36;
    const halfW = width / 2;
    const halfH = height / 2;
    return [
        // Background rectangle
        `<rect x="${x - halfW}" y="${y - halfH}" width="${width}" height="${height}" rx="4" fill="var(--vscode-editor-background, #1e1e1e)" stroke="var(--vscode-editor-foreground, #aaa)" stroke-width="1.5"/>`,
        // Arc (meter needle arc)
        `<path d="M ${x - 10} ${y + 8} Q ${x} ${y - 12} ${x + 10} ${y + 8}" fill="none" stroke="var(--vscode-editor-foreground, #aaa)" stroke-width="1.5"/>`,
        // Diagonal needle line
        `<line x1="${x}" y1="${y + 8}" x2="${x + 8}" y2="${y - 8}" stroke="var(--vscode-editor-foreground, #aaa)" stroke-width="1.5"/>`,
    ].join('');
}
function escapeXml(s) {
    return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}
