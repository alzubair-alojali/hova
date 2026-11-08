#!/usr/bin/env node
'use strict';

const { spawn } = require('node:child_process');
const { constants } = require('node:os');
const path = require('node:path');
const { binaryPath } = require('./platform.cjs');

function reportError(error, args) {
  const boundary = args.indexOf('--');
  const flags = boundary < 0 ? args : args.slice(0, boundary);
  const message = `${error.message}. Reinstall with npm install -g hova-cli, or download from https://github.com/alzubair-alojali/hova/releases`;
  const json = flags.includes('--json') || flags.includes('--json=true');
  process.stderr.write(json ? `${JSON.stringify({ error: message })}\n` : `hova: ${message}\n`);
  process.exitCode = 1;
}

function launch(executable, args) {
  const child = spawn(executable, args, { stdio: 'inherit', shell: false });
  let failed = false;
  const forward = signal => child.kill(signal);
  const onInterrupt = () => forward('SIGINT');
  const onTerminate = () => forward('SIGTERM');
  process.on('SIGINT', onInterrupt);
  process.on('SIGTERM', onTerminate);
  const cleanup = () => {
    process.removeListener('SIGINT', onInterrupt);
    process.removeListener('SIGTERM', onTerminate);
  };
  child.once('error', error => { failed = true; cleanup(); reportError(error, args); });
  child.once('close', (code, signal) => {
    cleanup();
    if (failed) return;
    process.exitCode = signal ? 128 + (constants.signals[signal] || 1) : (code < 0 ? 1 : code ?? 1);
  });
  return child;
}

if (require.main === module) {
  const args = process.argv.slice(2);
  try { launch(binaryPath(path.join(__dirname, '..')), args); }
  catch (error) { reportError(error, args); }
}

module.exports = { launch };
