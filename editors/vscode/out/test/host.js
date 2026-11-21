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
exports.run = run;
const node_assert_1 = require("node:assert");
const promises_1 = require("node:fs/promises");
const path = __importStar(require("node:path"));
const vscode = __importStar(require("vscode"));
async function eventually(check, accept) {
    const until = Date.now() + 15000;
    while (Date.now() < until) {
        const value = await check();
        if (accept(value))
            return value;
        await new Promise(resolve => setTimeout(resolve, 100));
    }
    throw new Error('Timed out waiting for the Qasmith provider');
}
async function run() {
    const root = vscode.workspace.workspaceFolders?.[0].uri.fsPath;
    node_assert_1.strict.ok(root);
    const config = vscode.workspace.getConfiguration('qasmith');
    const global = vscode.ConfigurationTarget.Global;
    if (process.env.QASMITH_TEST_RESTRICTED) {
        node_assert_1.strict.equal(vscode.workspace.isTrusted, false);
        const doc = await vscode.workspace.openTextDocument(path.join(root, 'entry file.smith'));
        node_assert_1.strict.equal(doc.languageId, 'qasmith');
        await vscode.window.showTextDocument(doc);
        await vscode.commands.executeCommand('qasmith.restart');
        node_assert_1.strict.equal(await vscode.commands.executeCommand('qasmith.check'), false);
        const definitions = await vscode.commands.executeCommand('vscode.executeDefinitionProvider', doc.uri, doc.positionAt(doc.getText().lastIndexOf('count()')));
        node_assert_1.strict.equal(definitions?.length ?? 0, 0);
        console.log('Qasmith Restricted Mode checks passed');
        return;
    }
    await config.update('executablePath', path.join(root, 'missing compiler'), global);
    const dependency = path.join(root, 'library file.smith');
    const file = path.join(root, 'entry file.smith');
    await (0, promises_1.writeFile)(dependency, 'pub fn helper() -> Int { return 1; }\n');
    const source = 'import { helper as count } from "./library file.smith";\nfn main() -> MeasuredResult { let n = count(); let q = qubit(); h(q); return measure(q); }\n';
    await (0, promises_1.writeFile)(file, source);
    const doc = await vscode.workspace.openTextDocument(file);
    node_assert_1.strict.equal(doc.languageId, 'qasmith');
    const editor = await vscode.window.showTextDocument(doc);
    const extension = vscode.extensions.getExtension('Quvly.qasmith-language');
    node_assert_1.strict.ok(extension);
    await extension.activate();
    node_assert_1.strict.equal(await vscode.commands.executeCommand('qasmith.check'), false);
    await config.update('executablePath', process.env.QASMITH_TEST_CLI, global);
    await vscode.commands.executeCommand('qasmith.restart');
    const at = (name) => doc.positionAt(doc.getText().lastIndexOf(name));
    const definitions = await eventually(() => vscode.commands.executeCommand('vscode.executeDefinitionProvider', doc.uri, at('count()')), locations => (locations?.length ?? 0) > 0);
    node_assert_1.strict.equal(await (0, promises_1.realpath)(definitions[0].uri.fsPath), await (0, promises_1.realpath)(dependency));
    const completions = await vscode.commands.executeCommand('vscode.executeCompletionItemProvider', doc.uri, at('return'));
    node_assert_1.strict.ok(completions.items.some(item => item.label === 'measure'));
    const position = at('measure(q)');
    const token = new vscode.Range(position, position.translate(0, 'measure'.length));
    await editor.edit(edit => edit.replace(token, 'badGate'));
    await eventually(() => vscode.languages.getDiagnostics(doc.uri), found => found.some(diagnostic => diagnostic.message.includes('badGate')));
    const bad = at('badGate');
    await editor.edit(edit => edit.replace(new vscode.Range(bad, bad.translate(0, 7)), 'measure'));
    await eventually(() => vscode.languages.getDiagnostics(doc.uri), found => found.length === 0);
    await doc.save();
    node_assert_1.strict.equal(await vscode.commands.executeCommand('qasmith.check'), true);
    node_assert_1.strict.equal(await vscode.commands.executeCommand('qasmith.compile'), true);
    const generated = await (0, promises_1.readFile)(path.join(root, 'entry file.qasm'), 'utf8');
    node_assert_1.strict.match(generated, /OPENQASM 3/);
    node_assert_1.strict.equal(await vscode.commands.executeCommand('qasmith.run'), true);
    node_assert_1.strict.equal(await vscode.commands.executeCommand('qasmith.preview'), true);
    node_assert_1.strict.ok(vscode.window.visibleTextEditors.some(view => view.document.languageId === 'qasmith-openqasm' && view.document.getText() === generated));
    const optimize = path.join(root, 'optimize entry.smith');
    await (0, promises_1.copyFile)(path.resolve(__dirname, '../../../../examples/optimization/qubo-language/matrix.smith'), optimize);
    await vscode.window.showTextDocument(await vscode.workspace.openTextDocument(optimize));
    await config.update('optimize.maxEvaluations', 8, global);
    await config.update('optimize.restarts', 0, global);
    node_assert_1.strict.equal(await vscode.commands.executeCommand('qasmith.optimize'), true);
    await vscode.commands.executeCommand('qasmith.output');
    console.log('Qasmith extension-host checks passed');
}
