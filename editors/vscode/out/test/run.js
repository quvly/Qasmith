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
const test_electron_1 = require("@vscode/test-electron");
const node_child_process_1 = require("node:child_process");
const promises_1 = require("node:fs/promises");
const node_os_1 = require("node:os");
const path = __importStar(require("node:path"));
async function main() {
    // The desktop Electron executable must start as VS Code when invoked from a VS Code terminal.
    delete process.env.ELECTRON_RUN_AS_NODE;
    const root = await (0, promises_1.mkdtemp)(path.join((0, node_os_1.tmpdir)(), 'qasmith extension test '));
    try {
        const workspace = path.join(root, 'smith workspace');
        await (0, promises_1.mkdir)(workspace);
        if (!process.env.QASMITH_TEST_CLI)
            throw new Error('Set QASMITH_TEST_CLI to the absolute path of the built qasmith CLI.');
        const common = {
            vscodeExecutablePath: process.env.VSCODE_EXECUTABLE ?? await (0, test_electron_1.downloadAndUnzipVSCode)(),
            extensionDevelopmentPath: path.resolve(__dirname, '../..'),
            extensionTestsPath: path.join(__dirname, 'host.js'),
        };
        await (0, test_electron_1.runTests)({
            ...common,
            launchArgs: [workspace, '--user-data-dir', path.join(root, 'profile'), '--extensions-dir', path.join(root, 'extensions'),
                '--skip-welcome', '--skip-release-notes', '--disable-updates', '--disable-workspace-trust'],
        });
        const restrictedProfile = path.join(root, 'r');
        const user = path.join(restrictedProfile, 'User');
        await (0, promises_1.mkdir)(user, { recursive: true });
        await (0, promises_1.writeFile)(path.join(user, 'settings.json'), JSON.stringify({
            'security.workspace.trust.startupPrompt': 'never',
            'security.workspace.trust.banner': 'never',
            'qasmith.executablePath': process.env.QASMITH_TEST_CLI,
        }));
        // The standard launcher always adds --disable-workspace-trust.
        await new Promise((resolve, reject) => {
            const host = (0, node_child_process_1.spawn)(common.vscodeExecutablePath, [
                workspace, '--user-data-dir', restrictedProfile, '--extensions-dir', path.join(root, 'extensions'),
                '--skip-welcome', '--skip-release-notes', '--disable-updates', '--no-sandbox',
                '--extensionDevelopmentPath=' + common.extensionDevelopmentPath,
                '--extensionTestsPath=' + common.extensionTestsPath,
            ], { env: { ...process.env, QASMITH_TEST_RESTRICTED: '1' }, stdio: 'inherit' });
            host.once('error', reject);
            host.once('close', (code, signal) => code === 0 ? resolve() : reject(new Error(`Restricted Mode host exited with ${signal ?? code}`)));
        });
    }
    finally {
        await (0, promises_1.rm)(root, { recursive: true, force: true });
    }
}
void main().catch(error => { console.error(error); process.exitCode = 1; });
