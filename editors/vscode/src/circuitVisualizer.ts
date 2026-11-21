import * as vscode from 'vscode';
import { parseCircuit, layoutCircuit, renderCircuit } from './circuitLayout';

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

    vscode.window.onDidChangeActiveTextEditor(editor => {
      if (editor && editor.document.languageId === 'qasmith') {
        this._updateCircuit(editor.document);
      }
    });

    vscode.workspace.onDidChangeTextDocument(event => {
      if (event.document.languageId === 'qasmith') {
        const editor = vscode.window.activeTextEditor;
        if (editor && editor.document === event.document) {
          if (this._debounceTimer) clearTimeout(this._debounceTimer);
          this._debounceTimer = setTimeout(() => this._updateCircuit(event.document), 300);
        }
      }
    });

    const editor = vscode.window.activeTextEditor;
    if (editor && editor.document.languageId === 'qasmith') {
      this._updateCircuit(editor.document);
    } else {
      webviewView.webview.html = this._getEmptyHtml();
    }
  }

  private _updateCircuit(document: vscode.TextDocument) {
    if (!this._view) return;

    try {
      const circuit = parseCircuit(document.getText());
      
      if (circuit.qubits === 0) {
        this._view.webview.html = this._getNoQubitsHtml();
        return;
      }
      
      if (circuit.operations.length === 0) {
        this._view.webview.html = this._getNoGatesHtml();
        return;
      }
      
      const layout = layoutCircuit(circuit);
      const svg = renderCircuit(circuit, layout);
      this._view.webview.html = this._getHtml(svg);
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      this._view.webview.html = this._getErrorHtml(message);
    }
  }

  private _getEmptyHtml(): string {
    return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <style>
    body { margin: 0; padding: 20px; background: var(--vscode-editor-background); color: var(--vscode-editor-foreground); font-family: var(--vscode-font-family); }
    .empty { text-align: center; color: var(--vscode-descriptionForeground); margin-top: 40px; }
  </style>
</head>
<body>
  <div class="empty">
    <p>Open a .smith file to visualize the quantum circuit</p>
  </div>
</body>
</html>`;
  }

  private _getNoQubitsHtml(): string {
    return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <style>
    body { margin: 0; padding: 20px; background: var(--vscode-editor-background); color: var(--vscode-editor-foreground); font-family: var(--vscode-font-family); }
    .info { text-align: center; color: var(--vscode-descriptionForeground); margin-top: 40px; }
    code { background: var(--vscode-textBlockQuote-background); padding: 2px 6px; border-radius: 3px; }
  </style>
</head>
<body>
  <div class="info">
    <p>No qubit allocations found.</p>
    <p>Declare qubits with <code>let q = qubit()</code> or <code>let q = qreg(3)</code></p>
  </div>
</body>
</html>`;
  }

  private _getNoGatesHtml(): string {
    return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <style>
    body { margin: 0; padding: 20px; background: var(--vscode-editor-background); color: var(--vscode-editor-foreground); font-family: var(--vscode-font-family); }
    .info { text-align: center; color: var(--vscode-descriptionForeground); margin-top: 40px; }
  </style>
</head>
<body>
  <div class="info">
    <p>No gate operations found.</p>
    <p>Add quantum gates like <code>h(q)</code>, <code>x(q)</code>, or <code>cx(q0, q1)</code></p>
  </div>
</body>
</html>`;
  }

  private _getErrorHtml(message: string): string {
    const escaped = message.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
    return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <style>
    body { margin: 0; padding: 20px; background: var(--vscode-editor-background); color: var(--vscode-editor-foreground); font-family: var(--vscode-font-family); }
    .error { text-align: center; color: var(--vscode-errorForeground, #f44); margin-top: 40px; }
    .error pre { text-align: left; background: var(--vscode-textBlockQuote-background); padding: 10px; border-radius: 4px; overflow-x: auto; margin-top: 10px; font-size: 12px; }
  </style>
</head>
<body>
  <div class="error">
    <p>Circuit visualization error:</p>
    <pre>${escaped}</pre>
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
    body { margin: 0; padding: 10px; background: var(--vscode-editor-background); color: var(--vscode-editor-foreground); font-family: var(--vscode-font-family); overflow: auto; }
    .container { display: flex; justify-content: center; align-items: flex-start; }
    svg { max-width: 100%; height: auto; }
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
    if (this._debounceTimer) clearTimeout(this._debounceTimer);
  }
}
