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
    const circuit = parseCircuit(document.getText());
    const layout = layoutCircuit(circuit);
    const svg = renderCircuit(circuit, layout);
    this._view.webview.html = svg ? this._getHtml(svg) : this._getEmptyHtml();
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
