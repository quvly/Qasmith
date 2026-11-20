import * as vscode from 'vscode';
import { CloseAction, ErrorAction, LanguageClient, State, Trace } from 'vscode-languageclient/node';
import { integer, resolveCompiler } from './compiler';

export async function executable(): Promise<string> {
    if (!vscode.workspace.isTrusted) throw new Error('Trust this workspace before launching Qasmith.');
    const config = vscode.workspace.getConfiguration('qasmith').inspect<string>('executablePath');
    return resolveCompiler(config?.globalValue ?? '', process.env, vscode.workspace.workspaceFolders?.filter(folder => folder.uri.scheme === 'file').map(folder => folder.uri.fsPath) ?? []);
}

export class Server {
    private client?: LanguageClient;
    private missingNotified = false;
    private pending: Promise<void> = Promise.resolve();
    private closed = false;

    constructor(private readonly output: vscode.OutputChannel) {}

    restart(): Promise<void> {
        this.pending = this.pending.then(async () => {
            await this.stop();
            if (!this.closed) await this.start();
        }).catch(error => this.report(error));
        return this.pending;
    }

    async dispose(): Promise<void> {
        this.closed = true;
        await this.pending;
        await this.stop();
    }

    private async stop(): Promise<void> {
        const client = this.client;
        this.client = undefined;
        if (client) await client.dispose();
    }

    private async start(): Promise<void> {
        const config = vscode.workspace.getConfiguration('qasmith');
        if (!vscode.workspace.isTrusted || !config.get('lsp.enabled', true)) return;
        const debounceMs = integer(config.get('diagnostics.debounceMs', 200), 0, 2000, 'Diagnostics debounce');
        const command = await executable();
        const watcher = vscode.workspace.createFileSystemWatcher('**/{*.smith,qasmith.json}');
        const client = new LanguageClient('qasmith', 'Qasmith', { command, args: ['lsp', '--stdio'] }, {
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
                error: () => ({ action: ErrorAction.Shutdown }),
                closed: () => ({ action: CloseAction.DoNotRestart }),
            },
        });
        this.client = client;
        client.onDidChangeState(({newState}) => {
            if (newState === State.Stopped) {
                watcher.dispose();
                if (this.client === client) this.report(new Error('Qasmith language server stopped. Use Qasmith: Restart Language Server to retry.'));
            }
        });
        try {
            await client.start();
            const trace = config.get<string>('lsp.trace', 'off');
            await client.setTrace(trace === 'verbose' ? Trace.Verbose : trace === 'messages' ? Trace.Messages : Trace.Off);
            this.missingNotified = false;
        } catch (error) {
            watcher.dispose();
            await this.stop().catch(stopError => this.output.appendLine(String(stopError)));
            throw error;
        }
    }

    private report(error: unknown): void {
        const message = error instanceof Error ? error.message : String(error);
        this.output.appendLine(message);
        if (!this.missingNotified && !this.closed) {
            this.missingNotified = true;
            void vscode.window.showWarningMessage(message);
        }
    }
}
