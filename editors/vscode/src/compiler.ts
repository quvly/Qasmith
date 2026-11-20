import { spawn } from 'node:child_process';
import { access, realpath, stat } from 'node:fs/promises';
import { constants } from 'node:fs';
import * as path from 'node:path';

export async function resolveCompiler(configured: string, environment: NodeJS.ProcessEnv, workspaceRoots: readonly string[]): Promise<string> {
    const explicit = configured.trim();
    if (explicit && !path.isAbsolute(explicit)) {
        throw new Error('Qasmith: set qasmith.executablePath to an absolute path in User Settings.');
    }
    const roots = await Promise.all(workspaceRoots.map(root => realpath(root).catch(() => root)));
    const name = process.platform === 'win32' ? 'qasmith.exe' : 'qasmith';
    const candidates = explicit ? [explicit] : (environment.PATH ?? environment.Path ?? '')
        .split(path.delimiter).filter(directory => path.isAbsolute(directory)).map(directory => path.join(directory, name));
    for (const candidate of candidates) {
        try {
            const resolved = await realpath(candidate);
            if (!explicit && roots.some(root => {
                const relative = path.relative(root, resolved);
                return relative === '' || (!path.isAbsolute(relative) && relative !== '..' && !relative.startsWith('..' + path.sep));
            })) {
                continue;
            }
            if (!(await stat(resolved)).isFile()) {
                continue;
            }
            await access(resolved, constants.X_OK);
            return resolved;
        } catch {
            continue;
        }
    }
    throw new Error('Qasmith executable unavailable. Install Qasmith 0.9.0 or set qasmith.executablePath in User Settings.');
}

export type Operation = 'check' | 'compile' | 'run' | 'optimize';
export interface Settings { shots: number; seed: number; evaluations: number; restarts: number; mode: 'exact' | 'sampled' }

export function integer(value: unknown, min: number, max: number, name: string): number {
    if (typeof value !== 'number' || !Number.isSafeInteger(value) || value < min || value > max) {
        throw new Error(`${name} must be an integer between ${min} and ${max}.`);
    }
    return value;
}

export function compilerArgs(operation: Operation, source: string, settings: Settings, output?: string): string[] {
    const args: string[] = [operation];
    if (operation === 'compile' && output) args.push('-o', output);
    if (operation === 'run' || operation === 'optimize') args.push('--shots', String(settings.shots), '--seed', String(settings.seed));
    if (operation === 'optimize') args.push('--max-evals', String(settings.evaluations), '--restarts', String(settings.restarts), '--eval-mode', settings.mode);
    return [...args, source];
}

export function runCLI(executable: string, args: readonly string[], cwd: string, signal: AbortSignal, output: (text: string) => void): Promise<number> {
    return new Promise((resolve, reject) => {
        if (signal.aborted) {
            reject(new Error('Qasmith operation canceled.'));
            return;
        }
        const child = spawn(executable, args, { cwd, shell: false, windowsHide: true, stdio: ['ignore', 'pipe', 'pipe'] });
        let killTimer: ReturnType<typeof setTimeout> | undefined;
        const cancel = () => {
            child.kill();
            killTimer = setTimeout(() => child.kill('SIGKILL'), 2000);
            killTimer.unref();
        };
        signal.addEventListener('abort', cancel, { once: true });
        child.stdout.setEncoding('utf8').on('data', output);
        child.stderr.setEncoding('utf8').on('data', output);
        child.once('error', reject);
        child.once('close', (code, termination) => {
            signal.removeEventListener('abort', cancel);
            clearTimeout(killTimer);
            if (signal.aborted || termination) reject(new Error(signal.aborted ? 'Qasmith operation canceled.' : `Qasmith was terminated by ${termination}.`));
            else resolve(code ?? 1);
        });
    });
}
