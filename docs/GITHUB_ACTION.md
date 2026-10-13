# ⚡ Sage GitHub Action Guide

> **Autonomous Git and GitHub decision graph sync powered by embedded SQLite.**

---

## 🚀 Quick Usage

Add this step to any GitHub Actions workflow job:

```yaml
- name: Run Sage Knowledge Sync
  uses: diamondosas/sage@v1
  with:
    github_token: ${{ secrets.GITHUB_TOKEN }}
    llm_provider: 'google'
    llm_api_key: ${{ secrets.GEMINI_API_KEY }} # Optional
    max_commits: 500
```

---

## ⚙️ Inputs & Parameters

| Input | Required | Default | Description |
|---|---|---|---|
| `github_token` | No | `${{ github.token }}` | GitHub token for fetching PR discussions and linked issues. |
| `llm_provider` | No | `'google'` | Provider for PR decision summarization (`google`, `openai`, `anthropic`, `ollama`). |
| `llm_api_key` | No | `""` | API key for Gemini, Groq, or OpenAI. (Optional: commit graph indexes 100% offline without key). |
| `max_commits` | No | `'500'` | Maximum recent commits to scan for author recency scoring. |

---

## 🔄 Standard Push & Merged PR Workflow

```yaml
name: Sage Knowledge Graph Sync

on:
  push:
    branches: [ main ]
  pull_request:
    types: [ closed ]
  workflow_dispatch:

permissions:
  contents: write
  pull-requests: read

jobs:
  sync:
    name: Sync Repository Knowledge
    if: github.event_name != 'pull_request' || github.event.pull_request.merged == true
    runs-on: ubuntu-latest

    steps:
      - name: Check out repository
        uses: actions/checkout@v4
        with:
          fetch-depth: 0 # Full commit history needed

      - name: Run Sage Action
        uses: diamondosas/sage@v1
        with:
          github_token: ${{ secrets.GITHUB_TOKEN }}
          llm_provider: 'google'
          llm_api_key: ${{ secrets.GEMINI_API_KEY }}
          max_commits: 500

      - name: Commit & Push Knowledge Database
        run: |
          git config --local user.email "action@github.com"
          git config --local user.name "Sage Bot"
          if [ -d ".github/sage" ]; then
            git add .github/sage
            git diff --staged --quiet || git commit -m "chore(sage): update knowledge graph [skip ci]"
            git push || true
          fi
```
