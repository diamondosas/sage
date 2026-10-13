# Support & Help for Sage

Welcome to the Sage support center! We are dedicated to ensuring a seamless experience using the Sage CLI, GitHub Action, and AI Agent MCP integration.

<!-- CHANGE HERE: Update support and contact links -->
- **Publisher / Maintainer:** diamondosas
- **Primary Support Email:** [diamondsazwuwa@gmail.com](mailto:diamondosazuwa@gmail.com)
- **GitHub Issue Tracker:** [https://github.com/diamondosas/sage/issues](https://github.com/diamondosas/sage/issues)
- **GitHub Discussions:** [https://github.com/diamondosas/sage/discussions](https://github.com/diamondosas/sage/discussions)

---

## Getting Help

### 1. Documentation & Usage
- **CLI Commands & Quickstart:** See [`README.md`](README.md)
- **AI Agent Integration (Cursor, Claude Code, Antigravity):** See [`skills/sage/SKILL.md`](skills/sage/SKILL.md) and [`AGENTS.md`](AGENTS.md)
- **GitHub Action Setup:** See [`action.yml`](action.yml) and [`SETUP.md`](SETUP.md)

### 2. Frequently Encountered Issues

#### Issue: `sage` command not found in GitHub Action
**Solution:** Ensure the installation step runs before `sage sync`. You can also run the 1-line installer:
```bash
curl -fsSL https://raw.githubusercontent.com/diamondosas/sage/main/install/install.sh | bash
```


#### Issue: Rate limiting on GitHub PR sync
**Solution:** Provide a `GITHUB_TOKEN` input in your workflow so Sage can utilize authenticated GitHub API rate limits (5,000 requests/hour vs 60 unauthenticated).

---

## Submitting Bug wReports & Feature Requests

Please report bugs and suggest features through GitHub Issues:
1. Go to [https://github.com/diamondosas/sage/issues/new](https://github.com/diamondosas/sage/issues/new).
2. Include your OS, Go version, and command line output.
3. Remove any sensitive tokens or secrets from logs before posting.

---

## Direct Inquiries & Security

For sensitive matters or publisher inquiries, email [diamondosazwa@gmail.com](mailto:diamondosazuwa@gmail.com). For security vulnerabilities, please refer to [`SECURITY.md`](SECURITY.md).
