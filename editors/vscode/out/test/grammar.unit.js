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
const node_assert_1 = require("node:assert");
const node_fs_1 = require("node:fs");
const path = __importStar(require("node:path"));
const node_test_1 = require("node:test");
const onig = __importStar(require("vscode-oniguruma"));
const vscode_textmate_1 = require("vscode-textmate");
(0, node_test_1.test)('TextMate scopes respect lexical context and survive incomplete input', async () => {
    const wasm = Uint8Array.from((0, node_fs_1.readFileSync)(require.resolve('vscode-oniguruma/release/onig.wasm')));
    await onig.loadWASM(wasm.buffer);
    const registry = new vscode_textmate_1.Registry({
        onigLib: Promise.resolve(onig),
        loadGrammar: async (scope) => {
            if (scope !== 'source.qasmith')
                return null;
            const file = path.resolve(__dirname, '../../syntaxes/qasmith.tmLanguage.json');
            return (0, vscode_textmate_1.parseRawGrammar)((0, node_fs_1.readFileSync)(file, 'utf8'), file);
        },
    });
    const grammar = await registry.loadGrammar('source.qasmith');
    node_assert_1.strict.ok(grammar);
    let stack = vscode_textmate_1.INITIAL;
    function line(source, checks) {
        const result = grammar.tokenizeLine(source, stack);
        stack = result.ruleStack;
        for (const [fragment, expected] of checks) {
            const offset = source.indexOf(fragment);
            const token = result.tokens.find(token => token.startIndex <= offset && offset < token.endIndex);
            node_assert_1.strict.ok(token && token.scopes.includes(expected + '.qasmith'), `${fragment} should have ${expected}: ${token?.scopes}`);
            if (expected.startsWith('string.') || expected.startsWith('comment.')) {
                node_assert_1.strict.ok(!token.scopes.some(scope => scope.startsWith('keyword.')));
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
