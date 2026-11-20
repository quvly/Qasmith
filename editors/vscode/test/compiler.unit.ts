import { strict as assert } from 'node:assert';
import { mkdtemp, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import * as path from 'node:path';
import { test } from 'node:test';
import { compilerArgs, integer, resolveCompiler, runCLI } from '../src/compiler';

test('settings and command arguments preserve literal paths and seeds', () => {
    assert.deepEqual(compilerArgs('optimize', '/my project/entry.smith', {
        shots: 8, seed: -2, evaluations: 20, restarts: 0, mode: 'sampled',
    }), ['optimize', '--shots', '8', '--seed', '-2', '--max-evals', '20', '--restarts', '0', '--eval-mode', 'sampled', '/my project/entry.smith']);
    for (const value of [NaN, Infinity, -1, 2.5, '2']) assert.throws(() => integer(value, 0, 4, 'setting'));
});

test('process execution avoids shell interpretation and supports cancellation', async () => {
    const work = await mkdtemp(path.join(tmpdir(), 'qasmith unit '));
    try {
        const executable = await resolveCompiler(process.execPath, {}, []);
        let output = '';
        const file = 'a $(invalid) source.smith';
        const controller = new AbortController();
        const code = await runCLI(executable, ['-e', 'process.stdout.write(process.argv[1])', file], work, controller.signal, text => { output += text; });
        assert.equal(code, 0);
        assert.equal(output, file);
        const running = runCLI(executable, ['-e', 'setInterval(() => {}, 1000)'], work, controller.signal, () => {});
        controller.abort();
        await assert.rejects(running, /canceled/);
        await assert.rejects(resolveCompiler('workspace/qasmith', {}, []), /absolute path/);
        await assert.rejects(resolveCompiler('', { PATH: '.' }, []), /unavailable/);
    } finally {
        await rm(work, { recursive: true, force: true });
    }
});
