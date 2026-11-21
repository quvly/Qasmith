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
const promises_1 = require("node:fs/promises");
const node_os_1 = require("node:os");
const path = __importStar(require("node:path"));
const node_test_1 = require("node:test");
const compiler_1 = require("../src/compiler");
(0, node_test_1.test)('settings and command arguments preserve literal paths and seeds', () => {
    node_assert_1.strict.deepEqual((0, compiler_1.compilerArgs)('optimize', '/my project/entry.smith', {
        shots: 8, seed: -2, evaluations: 20, restarts: 0, mode: 'sampled',
    }), ['optimize', '--shots', '8', '--seed', '-2', '--max-evals', '20', '--restarts', '0', '--eval-mode', 'sampled', '/my project/entry.smith']);
    for (const value of [NaN, Infinity, -1, 2.5, '2'])
        node_assert_1.strict.throws(() => (0, compiler_1.integer)(value, 0, 4, 'setting'));
});
(0, node_test_1.test)('process execution avoids shell interpretation and supports cancellation', async () => {
    const work = await (0, promises_1.mkdtemp)(path.join((0, node_os_1.tmpdir)(), 'qasmith unit '));
    try {
        const executable = await (0, compiler_1.resolveCompiler)(process.execPath, {}, []);
        let output = '';
        const file = 'a $(invalid) source.smith';
        const controller = new AbortController();
        const code = await (0, compiler_1.runCLI)(executable, ['-e', 'process.stdout.write(process.argv[1])', file], work, controller.signal, text => { output += text; });
        node_assert_1.strict.equal(code, 0);
        node_assert_1.strict.equal(output, file);
        const running = (0, compiler_1.runCLI)(executable, ['-e', 'setInterval(() => {}, 1000)'], work, controller.signal, () => { });
        controller.abort();
        await node_assert_1.strict.rejects(running, /canceled/);
        await node_assert_1.strict.rejects((0, compiler_1.resolveCompiler)('workspace/qasmith', {}, []), /absolute path/);
        await node_assert_1.strict.rejects((0, compiler_1.resolveCompiler)('', { PATH: '.' }, []), /unavailable/);
    }
    finally {
        await (0, promises_1.rm)(work, { recursive: true, force: true });
    }
});
