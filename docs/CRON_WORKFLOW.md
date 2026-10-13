# ⏰ Scheduled Cron Job Workflow for Sage

> **Run Sage on a nightly cron timer to recalculate expertise scores, map recent commits, and synthesize PR decisions with zero manual effort.**

---

## 📋 Complete Workflow: `.github/workflows/sage-cron-sync.yml`

Copy and paste this file into `.github/workflows/sage-cron-sync.yml` in your repository:

```yaml
name: Sage Nightly Knowledge Sync

on:
  # Runs automatically every night at 2:00 AM UTC
  schedule:
    - cron: '0 2 * * *'
  
  # Trigger on pushes to main / master
  push:
    branches:
      - main
      - master

  # Allow manual 1-click execution in GitHub Actions UI
  workflow_dispatch:

permissions:
  contents: write
  pull-requests: read

concurrency:
  group: sage-sync-${{ github.ref }}
  cancel-in-progress: true

jobs:
  sage-sync:
    name: Sync Knowledge Graph & Expertise
    runs-on: ubuntu-latest

    steps:
      # Step 1: Checkout the full Git history
      - name: Check out repository (full history)
        uses: actions/checkout@v4
        with:
          fetch-depth: 0 # CRITICAL: full history needed for Git commit graph

      # Step 2: Execute Sage GitHub Action
      - name: Run Sage Sync Action
        uses: diamondosas/sage@v1
        with:
          github_token: ${{ secrets.GITHUB_TOKEN }}
          llm_provider: 'google'
          llm_api_key: ${{ secrets.GEMINI_API_KEY }}
          max_commits: 1000

      # Step 3: Commit and push updated SQLite database
      - name: Commit & Push Updated Knowledge Base
        run: |
          git config --local user.email "action@github.com"
          git config --local user.name "Sage Bot"
          
          if [ -d ".github/sage" ]; then
            git add .github/sage
            # Commit only if changes exist; [skip ci] prevents infinite loop
            if ! git diff --staged --quiet; then
              git commit -m "chore(sage): nightly knowledge graph update [skip ci]"
              git push origin ${{ github.ref_name }}
              echo "✓ Successfully pushed updated Sage database to repository"
            else
              echo "ℹ No changes detected in .github/sage; skipping commit."
            fi
          fi
```

---

## 🏗️ Program & Repository Layout

```plaintext
my-repository/
├── .github/
│   ├── sage/
│   │   ├── sage.db              <-- The single SQLite database (committed to Git)
│   │   └── config.json          <-- Repository configuration
│   └── workflows/
│       └── sage-cron-sync.yml    <-- Scheduled cron action workflow
├── AGENTS.md                     <-- AI agent instructions (sage who / why)
├── CODEOWNERS                    <-- Formal ownership definitions
├── src/                          <-- Application code
└── README.md
```

---

## 🔑 Permissions & Configuration Notes

1. **`permissions: contents: write`**: Required for the workflow runner to push changes back to your branch.
2. **`permissions: pull-requests: read`**: Required to fetch PR titles, discussion debates, and reviews.
3. **`fetch-depth: 0`**: Required so `git log` can trace past commits. Without this, only the latest commit is downloaded.
4. **`[skip ci]`**: Included in the commit message to prevent GitHub Actions from recursively triggering another build when the bot commits `sage.db`.
5. **Secrets**:
   - `${{ secrets.GITHUB_TOKEN }}` is provided automatically by GitHub Actions.
   - `GEMINI_API_KEY` (optional) can be added under **Repository Settings → Secrets and variables → Actions**.
