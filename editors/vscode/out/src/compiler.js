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
exports.resolveCompiler = resolveCompiler;
exports.integer = integer;
exports.compilerArgs = compilerArgs;
exports.runCLI = runCLI;
const node_child_process_1 = require("node:child_process");
const promises_1 = require("node:fs/promises");
const node_fs_1 = require("node:fs");
const path = __importStar(require("node:path"));
async function resolveCompiler(configured, environment, workspaceRoots) {
    const explicit = configured.trim();
    if (explicit && !path.isAbsolute(explicit)) {
        throw new Error('Qasmith: set qasmith.executablePath to an absolute path in User Settings.');
    }
    const roots = await Promise.all(workspaceRoots.map(root => (0, promises_1.realpath)(root).catch(() => root)));
    const name = process.platform === 'win32' ? 'qasmith.exe' : 'qasmith';
    const candidates = explicit ? [explicit] : (environment.PATH ?? environment.Path ?? '')
        .split(path.delimiter).filter(directory => path.isAbsolute(directory)).map(directory => path.join(directory, name));
    for (const candidate of candidates) {
        try {
            const resolved = await (0, promises_1.realpath)(candidate);
            if (!explicit && roots.some(root => {
                const relative = path.relative(root, resolved);
                return relative === '' || (!path.isAbsolute(relative) && relative !== '..' && !relative.startsWith('..' + path.sep));
            })) {
                continue;
            }
            if (!(await (0, promises_1.stat)(resolved)).isFile()) {
                continue;
            }
            await (0, promises_1.access)(resolved, node_fs_1.constants.X_OK);
            return resolved;
        }
        catch {
            continue;
        }
    }
    throw new Error('Qasmith executable unavailable. Install Qasmith 0.9.0 or set qasmith.executablePath in User Settings.');
}
function integer(value, min, max, name) {
    if (typeof value !== 'number' || !Number.isSafeInteger(value) || value < min || value > max) {
        throw new Error(`${name} must be an integer between ${min} and ${max}.`);
    }
    return value;
}
function compilerArgs(operation, source, settings, output) {
    const args = [operation];
    if (operation === 'compile' && output)
        args.push('-o', output);
    if (operation === 'run' || operation === 'optimize')
        args.push('--shots', String(settings.shots), '--seed', String(settings.seed));
    if (operation === 'optimize')
        args.push('--max-evals', String(settings.evaluations), '--restarts', String(settings.restarts), '--eval-mode', settings.mode);
    return [...args, source];
}
function runCLI(executable, args, cwd, signal, output) {
    return new Promise((resolve, reject) => {
        if (signal.aborted) {
            reject(new Error('Qasmith operation canceled.'));
            return;
        }
        const child = (0, node_child_process_1.spawn)(executable, args, { cwd, shell: false, windowsHide: true, stdio: ['ignore', 'pipe', 'pipe'] });
        let killTimer;
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
            if (signal.aborted || termination)
                reject(new Error(signal.aborted ? 'Qasmith operation canceled.' : `Qasmith was terminated by ${termination}.`));
            else
                resolve(code ?? 1);
        });
    });
}
