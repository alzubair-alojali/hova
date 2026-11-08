const { test } = require('node:test');
const assert = require('node:assert/strict');
const { spawnSync, spawn } = require('node:child_process');
const path = require('node:path');
const os = require('node:os');

const launcher = path.resolve(__dirname, '../../npm/cli.cjs');
const fixture = `const { launch } = require(${JSON.stringify(launcher)}); launch(process.execPath, ['-e', process.env.HOVA_FIXTURE_SOURCE, ...process.argv.slice(1)]);`;

test('inherits streams and environment, preserves literal arguments and exit status', () => {
  const source = `process.stdin.on('data', data => { process.stdout.write(JSON.stringify({args:process.argv.slice(1),input:data.toString(),env:process.env.HOVA_FIXTURE_VALUE}));process.stderr.write('remote error');process.exitCode=7; });`;
  const result = spawnSync(process.execPath, ['-e', fixture, 'a b', '$(not-a-shell)', '--json'], {
    input: 'hello', encoding: 'utf8', env: { ...process.env, HOVA_FIXTURE_SOURCE: source, HOVA_FIXTURE_VALUE: 'preserved' },
  });
  assert.equal(result.status, 7);
  assert.equal(result.stderr, 'remote error');
  assert.deepEqual(JSON.parse(result.stdout), { args: ['a b', '$(not-a-shell)', '--json'], input: 'hello', env: 'preserved' });
});

test('reports launch failures as JSON on stderr without contaminating stdout', () => {
  const source = `require(${JSON.stringify(launcher)}).launch(${JSON.stringify(path.join(os.tmpdir(), 'hova-nonexistent-executable'))}, ['--json']);`;
  const result = spawnSync(process.execPath, ['-e', source], { encoding: 'utf8' });
  assert.equal(result.status, 1);
  assert.equal(result.stdout, '');
  assert.match(JSON.parse(result.stderr).error, /reinstall|install/i);
});

test('only treats --json before the remote command boundary as a wrapper flag', () => {
  const source = `require(${JSON.stringify(launcher)}).launch('hova-nonexistent-executable', ['exec','prod','--','--json']);`;
  const result = spawnSync(process.execPath, ['-e', source], { encoding: 'utf8' });
  assert.equal(result.status, 1);
  assert.match(result.stderr, /^hova:/);
});

test('forwards SIGTERM and returns the corresponding exit code', { skip: process.platform === 'win32', timeout: 10000 }, async t => {
  const child = spawn(process.execPath, ['-e', fixture], { env: { ...process.env, HOVA_FIXTURE_SOURCE: `process.stdout.write('ready');setInterval(()=>{},1000);` }, stdio: ['ignore', 'pipe', 'pipe'] });
  t.after(() => { if (child.exitCode === null && child.signalCode === null) child.kill('SIGKILL'); });
  const result = new Promise(resolve => child.once('exit', (code, signal) => resolve({ code, signal })));
  await new Promise((resolve, reject) => { child.stdout.once('data', resolve); child.once('error', reject); });
  child.kill('SIGTERM');
  assert.deepEqual(await result, { code: 143, signal: null });
});
