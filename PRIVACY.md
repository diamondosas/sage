# Privacy Policy for Sage

**Last Updated:** October 12, 2026

This Privacy Policy describes how **Sage** ("we", "us", or "our") collects, uses, and safeguards information when you use the Sage CLI, GitHub Action, MCP Server, or associated tools.

<!-- CHANGE HERE: Update publisher contact information -->
**Contact Email:** [support@diamondosas.dev](mailto:support@diamondosas.dev)  
**Publisher:** diamondosas / Sage  
**Repository:** [https://github.com/diamondosas/sage](https://github.com/diamondosas/sage)  

---

## 1. Local-First Architecture

Sage is designed from the ground up as a **local-first** developer tool:
- **No External Database Required**: Sage stores its knowledge graph directly inside the repository workspace (by default in `.github/sage` or `.sage`) using an embedded BadgerDB key-value engine.
- **No Cloud Telemetry or Tracking**: Sage does **not** transmit usage telemetry, analytics, behavioral logs, or user identities to any Sage servers.
- **On-Premise / In-Runner Execution**: In GitHub Actions, Sage executes entirely within your ephemeral GitHub Actions runner environment.

---

## 2. Information Handled by Sage

When you run Sage (`sage sync`, `sage who`, `sage why`, etc.) or use the GitHub Action:
1. **Git Metadata**: Sage scans local commit history (author names, email addresses, commit timestamps, and modified file paths) to construct ownership and decision graphs. This information stays strictly within your repository.
2. **GitHub Tokens (`GITHUB_TOKEN`)**: If provided, your GitHub token is used ephemerally to fetch pull request discussions and merged issue references via the official GitHub REST API. The token is never stored, logged, or forwarded to third parties.
3. **LLM API Keys (`GEMINI_API_KEY`, `OPENAI_API_KEY`, etc.)**: If you enable AI decision synthesis, your API key is sent directly over TLS to your chosen AI provider (Google, OpenAI, Anthropic, or local Ollama). Sage does not intercept, log, or proxy your keys.

---

## 3. Third-Party Services

Depending on your configuration, Sage may interact with:
- **GitHub API**: Subject to [GitHub Privacy Statement](https://docs.github.com/en/site-policy/privacy-policies/github-privacy-statement).
- **Your Chosen LLM Provider**: When synthesizing PR decisions, prompt text (PR title, description, diff summary) is processed in accordance with your provider's terms (e.g., Google Cloud, OpenAI, Anthropic). No proprietary code files are sent unless referenced in PR summaries.

---

## 4. Data Retention and Deletion

Because all indexed data is stored locally in your repository filesystem:
- You retain complete ownership of your data at all times.
- To delete all stored knowledge graph data, simply remove the `.github/sage` directory from your repository.

---

## 5. Security

We take security seriously:
- We do not store secrets in persistent databases.
- The `.env` file generated during `sage init` is automatically appended to `.gitignore`.
- Security disclosures should be submitted following [`SECURITY.md`](SECURITY.md).

---

## 6. Updates to This Policy

We may update this Privacy Policy from time to time. Any changes will be posted in this file within the repository.

---

## 7. Contact Us

If you have questions regarding this Privacy Policy, please open an issue at [https://github.com/diamondosas/sage/issues](https://github.com/diamondosas/sage/issues) or email [support@diamondosas.dev](mailto:support@diamondosas.dev).
