#!/usr/bin/env node

/**
 * Sage & Agent Skills Installer (Cross-Platform Node.js)
 * Installs both the Sage native binary and the Agent Skill into:
 *   - Universal Open Standard: ~/.agents/skills/sage (and .agents/skills/sage)
 *   - Claude Code:              ~/.claude/skills/sage (and .claude/skills/sage)
 *   - Cursor AI:                ~/.cursor/skills/sage (and .cursor/skills/sage)
 *
 * Usage:
 *   node assets/install.js
 *   or via npx: npx sage-codebase (or npx @diamondosas/sage)
 */

const fs = require('fs');
const path = require('path');
const os = require('os');
const https = require('https');

const REPO = process.env.SAGE_REPO || 'diamondosas/sage';
const REQUESTED_VERSION = process.env.SAGE_VERSION || 'latest';

// Platform & Arch Mapping
function getPlatform() {
  switch (process.platform) {
    case 'linux': return 'linux';
    case 'darwin': return 'darwin';
    case 'win32': return 'windows';
    default:
      throw new Error(`Unsupported operating system: ${process.platform}`);
  }
}

function getArch() {
  switch (process.arch) {
    case 'x64': return 'amd64';
    case 'arm64': return 'arm64';
    default:
      throw new Error(`Unsupported CPU architecture: ${process.arch}`);
  }
}

// Download helper with redirect handling
function downloadFile(url, dest) {
  return new Promise((resolve, reject) => {
    https.get(url, (res) => {
      if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
        return downloadFile(res.headers.location, dest).then(resolve).catch(reject);
      }
      if (res.statusCode !== 200) {
        return reject(new Error(`Failed to download from ${url} (HTTP ${res.statusCode})`));
      }
      const file = fs.createWriteStream(dest);
      res.pipe(file);
      file.on('finish', () => {
        file.close(() => resolve());
      });
      file.on('error', (err) => {
        fs.unlink(dest, () => reject(err));
      });
    }).on('error', (err) => {
      fs.unlink(dest, () => reject(err));
    });
  });
}

// Determine Binary Target Path
function getBinaryInstallDir() {
  if (process.env.BINDIR) return path.resolve(process.env.BINDIR);

  const home = os.homedir();
  if (process.platform === 'win32') {
    const localAppData = process.env.LOCALAPPDATA || path.join(home, 'AppData', 'Local');
    return path.join(localAppData, 'Microsoft', 'WindowsApps');
  }

  // Unix default: ~/.local/bin
  return path.join(home, '.local', 'bin');
}

// Discover all target skill directories (Global + Current Project)
function getTargetSkillDirs() {
  const home = os.homedir();
  const cwd = process.cwd();
  const targets = new Set();

  // 1. Universal Agent Skills standard (agentskills.io)
  targets.add(path.join(home, '.agents', 'skills', 'sage'));

  // 2. Claude Code global directory
  targets.add(path.join(home, '.claude', 'skills', 'sage'));

  // 3. Cursor AI global directory
  targets.add(path.join(home, '.cursor', 'skills', 'sage'));

  // 4. Project-level directories if inside a project / repo
  const hasGit = fs.existsSync(path.join(cwd, '.git'));
  const hasPkg = fs.existsSync(path.join(cwd, 'package.json'));

  if (hasGit || hasPkg) {
    // Project universal standard
    targets.add(path.join(cwd, '.agents', 'skills', 'sage'));
    
    // If project has .claude or .cursor folders, install there too
    if (fs.existsSync(path.join(cwd, '.claude'))) {
      targets.add(path.join(cwd, '.claude', 'skills', 'sage'));
    }
    if (fs.existsSync(path.join(cwd, '.cursor'))) {
      targets.add(path.join(cwd, '.cursor', 'skills', 'sage'));
    }
  }

  return Array.from(targets);
}

async function installSkill(targetSkillDir) {
  fs.mkdirSync(path.join(targetSkillDir, 'references'), { recursive: true });
  fs.mkdirSync(path.join(targetSkillDir, 'scripts'), { recursive: true });

  const localSkillDir = path.resolve(__dirname, '..', 'skills', 'sage');
  if (fs.existsSync(localSkillDir)) {
    // Copy from local repository
    fs.cpSync(localSkillDir, targetSkillDir, { recursive: true });
  } else {
    // Download directly from GitHub repository
    const baseUrl = `https://raw.githubusercontent.com/${REPO}/main/skills/sage`;
    await downloadFile(`${baseUrl}/SKILL.md`, path.join(targetSkillDir, 'SKILL.md'));
    await downloadFile(`${baseUrl}/references/language-syntax.md`, path.join(targetSkillDir, 'references', 'language-syntax.md'));
    await downloadFile(`${baseUrl}/scripts/ensure-sage.sh`, path.join(targetSkillDir, 'scripts', 'ensure-sage.sh'));
  }

  // Ensure script is executable
  const scriptPath = path.join(targetSkillDir, 'scripts', 'ensure-sage.sh');
  if (fs.existsSync(scriptPath) && process.platform !== 'win32') {
    fs.chmodSync(scriptPath, 0o755);
  }
}

async function installBinary() {
  const platform = getPlatform();
  const arch = getArch();
  const ext = platform === 'windows' ? '.exe' : '';
  const binaryName = `sage-${platform}-${arch}${ext}`;
  const targetExec = `sage${ext}`;

  const binDir = getBinaryInstallDir();
  fs.mkdirSync(binDir, { recursive: true });
  const destPath = path.join(binDir, targetExec);

  console.log(`\n🚀 Installing Sage CLI binary...`);
  console.log(`Detected Platform: ${platform} (${arch})`);

  let downloadUrl;
  if (REQUESTED_VERSION === 'latest') {
    downloadUrl = `https://github.com/${REPO}/releases/latest/download/${binaryName}`;
  } else {
    downloadUrl = `https://github.com/${REPO}/releases/download/${REQUESTED_VERSION}/${binaryName}`;
  }

  console.log(`Downloading ${binaryName} from: ${downloadUrl}`);
  const tmpFile = path.join(os.tmpdir(), `sage-${Date.now()}${ext}`);
  
  try {
    await downloadFile(downloadUrl, tmpFile);
    fs.copyFileSync(tmpFile, destPath);
    if (process.platform !== 'win32') {
      fs.chmodSync(destPath, 0o755);
    }
    fs.unlinkSync(tmpFile);
    console.log(`✅ Sage CLI installed to: ${destPath}`);
  } catch (err) {
    console.warn(`⚠️ Could not download prebuilt binary: ${err.message}`);
    console.warn(`If building from source, run: go build -o sage ./cmd/sage`);
  }

  return { destPath, binDir };
}

async function main() {
  console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
  console.log('       Sage & Universal Agent Skill Installer        ');
  console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');

  try {
    const { destPath, binDir } = await installBinary();
    const skillDirs = getTargetSkillDirs();

    console.log(`\n📦 Installing Skill into Agent directories:`);
    for (const dir of skillDirs) {
      await installSkill(dir);
      console.log(`  ✓ Installed to: ${dir}`);
    }

    console.log('\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
    console.log('✨ All installations completed successfully!\n');
    console.log('Quick verification:');
    console.log(`  1. Binary: Ensure "${binDir}" is in your PATH.`);
    console.log(`     Try running: sage --help`);
    console.log(`  2. Skills: Supported across agents & editors:`);
    console.log(`     - Universal agents (.agents/skills/sage)`);
    console.log(`     - Claude Code      (.claude/skills/sage)`);
    console.log(`     - Cursor AI        (.cursor/skills/sage)\n`);
  } catch (err) {
    console.error(`\n❌ Installation error: ${err.message}`);
    process.exit(1);
  }
}

if (require.main === module) {
  main();
}

module.exports = { main, installBinary, installSkill, getTargetSkillDirs };
