'use strict';

const { spawnSync } = require('node:child_process');
const fs = require('node:fs');
const path = require('node:path');
const { targets, architectures, binaryPath } = require('../npm/platform.cjs');
const { version } = require('../package.json');

const root = path.resolve(__dirname, '..');
const go = process.env.HOVA_GO || 'go';
if (!/^\d+\.\d+\.\d+(?:-[a-zA-Z0-9.-]+)?$/.test(version)) throw new Error('Invalid package version');
if (process.env.GITHUB_REF_TYPE === 'tag' && process.env.GITHUB_REF_NAME !== `v${version}`) {
  throw new Error('The release tag must match package.json version');
}

for (const [platform, goos] of Object.entries(targets)) {
  for (const [arch, goarch] of Object.entries(architectures)) {
    const output = binaryPath(root, platform, arch);
    fs.mkdirSync(path.dirname(output), { recursive: true });
    const result = spawnSync(go, ['build', '-trimpath', '-ldflags', `-s -w -X main.version=${version}`, '-o', output, './cmd/hova'], {
      cwd: root, stdio: 'inherit', shell: false,
      env: { ...process.env, GOOS: goos, GOARCH: goarch, CGO_ENABLED: '0' },
    });
    if (result.error || result.status !== 0) {
      process.stderr.write(`Cannot build ${platform}/${arch}: ${result.error?.message || `Go exited with ${result.status}`}. Maintainers need Go 1.24+; users can install the published package without Go.\n`);
      process.exit(1);
    }
    fs.chmodSync(output, 0o755);
    process.stdout.write(`Built ${platform}/${arch} · ${version}\n`);
  }
}
