const { test } = require('node:test');
const assert = require('node:assert/strict');
const { spawnSync } = require('node:child_process');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');

const root = path.resolve(__dirname, '../..');
const manifest = require('../../package.json');
const npmCLI = process.env.npm_execpath;

function npm(args, options = {}) {
  assert.ok(npmCLI, 'Run this test with npm run test:package');
  const result = spawnSync(process.execPath, [npmCLI, ...args], { cwd: root, encoding: 'utf8', timeout: 120000, ...options });
  assert.equal(result.status, 0, result.stderr || result.error?.message);
  return result.stdout;
}

test('packed package includes six binaries and excludes source, credentials and private data', () => {
  const [packed] = JSON.parse(npm(['pack', '--dry-run', '--json', '--ignore-scripts']));
  const names = packed.files.map(file => file.path);
  for (const platform of ['linux', 'darwin', 'win32']) {
    for (const arch of ['x64', 'arm64']) {
      const name = `native/${platform}-${arch}/hova${platform === 'win32' ? '.exe' : ''}`;
      assert.ok(names.includes(name));
      if (process.platform !== 'win32' && platform !== 'win32') {
        assert.equal(packed.files.find(file => file.path === name).mode & 0o111, 0o111, `${name} must be executable in the published tarball`);
      }
    }
  }
  assert.ok(names.includes('npm/cli.cjs'));
  for (const name of names) assert.match(name, /^(npm\/(cli|platform)\.cjs|native\/[^/]+\/hova(?:\.exe)?|assets\/[^/]+\.svg|package\.json|README\.md|LICENSE)$/);
});

test('tarball installs and runs with lifecycle scripts disabled and no Go compilation', t => {
  const temporary = fs.mkdtempSync(path.join(os.tmpdir(), 'hova-npm-'));
  t.after(() => fs.rmSync(temporary, { recursive: true, force: true }));
  const [packed] = JSON.parse(npm(['pack', '--json', '--ignore-scripts', '--pack-destination', temporary]));
  const archive = path.join(temporary, packed.filename);
  const prefix = path.join(temporary, 'prefix with spaces');
  npm(['install', '--global', '--prefix', prefix, '--ignore-scripts', '--no-audit', '--no-fund', archive]);
  const moduleRoot = process.platform === 'win32' ? path.join(prefix, 'node_modules') : path.join(prefix, 'lib', 'node_modules');
  const launcher = path.join(moduleRoot, 'hova-cli', 'npm', 'cli.cjs');
  const version = spawnSync(process.execPath, [launcher, '--version'], { encoding: 'utf8' });
  assert.equal(version.status, 0, version.stderr);
  assert.equal(version.stdout.trim(), `hova version ${manifest.version}`);
  const inventory = path.join(temporary, 'inventory.json');
  const listing = spawnSync(process.execPath, [launcher, '--config', inventory, 'list', '--json'], { encoding: 'utf8' });
  assert.equal(listing.status, 0, listing.stderr);
  assert.deepEqual(JSON.parse(listing.stdout), []);
  const shim = process.platform === 'win32' ? path.join(prefix, 'hova.cmd') : path.join(prefix, 'bin', 'hova');
  assert.ok(fs.existsSync(shim), 'npm did not create the hova command');
});
