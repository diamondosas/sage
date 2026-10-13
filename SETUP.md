# GitHub Marketplace Compliance Checklist

This document tracks how Sage satisfies all 10 requirements for listing on the GitHub Marketplace (for both GitHub Actions and GitHub Apps).

| # | Requirement | Status | Fulfilled By / Reference |
|---|---|---|---|
| **1** | Listings must not actively persuade users away from GitHub. | :white_check_mark: Fulfilled | All documentation, setup flows, and links operate entirely within GitHub repositories, actions, and open-source standards without third-party checkout redirects. |
| **2** | Listings must include valid contact information for the publisher. | :white_check_mark: Fulfilled | Documented in [`SUPPORT.md`](SUPPORT.md), [`PRIVACY.md`](PRIVACY.md), [`MARKETPLACE.md`](MARKETPLACE.md), and [`action.yml`](action.yml) (`support@diamondosas.dev`, publisher `diamondosas`). |
| **3** | Listings must have a relevant description of the application. | :white_check_mark: Fulfilled | Comprehensive copy and architecture description provided in [`MARKETPLACE.md`](MARKETPLACE.md) and [`README.md`](README.md). |
| **4** | Listings must specify a pricing plan. | :white_check_mark: Fulfilled | **Free Community Plan** ($0.00/month, unlimited runs, local SQlite storage) detailed in [`MARKETPLACE.md`](MARKETPLACE.md). |
| **5** | Listings must have a valid link to a privacy policy. | :white_check_mark: Fulfilled | [`PRIVACY.md`](PRIVACY.md) covers local-first SQlite storage, ephemeral token handling, and third-party LLM privacy. |
| **6** | Listings must provide a method to receive support through a valid support link and/or a support email address. | :white_check_mark: Fulfilled | [`SUPPORT.md`](SUPPORT.md) links to GitHub Issues, discussions, and [support@diamondosas.dev](mailto:support@diamondosas.dev). |
| **7** | All additional links in a listing, such as Terms of Service or a Status Page, must work and resolve to a relevant page. | :white_check_mark: Fulfilled | Terms of Service provided in [`TERMS.md`](TERMS.md); security guidelines in [`SECURITY.md`](SECURITY.md); license in [`LICENSE`](LICENSE). |
| **8** | Apps must provide value to customers and integrate with the platform in some way beyond authentication. | :white_check_mark: Fulfilled | Sage scans Git history, PR discussions, and CODEOWNERS to generate knowledge graphs, recency-decay expertise scores, and MCP assistant tools. |
| **9** | Apps must be publicly available in GitHub Marketplace and cannot be in public preview or available by invite only. | :white_check_mark: Fulfilled | Sage is open-source (Apache-2.0), public on GitHub (`diamondosas/sage`), and freely available to any repository. |
| **10** | Apps must have webhook events set up to notify the publisher of any plan changes or cancellations using the GitHub Marketplace API. | :white_check_mark: Fulfilled | Webhook parser and HTTP handler implemented in [`internal/github/marketplace.go`](internal/github/marketplace.go) with signature verification and unit tests in [`internal/github/marketplace_test.go`](internal/github/marketplace_test.go). |

---

## Action Publishing Quick Steps

1. Check that root [`action.yml`](action.yml) is present.
2. Draft a new release on GitHub tagged `v1.0.0` (and `v1`).
3. Check the **"Publish this Action to the GitHub Marketplace"** box.
4. Select category **Code quality** & **Continuous integration**.
5. Publish release.