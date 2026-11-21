import * as vscode from 'vscode';

interface Gate {
  name: string;
  qubits: number[];
  params?: number[];
  line: number;
}

interface Circuit {
  qubits: number;
  gates: Gate[];
}

export class CircuitVisualizer implements vscode.WebviewViewProvider {
  public static readonly viewType = 'qasmith.circuitView';
  private _view?: vscode.WebviewView;
  private _debounceTimer?: NodeJS.Timeout;

  constructor(private readonly _extensionUri: vscode.Uri) {}

  public resolveWebviewView(
    webviewView: vscode.WebviewView,
    _context: vscode.WebviewViewResolveContext,
    _token: vscode.CancellationToken,
  ) {
    this._view = webviewView;

    webviewView.webview.options = {
      enableScripts: true,
      localResourceRoots: [this._extensionUri]
    };

    // Listen to active editor changes
    vscode.window.onDidChangeActiveTextEditor(editor => {
      if (editor && editor.document.languageId === 'qasmith') {
        this._updateCircuit(editor.document);
      }
    });

    // Listen to document changes
    vscode.workspace.onDidChangeTextDocument(event => {
      if (event.document.languageId === 'qasmith') {
        const editor = vscode.window.activeTextEditor;
        if (editor && editor.document === event.document) {
          if (this._debounceTimer) {
            clearTimeout(this._debounceTimer);
          }
          this._debounceTimer = setTimeout(() => {
            this._updateCircuit(event.document);
          }, 300);
        }
      }
    });

    // Initial render
    const editor = vscode.window.activeTextEditor;
    if (editor && editor.document.languageId === 'qasmith') {
      this._updateCircuit(editor.document);
    } else {
      webviewView.webview.html = this._getEmptyHtml();
    }
  }

  private _updateCircuit(document: vscode.TextDocument) {
    if (!this._view) return;

    const circuit = this._parseCircuit(document.getText());
    const svg = this._renderCircuit(circuit);
    this._view.webview.html = this._getHtml(svg);
  }

  private _parseCircuit(code: string): Circuit {
    const gates: Gate[] = [];
    let maxQubit = 0;
    const lines = code.split('\n');

    // Track qubit allocations
    const qubitMap = new Map<string, number>();
    let nextQubit = 0;

    for (let i = 0; i < lines.length; i++) {
      const line = lines[i];
      const lineNum = i;

      // Parse qubit allocations: q0 = qubit()
      const qubitAllocMatch = line.match(/(\w+)\s*=\s*qubit\s*\(\s*\)/);
      if (qubitAllocMatch) {
        qubitMap.set(qubitAllocMatch[1], nextQubit++);
        continue;
      }

      // Parse qubit register: q = qreg<4>()
      const qregMatch = line.match(/(\w+)\s*=\s*qreg\s*<\s*(\d+)\s*>\s*\(\s*\)/);
      if (qregMatch) {
        const name = qregMatch[1];
        const size = parseInt(qregMatch[2]);
        for (let j = 0; j < size; j++) {
          qubitMap.set(`${name}[${j}]`, nextQubit++);
        }
        continue;
      }

      // Parse gate operations
      // Single qubit: h(q0), x(q[0]), ry(pi/2, q0)
      const singleQubitGate = line.match(/(\w+)\s*\(\s*(?:([^,]+),\s*)?(\w+(?:\[\d+\])?)\s*\)/);
      if (singleQubitGate) {
        const gateName = singleQubitGate[1];
        const param = singleQubitGate[2];
        const qubitRef = singleQubitGate[3];

        // Skip if it's not a gate (e.g., qubit(), qreg, measure, etc.)
        if (['qubit', 'qreg', 'measure', 'let', 'const', 'var'].includes(gateName)) {
          continue;
        }

        const qubitIdx = qubitMap.get(qubitRef);
        if (qubitIdx !== undefined) {
          const gate: Gate = {
            name: gateName.toUpperCase(),
            qubits: [qubitIdx],
            line: lineNum
          };

          if (param) {
            const paramValue = this._evalParam(param);
            if (paramValue !== undefined) {
              gate.params = [paramValue];
            }
          }

          gates.push(gate);
          maxQubit = Math.max(maxQubit, qubitIdx);
        }
        continue;
      }

      // Parse two-qubit gates: cx(q0, q1), cnot(q[0], q[1])
      const twoQubitGate = line.match(/(\w+)\s*\(\s*(\w+(?:\[\d+\])?)\s*,\s*(\w+(?:\[\d+\])?)\s*\)/);
      if (twoQubitGate) {
        const gateName = twoQubitGate[1];
        const qubitRef1 = twoQubitGate[2];
        const qubitRef2 = twoQubitGate[3];

        if (['cx', 'cnot', 'cz', 'swap'].includes(gateName.toLowerCase())) {
          const qubitIdx1 = qubitMap.get(qubitRef1);
          const qubitIdx2 = qubitMap.get(qubitRef2);

          if (qubitIdx1 !== undefined && qubitIdx2 !== undefined) {
            gates.push({
              name: gateName.toUpperCase(),
              qubits: [qubitIdx1, qubitIdx2],
              line: lineNum
            });
            maxQubit = Math.max(maxQubit, qubitIdx1, qubitIdx2);
          }
        }
      }
    }

    return {
      qubits: maxQubit + 1,
      gates: gates.sort((a, b) => a.line - b.line)
    };
  }

  private _evalParam(param: string): number | undefined {
    const trimmed = param.trim();
    
    // Try to evaluate simple math expressions with pi
    try {
      // Replace pi with Math.PI
      const expr = trimmed.replace(/pi/g, 'Math.PI');
      const result = eval(expr);
      if (typeof result === 'number' && !isNaN(result)) {
        return result;
      }
    } catch {
      // If eval fails, return undefined
    }
    
    return undefined;
  }

  private _renderCircuit(circuit: Circuit): string {
    if (circuit.qubits === 0 || circuit.gates.length === 0) {
      return '<div class="empty">No quantum gates detected</div>';
    }

    const qubitSpacing = 60;
    const gateWidth = 50;
    const gateSpacing = 70;
    const leftMargin = 60;
    const topMargin = 40;

    const width = leftMargin + circuit.gates.length * gateSpacing + 40;
    const height = topMargin + circuit.qubits * qubitSpacing;

    let svg = `<svg width="${width}" height="${height}" xmlns="http://www.w3.org/2000/svg">`;
    
    // Style
    svg += `
      <style>
        .qubit-line { stroke: #666; stroke-width: 2; }
        .gate { fill: #4a9eff; stroke: #2563eb; stroke-width: 2; }
        .gate-text { fill: white; font-family: monospace; font-size: 14px; font-weight: bold; text-anchor: middle; }
        .qubit-label { fill: #333; font-family: monospace; font-size: 12px; }
        .control { fill: #333; }
        .connection { stroke: #333; stroke-width: 2; }
        .empty { color: #999; font-family: sans-serif; padding: 20px; text-align: center; }
      </style>
    `;

    // Draw qubit lines
    for (let i = 0; i < circuit.qubits; i++) {
      const y = topMargin + i * qubitSpacing + qubitSpacing / 2;
      svg += `<line x1="${leftMargin}" y1="${y}" x2="${width - 20}" y2="${y}" class="qubit-line"/>`;
      svg += `<text x="${leftMargin - 10}" y="${y + 4}" class="qubit-label" text-anchor="end">q${i}</text>`;
    }

    // Draw gates
    const qubitColumn = new Map<number, number>();
    circuit.gates.forEach((gate, idx) => {
      const col = idx;
      const x = leftMargin + col * gateSpacing + gateSpacing / 2;

      if (gate.qubits.length === 1) {
        // Single qubit gate
        const qubitIdx = gate.qubits[0];
        const y = topMargin + qubitIdx * qubitSpacing + qubitSpacing / 2;
        
        svg += `<rect x="${x - gateWidth/2}" y="${y - 20}" width="${gateWidth}" height="40" rx="4" class="gate"/>`;
        
        let label = gate.name;
        if (gate.params && gate.params.length > 0) {
          label += `(${gate.params[0].toFixed(2)})`;
        }
        svg += `<text x="${x}" y="${y + 5}" class="gate-text">${label}</text>`;
        
        qubitColumn.set(qubitIdx, col);
      } else if (gate.qubits.length === 2) {
        // Two qubit gate (controlled gate)
        const qubitIdx1 = gate.qubits[0];
        const qubitIdx2 = gate.qubits[1];
        const y1 = topMargin + qubitIdx1 * qubitSpacing + qubitSpacing / 2;
        const y2 = topMargin + qubitIdx2 * qubitSpacing + qubitSpacing / 2;

        // Draw connection line
        svg += `<line x1="${x}" y1="${y1}" x2="${x}" y2="${y2}" class="connection"/>`;
        
        // Draw control dot
        svg += `<circle cx="${x}" cy="${y1}" r="5" class="control"/>`;
        
        // Draw target gate
        svg += `<rect x="${x - gateWidth/2}" y="${y2 - 20}" width="${gateWidth}" height="40" rx="4" class="gate"/>`;
        svg += `<text x="${x}" y="${y2 + 5}" class="gate-text">${gate.name}</text>`;
        
        qubitColumn.set(qubitIdx1, col);
        qubitColumn.set(qubitIdx2, col);
      }
    });

    svg += '</svg>';
    return svg;
  }

  private _getEmptyHtml(): string {
    return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <style>
    body {
      margin: 0;
      padding: 20px;
      background: var(--vscode-editor-background);
      color: var(--vscode-editor-foreground);
      font-family: var(--vscode-font-family);
    }
    .empty {
      text-align: center;
      color: var(--vscode-descriptionForeground);
      margin-top: 40px;
    }
  </style>
</head>
<body>
  <div class="empty">
    <p>Open a .smith file to visualize the quantum circuit</p>
  </div>
</body>
</html>`;
  }

  private _getHtml(svg: string): string {
    return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Circuit Visualization</title>
  <style>
    body {
      margin: 0;
      padding: 10px;
      background: var(--vscode-editor-background);
      color: var(--vscode-editor-foreground);
      font-family: var(--vscode-font-family);
      overflow: auto;
    }
    .container {
      display: flex;
      justify-content: center;
      align-items: flex-start;
    }
    svg {
      max-width: 100%;
      height: auto;
    }
  </style>
</head>
<body>
  <div class="container">
    ${svg}
  </div>
</body>
</html>`;
  }

  public dispose() {
    if (this._debounceTimer) {
      clearTimeout(this._debounceTimer);
    }
  }
}
