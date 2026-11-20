import { strict as assert } from 'node:assert';
import { copyFile, readFile, realpath, writeFile } from 'node:fs/promises';
import * as path from 'node:path';
import * as vscode from 'vscode';

async function eventually<T>(check: () => T | Thenable<T>, accept: (answer: T) => boolean): Promise<T> {
    const until = Date.now() + 15000;
    while (Date.now() < until) {
        const value = await check();
        if (accept(value)) return value;
        await new Promise(resolve => setTimeout(resolve, 100));
    }
    throw new Error('Timed out waiting for the Qasmith provider');
}

export async function run(): Promise<void> {
    const root = vscode.workspace.workspaceFolders?.[0].uri.fsPath;
    assert.ok(root);
    const config = vscode.workspace.getConfiguration('qasmith');
    const global = vscode.ConfigurationTarget.Global;
    if (process.env.QASMITH_TEST_RESTRICTED) {
        assert.equal(vscode.workspace.isTrusted, false);
        const doc = await vscode.workspace.openTextDocument(path.join(root, 'entry file.smith'));
        assert.equal(doc.languageId, 'qasmith');
        await vscode.window.showTextDocument(doc);
        await vscode.commands.executeCommand('qasmith.restart');
        assert.equal(await vscode.commands.executeCommand('qasmith.check'), false);
        const definitions = await vscode.commands.executeCommand<vscode.Location[]>(
            'vscode.executeDefinitionProvider', doc.uri, doc.positionAt(doc.getText().lastIndexOf('count()'))
        );
        assert.equal(definitions?.length ?? 0, 0);
        console.log('Qasmith Restricted Mode checks passed');
        return;
    }
    await config.update('executablePath', path.join(root, 'missing compiler'), global);
    const dependency = path.join(root, 'library file.smith');
    const file = path.join(root, 'entry file.smith');
    await writeFile(dependency, 'pub fn helper() -> Int { return 1; }\n');
    const source = 'import { helper as count } from "./library file.smith";\nfn main() -> MeasuredResult { let n = count(); let q = qubit(); h(q); return measure(q); }\n';
    await writeFile(file, source);
    const doc = await vscode.workspace.openTextDocument(file);
    assert.equal(doc.languageId, 'qasmith');
    const editor = await vscode.window.showTextDocument(doc);
    const extension = vscode.extensions.getExtension('Quvly.qasmith-language');
    assert.ok(extension);
    await extension.activate();
    assert.equal(await vscode.commands.executeCommand('qasmith.check'), false);
    await config.update('executablePath', process.env.QASMITH_TEST_CLI, global);
    await vscode.commands.executeCommand('qasmith.restart');
    const at = (name: string) => doc.positionAt(doc.getText().lastIndexOf(name));
    const definitions = await eventually(
        () => vscode.commands.executeCommand<vscode.Location[]>('vscode.executeDefinitionProvider', doc.uri, at('count()')),
        locations => (locations?.length ?? 0) > 0,
    );
    assert.equal(await realpath(definitions[0].uri.fsPath), await realpath(dependency));
    const completions = await vscode.commands.executeCommand<vscode.CompletionList>('vscode.executeCompletionItemProvider', doc.uri, at('return'));
    assert.ok(completions.items.some(item => item.label === 'measure'));
    const position = at('measure(q)');
    const token = new vscode.Range(position, position.translate(0, 'measure'.length));
    await editor.edit(edit => edit.replace(token, 'badGate'));
    await eventually(() => vscode.languages.getDiagnostics(doc.uri), found => found.some(diagnostic => diagnostic.message.includes('badGate')));
    const bad = at('badGate');
    await editor.edit(edit => edit.replace(new vscode.Range(bad, bad.translate(0, 7)), 'measure'));
    await eventually(() => vscode.languages.getDiagnostics(doc.uri), found => found.length === 0);
    await doc.save();
    assert.equal(await vscode.commands.executeCommand('qasmith.check'), true);
    assert.equal(await vscode.commands.executeCommand('qasmith.compile'), true);
    const generated = await readFile(path.join(root, 'entry file.qasm'), 'utf8');
    assert.match(generated, /OPENQASM 3/);
    assert.equal(await vscode.commands.executeCommand('qasmith.run'), true);
    assert.equal(await vscode.commands.executeCommand('qasmith.preview'), true);
    assert.ok(vscode.window.visibleTextEditors.some(view => view.document.languageId === 'qasmith-openqasm' && view.document.getText() === generated));
    const optimize = path.join(root, 'optimize entry.smith');
    await copyFile(path.resolve(__dirname, '../../../../examples/optimization/qubo-language/matrix.smith'), optimize);
    await vscode.window.showTextDocument(await vscode.workspace.openTextDocument(optimize));
    await config.update('optimize.maxEvaluations', 8, global);
    await config.update('optimize.restarts', 0, global);
    assert.equal(await vscode.commands.executeCommand('qasmith.optimize'), true);
    await vscode.commands.executeCommand('qasmith.output');
    console.log('Qasmith extension-host checks passed');
}
