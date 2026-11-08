'use strict';

const path = require('node:path');

const targets = Object.freeze({ linux: 'linux', darwin: 'darwin', win32: 'windows' });
const architectures = Object.freeze({ x64: 'amd64', arm64: 'arm64' });

function binaryPath(root, platform = process.platform, arch = process.arch) {
  if (!Object.hasOwn(targets, platform) || !Object.hasOwn(architectures, arch)) {
    throw new Error(`Unsupported system ${platform}/${arch}. Hova supports Windows, macOS and Linux on x64 and arm64. See https://github.com/alzubair-alojali/hova/releases`);
  }
  return path.join(root, 'native', `${platform}-${arch}`, platform === 'win32' ? 'hova.exe' : 'hova');
}

module.exports = { binaryPath, targets, architectures };
