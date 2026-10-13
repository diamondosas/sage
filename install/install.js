#!/usr/bin/env node
/**
 * Sage — Unified Cross-Platform Agent Skill & Binary Installer
 *
 * Usage:
 *   node installer/setup.js [--install] [--dry-run]
 */

'use strict';

const fs = require('fs');
const os = require('os');
const path = require('path');
const { execSync, spawnSync } = require('child_process');

const HOME = os.homedir();

// ── Agent Detection Matrix ──────────────────────────────────────────────────
// CHANGE HERE: Add more editors, agents, or custom skill paths as needed
const AGENTS = [
  {
    id: 'antigravity-cli',
    name: 'Antigravity CLI (agy)',
    // Detect if ~/.gemini/config or ~/.antigravity exists or agy is on PATH
    detect: () => hasCommand('agy') || dirExists(path.join(HOME, '.gemini')) || dirExists(path.join(HOME, '.antigravity')),
    skillPaths: [
      path.join(HOME, '.gemini', 'config', 'skills', 'sage'),
      path.join(HOME, '.antigravity', 'skills', 'sage')
    ]
  },
  {
    id: 'cursor',
    name: 'Cursor',
    detect: () => hasCommand('cursor') || dirExists(path.join(HOME, '.cursor')),
    skillPaths: [
      path.join(HOME, '.cursor', 'skills', 'sage')
    ]
  },
  {
    id: 'claude',
    name: 'Claude Code',
    detect: () => hasCommand('claude') || dirExists(path.join(HOME, '.claude')) || dirExists(path.join(HOME, '.config', 'claude')),
    skillPaths: [
      path.join(HOME, '.claude', 'skills', 'sage'),
      path.join(HOME, '.config', 'claude', 'skills', 'sage')
    ]
  },
  {
    id: 'copilot',
    name: 'GitHub Copilot',
    detect: () => hasCommand('copilot') || dirExists(path.join(HOME, '.copilot')),
    skillPaths: [
      path.join(HOME, '.copilot', 'skills', 'sage')
    ]
  },
  {
    id: 'codex',
    name: 'Codex CLI',
    detect: () => hasCommand('codex') || dirExists(path.join(HOME, '.codex')),
    skillPaths: [
      path.join(HOME, '.codex', 'skills', 'sage')
    ]
  },
  {
    id: 'windsurf',
    name: 'Windsurf',
    detect: () => hasCommand('windsurf') || dirExists(path.join(HOME, '.codeium', 'windsurf')),
    skillPaths: [
      path.join(HOME, '.codeium', 'windsurf', 'skills', 'sage')
    ]
  }
];

function hasCommand(bin) {
  try {
    const cmd = process.platform === 'win32' ? `where ${bin}` : `which ${bin}`;
    execSync(cmd, { stdio: 'ignore' });
    return true;
  } catch {
    return false;
  }
}

function dirExists(p) {
  try {
    return fs.existsSync(p);
  } catch {
    return false;
  }
}

// ── Main Setup Runner ──────────────────────────────────────────────────────
function runSetup(options = {}) {
  const { dryRun = false } = options;

  console.log('\n========================================');
  console.log('  🧙 Sage: AI Agent & Binary Setup');
  console.log('========================================\n');

  // 1. Locate source SKILL.md
  const projectRoot = path.resolve(__dirname, '..');
  const sourceSkill = path.join(projectRoot, 'skills', 'sage', 'SKILL.md');

  if (!fs.existsSync(sourceSkill)) {
    console.error(`❌ Error: Source SKILL.md not found at ${sourceSkill}`);
    process.exit(1);
  }
  const skillContent = fs.readFileSync(sourceSkill, 'utf8');

  // 2. Discover and install skills into detected agents
  console.log('🔍 Detecting installed AI coding agents:');
  let installedCount = 0;

  for (const agent of AGENTS) {
    if (agent.detect()) {
      for (const targetDir of agent.skillPaths) {
        const parentDir = path.dirname(targetDir);
        // Only install if base folder exists, or install to the primary path
        if (dirExists(parentDir) || targetDir === agent.skillPaths[0]) {
          const destFile = path.join(targetDir, 'SKILL.md');
          if (dryRun) {
            console.log(`  [dry-run] Would install skill -> ${destFile}`);
          } else {
            fs.mkdirSync(targetDir, { recursive: true });
            fs.writeFileSync(destFile, skillContent, 'utf8');
            console.log(`  ✓ Installed skill for ${agent.name} -> ${destFile}`);
          }
          installedCount++;
          break;
        }
      }
    }
  }

  if (installedCount === 0) {
    // If no specific editor directories detected, install to user's home .sage/skills as fallback
    const fallbackDir = path.join(HOME, '.sage', 'skills', 'sage');
    const destFile = path.join(fallbackDir, 'SKILL.md');
    if (!dryRun) {
      fs.mkdirSync(fallbackDir, { recursive: true });
      fs.writeFileSync(destFile, skillContent, 'utf8');
      console.log(`  ℹ  Installed default skill to fallback path -> ${destFile}`);
    }
  }

  // 3. Install or verify Sage CLI binary using install/install.sh (no Go required)
  console.log('\n🔨 Verifying Sage CLI binary:');
  const binDir = path.join(projectRoot, 'bin');
  const binName = process.platform === 'win32' ? 'sage.exe' : 'sage';
  const binPath = path.join(binDir, binName);

  if (!fs.existsSync(binPath) && !dryRun) {
    console.log('  ⚙  Running install script (install/install.sh) — no Go required...');
    try {
      if (process.platform === 'win32') {
        const psScript = path.join(projectRoot, 'install', 'install.ps1');
        execSync(`powershell -ExecutionPolicy Bypass -File "${psScript}"`, {
          cwd: projectRoot,
          env: { ...process.env, INSTALL_DIR: binDir },
          stdio: 'inherit'
        });
      } else {
        const shScript = path.join(projectRoot, 'install', 'install.sh');
        execSync(`bash "${shScript}"`, {
          cwd: projectRoot,
          env: { ...process.env, INSTALL_DIR: binDir },
          stdio: 'inherit'
        });
      }
      console.log(`  ✓ Native binary configured -> ${binPath}`);
    } catch (err) {
      console.warn(`  ⚠️  Install script notification: ${err.message}`);
    }
  } else if (fs.existsSync(binPath)) {
    console.log(`  ✓ Native binary ready at ${binPath}`);
  }

  console.log('\n✨ Sage setup complete!');
  console.log("👉 Run 'sage init' in your repository to configure context & GitHub Actions sync.\n");
}

if (require.main === module) {
  const args = process.argv.slice(2);
  const dryRun = args.includes('--dry-run');
  runSetup({ dryRun });
}

module.exports = { runSetup, AGENTS };
