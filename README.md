# 🌿 Sage

> **Shared knowledge of entire codebases for Humans and LLMs.**

Sage is a lightweight tool that gives your AI coding assistants (Claude, Cursor, Copilot, Antigravity) the background context, Git history, and team decisions behind your code.

Instead of making you re-explain your architecture over and over, Sage indexes your repository into **one simple file** (`.github/sage/sage.db`) so your AI always knows *who* built what and *why* it was built that way.

---

### 💡 Why Sage?

When an AI reads your code, it only sees the current files. It doesn't know:
- **Who to ask** when something breaks.
- **Why a weird workaround exists** from a PR discussion six months ago.
- **What architectural decisions** your team already agreed on.

Sage connects these dots automatically:

```
[Person] ──(Who owns it?)──► [File] ◄──(Why does it exist?)── [Commit & PR Decision]
```

---

### ✨ What It Does

* **🌐 Interactive Architecture Visualizer (`sage ui`):** Launch a local, zero-friction web visualizer grouping files into high-level domain clusters with deep file inspection (owners, experts, exports, dependencies, and PR decision history).
* **📊 High-Level Context (`sage overview`):** Instantly get repository-wide architecture statistics, top files, and ownership metrics.
* **🔎 Targeted Knowledge Queries (`sage query`):** Run full-text search or SQL queries directly against `.github/sage/sage.db`.
* **📦 Just One Clean File:** Everything is stored inside a single SQLite file (`.github/sage/sage.db`). No servers, no extra setup, no messy folders.

---

### 🚀 Quick Start

#### 1. Initialize your project
```bash
sage init
```
This sets up your local `.github/sage/` directory and creates agent instructions for your AI.

#### 2. Sync your repository knowledge
```bash
sage sync
```
Sage reads your Git history and builds your local knowledge base in seconds.

#### 3. Explore your codebase

**Launch the interactive architecture visualizer:**
```bash
sage ui
```
Opens your browser to `http://127.0.0.1:8080` (or use `--no-browser` and `--port`).

**Get a quick overview of your repository:**
```bash
sage overview
```

**Query knowledge or search records:**
```bash
sage query "auth"
```

---

### 🤖 AI Agent Skills Setup

Install Sage skills into all detected coding assistants (**Antigravity**, **Cursor**, **Claude Code**, **Copilot**, **Windsurf**) in 1 command:

```bash
sage setup -i
# or
npx sage-brain setup
```

See the [Agent Skills Setup Guide](docs/SKILLS_SETUP.md) for assistant-specific manual configurations.

---

### ⚡ GitHub Action & Nightly Cron Sync

Keep your knowledge base updated automatically using the official GitHub Action (`diamondosas/sage@v1`):

```yaml
- name: Run Sage Knowledge Sync
  uses: diamondosas/sage@v1
  with:
    github_token: ${{ secrets.GITHUB_TOKEN }}
    max_commits: 500
```

Check out the ready-to-copy [Scheduled Cron Workflow](docs/CRON_WORKFLOW.md) and [GitHub Action Guide](docs/GITHUB_ACTION.md).

---

### 📖 Full Documentation

Visit the interactive dark-mode documentation website at **[`docs/index.html`](docs/index.html)** or browse the guides in **[`docs/`](docs/)**:
- 🤖 [Agent Skills Setup](docs/SKILLS_SETUP.md)
- ⚡ [GitHub Action Guide](docs/GITHUB_ACTION.md)
- ⏰ [Scheduled Cron Workflow](docs/CRON_WORKFLOW.md)

---

### 📄 License

MIT