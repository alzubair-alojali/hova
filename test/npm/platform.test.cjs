const { test } = require('node:test');
const assert = require('node:assert/strict');
const path = require('node:path');
const { binaryPath } = require('../../npm/platform.cjs');

test('selects the executable for each supported operating system and architecture', () => {
  for (const platform of ['win32', 'darwin', 'linux']) {
    for (const arch of ['x64', 'arm64']) {
      const expected = path.join('/package', 'native', `${platform}-${arch}`, platform === 'win32' ? 'hova.exe' : 'hova');
      assert.equal(binaryPath('/package', platform, arch), expected);
    }
  }
});

test('rejects unsupported architectures with installation guidance', () => {
  assert.throws(() => binaryPath('/package', 'linux', 'ia32'), /Unsupported.*linux\/ia32.*releases/s);
  assert.throws(() => binaryPath('/package', '../../outside', 'x64'), /Unsupported/);
});
