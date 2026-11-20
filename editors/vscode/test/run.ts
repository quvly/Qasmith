import { downloadAndUnzipVSCode, runTests } from '@vscode/test-electron';
import { spawn } from 'node:child_process';
import { mkdtemp, mkdir, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import * as path from 'node:path';

async function main(): Promise<void> {
    // The desktop Electron executable must start as VS Code when invoked from a VS Code terminal.
    delete process.env.ELECTRON_RUN_AS_NODE;
    const root = await mkdtemp(path.join(tmpdir(), 'qasmith extension test '));
    try {
        const workspace = path.join(root, 'smith workspace');
        await mkdir(workspace);
        if (!process.env.QASMITH_TEST_CLI) throw new Error('Set QASMITH_TEST_CLI to the absolute path of the built qasmith CLI.');
        const common = {
            vscodeExecutablePath: process.env.VSCODE_EXECUTABLE ?? await downloadAndUnzipVSCode(),
            extensionDevelopmentPath: path.resolve(__dirname, '../..'),
            extensionTestsPath: path.join(__dirname, 'host.js'),
        };
        await runTests({
            ...common,
            launchArgs: [workspace, '--user-data-dir', path.join(root, 'profile'), '--extensions-dir', path.join(root, 'extensions'),
                '--skip-welcome', '--skip-release-notes', '--disable-updates', '--disable-workspace-trust'],
        });
        const restrictedProfile = path.join(root, 'r');
        const user = path.join(restrictedProfile, 'User');
        await mkdir(user, { recursive: true });
        await writeFile(path.join(user, 'settings.json'), JSON.stringify({
            'security.workspace.trust.startupPrompt': 'never',
            'security.workspace.trust.banner': 'never',
            'qasmith.executablePath': process.env.QASMITH_TEST_CLI,
        }));
        // The standard launcher always adds --disable-workspace-trust.
        await new Promise<void>((resolve, reject) => {
            const host = spawn(common.vscodeExecutablePath, [
                workspace, '--user-data-dir', restrictedProfile, '--extensions-dir', path.join(root, 'extensions'),
                '--skip-welcome', '--skip-release-notes', '--disable-updates', '--no-sandbox',
                '--extensionDevelopmentPath=' + common.extensionDevelopmentPath,
                '--extensionTestsPath=' + common.extensionTestsPath,
            ], { env: { ...process.env, QASMITH_TEST_RESTRICTED: '1' }, stdio: 'inherit' });
            host.once('error', reject);
            host.once('close', (code, signal) => code === 0 ? resolve() : reject(new Error(`Restricted Mode host exited with ${signal ?? code}`)));
        });
    } finally {
        await rm(root, { recursive: true, force: true });
    }
}
void main().catch(error => { console.error(error); process.exitCode = 1; });
