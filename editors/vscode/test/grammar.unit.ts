import { strict as assert } from 'node:assert';
import { readFileSync } from 'node:fs';
import * as path from 'node:path';
import { test } from 'node:test';
import * as onig from 'vscode-oniguruma';
import { INITIAL, parseRawGrammar, Registry } from 'vscode-textmate';

test('TextMate scopes respect lexical context and survive incomplete input', async () => {
    const wasm = Uint8Array.from(readFileSync(require.resolve('vscode-oniguruma/release/onig.wasm')));
    await onig.loadWASM(wasm.buffer);
    const registry = new Registry({
        onigLib: Promise.resolve(onig),
        loadGrammar: async scope => {
            if (scope !== 'source.qasmith') return null;
            const file = path.resolve(__dirname, '../../syntaxes/qasmith.tmLanguage.json');
            return parseRawGrammar(readFileSync(file, 'utf8'), file);
        },
    });
    const grammar = await registry.loadGrammar('source.qasmith');
    assert.ok(grammar);
    let stack = INITIAL;
    function line(source: string, checks: Array<[string, string]>) {
        const result = grammar!.tokenizeLine(source, stack);
        stack = result.ruleStack;
        for (const [fragment, expected] of checks) {
            const offset = source.indexOf(fragment);
            const token = result.tokens.find(token => token.startIndex <= offset && offset < token.endIndex);
            assert.ok(token && token.scopes.includes(expected + '.qasmith'), `${fragment} should have ${expected}: ${token?.scopes}`);
            if (expected.startsWith('string.') || expected.startsWith('comment.')) {
                assert.ok(!token.scopes.some(scope => scope.startsWith('keyword.')));
            }
        }
    }
    line('pub fn build(q: QReg<4>) -> Void {', [['pub', 'storage.modifier'], ['build', 'entity.name.function'], ['QReg', 'support.type'], ['4', 'constant.numeric']]);
    line('for i in 0..4 { rx(3.0e-4, q[i]); }', [['for', 'keyword.control'], ['..', 'keyword.operator'], ['4', 'constant.numeric'], ['rx', 'support.function.quantum'], ['3.0e-4', 'constant.numeric'], ['[', 'punctuation.section.brackets']]);
    line('import { a } from "./fn.smith";', [['import', 'keyword.control'], ['fn.smith', 'string.quoted.double.import']]);
    line('/* for', [['for', 'comment.block']]);
    line('return */ let x = [[1]]; // else', [['return', 'comment.block'], ['let', 'keyword.control'], ['else', 'comment.line.double-slash']]);
    line('"for', [['for', 'string.quoted.double']]);
    line('if true {', [['if', 'keyword.control'], ['true', 'constant.language.boolean']]);
});
