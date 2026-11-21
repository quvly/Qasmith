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
exports.registerCommands = registerCommands;
const vscode = __importStar(require("vscode"));
const path = __importStar(require("node:path"));
const node_os_1 = require("node:os");
const promises_1 = require("node:fs/promises");
const compiler_1 = require("./compiler");
const languageClient_1 = require("./languageClient");
function settings(operation, config) {
    const usesSimulation = operation === 'run' || operation === 'optimize';
    const mode = config.get('optimize.evaluationMode', 'exact');
    if (operation === 'optimize' && mode !== 'exact' && mode !== 'sampled')
        throw new Error('Qasmith evaluation mode must be exact or sampled.');
    return {
        shots: usesSimulation ? (0, compiler_1.integer)(config.get('run.shots', 1000), 1, 1000000, 'Shots') : 1000,
        seed: usesSimulation ? (0, compiler_1.integer)(config.get('run.seed', 42), Number.MIN_SAFE_INTEGER, Number.MAX_SAFE_INTEGER, 'Seed') : 42,
        evaluations: operation === 'optimize' ? (0, compiler_1.integer)(config.get('optimize.maxEvaluations', 200), 1, 100000, 'Maximum evaluations') : 200,
        restarts: operation === 'optimize' ? (0, compiler_1.integer)(config.get('optimize.restarts', 3), 0, 64, 'Restarts') : 3,
        mode: mode === 'sampled' ? 'sampled' : 'exact',
    };
}
async function saveSources(source) {
    const dirty = vscode.workspace.textDocuments.filter(doc => doc.isDirty &&
        (doc === source || doc.languageId === 'qasmith' || path.basename(doc.fileName) === 'qasmith.json'));
    if (dirty.length && await vscode.window.showWarningMessage('Save modified Qasmith files before running the compiler?', { modal: true }, 'Save and Continue') !== 'Save and Continue')
        return false;
    for (const doc of dirty) {
        if (!await doc.save() || doc.isDirty)
            throw new Error(`Could not save ${doc.fileName}.`);
    }
    if (source.uri.scheme !== 'file')
        throw new Error('Save the entry file with a .smith extension, then retry.');
    return true;
}
async function checkDestination(destination) {
    try {
        const entry = await (0, promises_1.lstat)(destination);
        if (!entry.isFile())
            throw new Error('Choose a regular .qasm output file.');
        return await vscode.window.showWarningMessage(`Replace ${destination}?`, { modal: true }, 'Replace') === 'Replace';
    }
    catch (error) {
        if (error.code === 'ENOENT')
            return true;
        throw error;
    }
}
function registerCommands(context, output, server) {
    let running = false;
    async function execute(operation, preview = false) {
        if (running) {
            void vscode.window.showInformationMessage('A Qasmith command is already running.');
            return false;
        }
        running = true;
        let temporary;
        try {
            const document = vscode.window.activeTextEditor?.document;
            if (!document || document.languageId !== 'qasmith')
                throw new Error('Open a .smith entry file first.');
            const compiler = await (0, languageClient_1.executable)();
            if (!await saveSources(document))
                return false;
            if (document.isDirty)
                throw new Error('The entry file changed. Retry after saving.');
            const source = document.uri.fsPath;
            const cwd = path.dirname(source);
            const config = vscode.workspace.getConfiguration('qasmith', document.uri);
            const runSettings = settings(operation, config);
            let destination;
            if (preview) {
                temporary = await (0, promises_1.mkdtemp)(path.join((0, node_os_1.tmpdir)(), 'qasmith-preview-'));
                destination = path.join(temporary, 'preview.qasm');
            }
            else if (operation === 'compile') {
                const configured = config.get('compile.outputPath', '').trim();
                destination = configured ? path.resolve(cwd, configured) : path.join(cwd, path.basename(source, path.extname(source)) + '.qasm');
                if (path.extname(destination) !== '.qasm')
                    throw new Error('The Qasmith output path must end in .qasm.');
                if (!await checkDestination(destination))
                    return false;
            }
            const args = (0, compiler_1.compilerArgs)(operation, source, runSettings, destination);
            output.appendLine(`\nQasmith ${preview ? 'preview' : operation} — ${source}`);
            if (config.get('verbose', false))
                output.appendLine(JSON.stringify([compiler, ...args]));
            if (!preview)
                output.show(true);
            const code = await vscode.window.withProgress({
                location: vscode.ProgressLocation.Notification, title: `Qasmith: ${preview ? 'OpenQASM preview' : operation}`, cancellable: true,
            }, async (_progress, token) => {
                const controller = new AbortController();
                if (token.isCancellationRequested)
                    controller.abort();
                const cancellation = token.onCancellationRequested(() => controller.abort());
                try {
                    if (vscode.workspace.textDocuments.some(doc => doc.isDirty && (doc === document ||
                        doc.languageId === 'qasmith' || path.basename(doc.fileName) === 'qasmith.json'))) {
                        throw new Error('A Qasmith file changed after saving. Retry the command.');
                    }
                    return await (0, compiler_1.runCLI)(compiler, args, cwd, controller.signal, text => output.append(text));
                }
                finally {
                    cancellation.dispose();
                }
            });
            output.appendLine(`Qasmith exited with status ${code}.`);
            if (code !== 0)
                throw new Error(`Qasmith ${operation} failed (exit ${code}). See the Qasmith output channel.`);
            if (preview && destination) {
                const content = await (0, promises_1.readFile)(destination, 'utf8');
                const qasm = await vscode.workspace.openTextDocument({ language: 'qasmith-openqasm', content });
                await vscode.window.showTextDocument(qasm, { viewColumn: vscode.ViewColumn.Beside, preview: true, preserveFocus: true });
            }
            else if (destination) {
                output.appendLine(`Wrote ${destination}`);
            }
            return true;
        }
        catch (error) {
            const message = error instanceof Error ? error.message : String(error);
            output.appendLine(message);
            void vscode.window.showWarningMessage(message);
            return false;
        }
        finally {
            try {
                if (temporary)
                    await (0, promises_1.rm)(temporary, { recursive: true, force: true });
            }
            catch (error) {
                output.appendLine(String(error));
            }
            running = false;
        }
    }
    const operations = ['check', 'compile', 'run', 'optimize'];
    for (const operation of operations) {
        context.subscriptions.push(vscode.commands.registerCommand(`qasmith.${operation}`, () => execute(operation)));
    }
    context.subscriptions.push(vscode.commands.registerCommand('qasmith.preview', () => execute('compile', true)), vscode.commands.registerCommand('qasmith.restart', () => server.restart()), vscode.commands.registerCommand('qasmith.output', () => output.show(true)));
}
