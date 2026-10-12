#!/usr/bin/env node
/**
 * Sage CLI Node.js Executable Shim
 *
 * Provides the global 'sage' executable when installed via npm:
 *   npm install -g sage-brain
 *
 * Forwards commands to the native Sage binary, and intercepts
 * 'sage setup' to install skills into AI agents (Caveman pattern).
 */

'use strict';

const fs = require('fs');
const path = require('path');
const { spawn, execSync } = require('child_process');

const args = process.argv.slice(2);

// Intercept 'sage setup' or 'sage setup --install'
if (args[0] === 'setup') {
  const { runSetup } = require('../installer/setup');
  const dryRun = args.includes('--dry-run');
  runSetup({ dryRun });
  process.exit(0);
}

// Find or build native Sage binary
const projectRoot = path.resolve(__dirname, '..');
const binName = process.platform === 'win32' ? 'sage.exe' : 'sage';
const binPath = path.join(projectRoot, 'bin', binName);

if (!fs.existsSync(binPath)) {
  // If not present, run install/install.sh (or install.ps1) to obtain binary without requiring Go
  try {
    fs.mkdirSync(path.join(projectRoot, 'bin'), { recursive: true });
    if (process.platform === 'win32') {
      const psScript = path.join(projectRoot, 'install', 'install.ps1');
      execSync(`powershell -ExecutionPolicy Bypass -File "${psScript}"`, {
        cwd: projectRoot,
        env: { ...process.env, INSTALL_DIR: path.join(projectRoot, 'bin') },
        stdio: 'inherit'
      });
    } else {
      const shScript = path.join(projectRoot, 'install', 'install.sh');
      execSync(`bash "${shScript}"`, {
        cwd: projectRoot,
        env: { ...process.env, INSTALL_DIR: path.join(projectRoot, 'bin') },
        stdio: 'inherit'
      });
    }
  } catch (err) {
    // If install script fails, check if 'sage' is already available in PATH
    try {
      execSync(process.platform === 'win32' ? 'where sage' : 'which sage', { stdio: 'ignore' });
    } catch {
      console.error('❌ Error: Native Sage binary not found and install script failed.');
      console.error('👉 Run "sage setup --install" to configure Sage.');
      process.exit(1);
    }
  }
}

// Spawn native Sage binary with passed arguments
const proc = spawn(binPath, args, { stdio: 'inherit' });
proc.on('exit', (code, signal) => {
  if (signal) {
    process.kill(process.pid, signal);
  } else {
    process.exit(code || 0);
  }
});
proc.on('error', (err) => {
  console.error(`Failed to launch Sage binary: ${err.message}`);
  process.exit(1);
});
