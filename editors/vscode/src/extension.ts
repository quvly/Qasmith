import * as vscode from 'vscode';
import { registerCommands } from './commands';
import { Server } from './languageClient';
import { CircuitVisualizer } from './circuitVisualizer';

let server: Server | undefined;
let visualizer: CircuitVisualizer | undefined;

export function activate(context: vscode.ExtensionContext): void {
    const output = vscode.window.createOutputChannel('Qasmith', { log: true });
    const instance = new Server(output);
    server = instance;
    registerCommands(context, output, instance);

    // Register circuit visualizer
    const circuitViz = new CircuitVisualizer(context.extensionUri);
    visualizer = circuitViz;
    context.subscriptions.push(
        vscode.window.registerWebviewViewProvider(CircuitVisualizer.viewType, circuitViz)
    );

    context.subscriptions.push(
        output,
        vscode.workspace.onDidGrantWorkspaceTrust(() => void instance.restart()),
        vscode.workspace.onDidChangeConfiguration(event => {
            if (['executablePath', 'lsp', 'diagnostics'].some(key => event.affectsConfiguration(`qasmith.${key}`))) {
                void instance.restart();
            }
        }),
    );
    void instance.restart();
}

export async function deactivate(): Promise<void> {
    visualizer?.dispose();
    await server?.dispose();
    server = undefined;
}
