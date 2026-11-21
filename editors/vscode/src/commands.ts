import * as vscode from 'vscode';
import * as path from 'node:path';
import { tmpdir } from 'node:os';
import { mkdtemp, readFile, rm, lstat } from 'node:fs/promises';
import { compilerArgs, integer, Operation, runCLI, Settings } from './compiler';
import { executable, Server } from './languageClient';

function settings(operation: Operation, config: vscode.WorkspaceConfiguration): Settings {
    const usesSimulation = operation === 'run' || operation === 'optimize';
    const mode = config.get<string>('optimize.evaluationMode', 'exact');
    if (operation === 'optimize' && mode !== 'exact' && mode !== 'sampled') throw new Error('Qasmith evaluation mode must be exact or sampled.');
    return {
        shots: usesSimulation ? integer(config.get('run.shots', 1000), 1, 1000000, 'Shots') : 1000,
        seed: usesSimulation ? integer(config.get('run.seed', 42), Number.MIN_SAFE_INTEGER, Number.MAX_SAFE_INTEGER, 'Seed') : 42,
        evaluations: operation === 'optimize' ? integer(config.get('optimize.maxEvaluations', 200), 1, 100000, 'Maximum evaluations') : 200,
        restarts: operation === 'optimize' ? integer(config.get('optimize.restarts', 3), 0, 64, 'Restarts') : 3,
        mode: mode === 'sampled' ? 'sampled' : 'exact',
    };
}

async function saveSources(source: vscode.TextDocument): Promise<boolean> {
    const dirty = vscode.workspace.textDocuments.filter(doc => doc.isDirty &&
        (doc === source || doc.languageId === 'qasmith' || path.basename(doc.fileName) === 'qasmith.json'));
    if (dirty.length && await vscode.window.showWarningMessage(
        'Save modified Qasmith files before running the compiler?', { modal: true }, 'Save and Continue'
    ) !== 'Save and Continue') return false;
    for (const doc of dirty) {
        if (!await doc.save() || doc.isDirty) throw new Error(`Could not save ${doc.fileName}.`);
    }
    if (source.uri.scheme !== 'file') throw new Error('Save the entry file with a .smith extension, then retry.');
    return true;
}

async function checkDestination(destination: string): Promise<boolean> {
    try {
        const entry = await lstat(destination);
        if (!entry.isFile()) throw new Error('Choose a regular .qasm output file.');
        return await vscode.window.showWarningMessage(`Replace ${destination}?`, { modal: true }, 'Replace') === 'Replace';
    } catch (error) {
        if ((error as NodeJS.ErrnoException).code === 'ENOENT') return true;
        throw error;
    }
}

export function registerCommands(context: vscode.ExtensionContext, output: vscode.LogOutputChannel, server: Server): void {
    let running = false;
    async function execute(operation: Operation, preview = false): Promise<boolean> {
        if (running) {
            void vscode.window.showInformationMessage('A Qasmith command is already running.');
            return false;
        }
        running = true;
        let temporary: string | undefined;
        try {
            const document = vscode.window.activeTextEditor?.document;
            if (!document || document.languageId !== 'qasmith') throw new Error('Open a .smith entry file first.');
            const compiler = await executable();
            if (!await saveSources(document)) return false;
            if (document.isDirty) throw new Error('The entry file changed. Retry after saving.');
            const source = document.uri.fsPath;
            const cwd = path.dirname(source);
            const config = vscode.workspace.getConfiguration('qasmith', document.uri);
            const runSettings = settings(operation, config);
            let destination: string | undefined;
            if (preview) {
                temporary = await mkdtemp(path.join(tmpdir(), 'qasmith-preview-'));
                destination = path.join(temporary, 'preview.qasm');
            } else if (operation === 'compile') {
                const configured = config.get('compile.outputPath', '').trim();
                destination = configured ? path.resolve(cwd, configured) : path.join(cwd, path.basename(source, path.extname(source)) + '.qasm');
                if (path.extname(destination) !== '.qasm') throw new Error('The Qasmith output path must end in .qasm.');
                if (!await checkDestination(destination)) return false;
            }
            const args = compilerArgs(operation, source, runSettings, destination);
            output.appendLine(`\nQasmith ${preview ? 'preview' : operation} — ${source}`);
            if (config.get('verbose', false)) output.appendLine(JSON.stringify([compiler, ...args]));
            if (!preview) output.show(true);
            const code = await vscode.window.withProgress({
                location: vscode.ProgressLocation.Notification, title: `Qasmith: ${preview ? 'OpenQASM preview' : operation}`, cancellable: true,
            }, async (_progress, token) => {
                const controller = new AbortController();
                if (token.isCancellationRequested) controller.abort();
                const cancellation = token.onCancellationRequested(() => controller.abort());
                try {
                    if (vscode.workspace.textDocuments.some(doc => doc.isDirty && (doc === document ||
                        doc.languageId === 'qasmith' || path.basename(doc.fileName) === 'qasmith.json'))) {
                        throw new Error('A Qasmith file changed after saving. Retry the command.');
                    }
                    return await runCLI(compiler, args, cwd, controller.signal, text => output.append(text));
                } finally {
                    cancellation.dispose();
                }
            });
            output.appendLine(`Qasmith exited with status ${code}.`);
            if (code !== 0) throw new Error(`Qasmith ${operation} failed (exit ${code}). See the Qasmith output channel.`);
            if (preview && destination) {
                const content = await readFile(destination, 'utf8');
                const qasm = await vscode.workspace.openTextDocument({ language: 'qasmith-openqasm', content });
                await vscode.window.showTextDocument(qasm, { viewColumn: vscode.ViewColumn.Beside, preview: true, preserveFocus: true });
            } else if (destination) {
                output.appendLine(`Wrote ${destination}`);
            }
            return true;
        } catch (error) {
            const message = error instanceof Error ? error.message : String(error);
            output.appendLine(message);
            void vscode.window.showWarningMessage(message);
            return false;
        } finally {
            try {
                if (temporary) await rm(temporary, { recursive: true, force: true });
            } catch (error) {
                output.appendLine(String(error));
            }
            running = false;
        }
    }
    const operations: Operation[] = ['check', 'compile', 'run', 'optimize'];
    for (const operation of operations) {
        context.subscriptions.push(vscode.commands.registerCommand(`qasmith.${operation}`, () => execute(operation)));
    }
    context.subscriptions.push(
        vscode.commands.registerCommand('qasmith.preview', () => execute('compile', true)),
        vscode.commands.registerCommand('qasmith.restart', () => server.restart()),
        vscode.commands.registerCommand('qasmith.output', () => output.show(true)),
    );
}
