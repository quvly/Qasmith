"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || (function () {
    var ownKeys = function(o) {
        ownKeys = Object.getOwnPropertyNames || function (o) {
            var ar = [];
            for (var k in o) if (Object.prototype.hasOwnProperty.call(o, k)) ar[ar.length] = k;
            return ar;
        };
        return ownKeys(o);
    };
    return function (mod) {
        if (mod && mod.__esModule) return mod;
        var result = {};
        if (mod != null) for (var k = ownKeys(mod), i = 0; i < k.length; i++) if (k[i] !== "default") __createBinding(result, mod, k[i]);
        __setModuleDefault(result, mod);
        return result;
    };
})();
Object.defineProperty(exports, "__esModule", { value: true });
exports.CircuitVisualizer = void 0;
const vscode = __importStar(require("vscode"));
const circuitLayout_1 = require("./circuitLayout");
class CircuitVisualizer {
    _extensionUri;
    static viewType = 'qasmith.circuitView';
    _view;
    _debounceTimer;
    constructor(_extensionUri) {
        this._extensionUri = _extensionUri;
    }
    resolveWebviewView(webviewView, _context, _token) {
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
                    if (this._debounceTimer)
                        clearTimeout(this._debounceTimer);
                    this._debounceTimer = setTimeout(() => this._updateCircuit(event.document), 300);
                }
            }
        });
        const editor = vscode.window.activeTextEditor;
        if (editor && editor.document.languageId === 'qasmith') {
            this._updateCircuit(editor.document);
        }
        else {
            webviewView.webview.html = this._getEmptyHtml();
        }
    }
    _updateCircuit(document) {
        if (!this._view)
            return;
        const circuit = (0, circuitLayout_1.parseCircuit)(document.getText());
        const layout = (0, circuitLayout_1.layoutCircuit)(circuit);
        const svg = (0, circuitLayout_1.renderCircuit)(circuit, layout);
        this._view.webview.html = svg ? this._getHtml(svg) : this._getEmptyHtml();
    }
    _getEmptyHtml() {
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
    _getHtml(svg) {
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
    dispose() {
        if (this._debounceTimer)
            clearTimeout(this._debounceTimer);
    }
}
exports.CircuitVisualizer = CircuitVisualizer;
