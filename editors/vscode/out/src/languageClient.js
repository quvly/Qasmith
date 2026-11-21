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
exports.Server = void 0;
exports.executable = executable;
const vscode = __importStar(require("vscode"));
const node_1 = require("vscode-languageclient/node");
const compiler_1 = require("./compiler");
async function executable() {
    if (!vscode.workspace.isTrusted)
        throw new Error('Trust this workspace before launching Qasmith.');
    const config = vscode.workspace.getConfiguration('qasmith').inspect('executablePath');
    return (0, compiler_1.resolveCompiler)(config?.globalValue ?? '', process.env, vscode.workspace.workspaceFolders?.filter(folder => folder.uri.scheme === 'file').map(folder => folder.uri.fsPath) ?? []);
}
class Server {
    output;
    client;
    missingNotified = false;
    pending = Promise.resolve();
    closed = false;
    constructor(output) {
        this.output = output;
    }
    restart() {
        this.pending = this.pending.then(async () => {
            await this.stop();
            if (!this.closed)
                await this.start();
        }).catch(error => this.report(error));
        return this.pending;
    }
    async dispose() {
        this.closed = true;
        await this.pending;
        await this.stop();
    }
    async stop() {
        const client = this.client;
        this.client = undefined;
        if (client)
            await client.dispose();
    }
    async start() {
        const config = vscode.workspace.getConfiguration('qasmith');
        if (!vscode.workspace.isTrusted || !config.get('lsp.enabled', true))
            return;
        const debounceMs = (0, compiler_1.integer)(config.get('diagnostics.debounceMs', 200), 0, 2000, 'Diagnostics debounce');
        const command = await executable();
        const watcher = vscode.workspace.createFileSystemWatcher('**/{*.smith,qasmith.json}');
        const client = new node_1.LanguageClient('qasmith', 'Qasmith', { command, args: ['lsp', '--stdio'] }, {
            documentSelector: [{ scheme: 'file', language: 'qasmith' }],
            outputChannel: this.output,
            traceOutputChannel: this.output,
            synchronize: { fileEvents: watcher },
            initializationOptions: {
                diagnostics: {
                    enabled: config.get('diagnostics.enabled', true),
                    debounceMs,
                },
            },
            errorHandler: {
                error: () => ({ action: node_1.ErrorAction.Shutdown }),
                closed: () => ({ action: node_1.CloseAction.DoNotRestart }),
            },
        });
        this.client = client;
        client.onDidChangeState(({ newState }) => {
            if (newState === node_1.State.Stopped) {
                watcher.dispose();
                if (this.client === client)
                    this.report(new Error('Qasmith language server stopped. Use Qasmith: Restart Language Server to retry.'));
            }
        });
        try {
            await client.start();
            const trace = config.get('lsp.trace', 'off');
            await client.setTrace(trace === 'verbose' ? node_1.Trace.Verbose : trace === 'messages' ? node_1.Trace.Messages : node_1.Trace.Off);
            this.missingNotified = false;
        }
        catch (error) {
            watcher.dispose();
            await this.stop().catch(stopError => this.output.appendLine(String(stopError)));
            throw error;
        }
    }
    report(error) {
        const message = error instanceof Error ? error.message : String(error);
        this.output.appendLine(message);
        if (!this.missingNotified && !this.closed) {
            this.missingNotified = true;
            void vscode.window.showWarningMessage(message);
        }
    }
}
exports.Server = Server;
