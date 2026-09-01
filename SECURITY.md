# Security Policy

This is a self-hosted application maintained by a solo developer (with AI
assistance — see [DISCLOSURES.md](DISCLOSURES.md)). It is built for trusted,
personal/family use, and I take vulnerabilities that could expose your
collection data seriously. This document explains how to report them and what
to expect.

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 1.0.x   | :white_check_mark: |
| < 1.0   | :x:                |

Security fixes are released as patch versions on `main` and tagged with
[semver](https://semver.org/). Because this app is self-hosted, **you** control
when you receive fixes — pull the newest image/tag and restart when you're able.

## Reporting a Vulnerability

**Please do not open a public GitHub issue for security problems.** A public
issue is an unpatched vulnerability disclosure.

Instead, use **GitHub's Private Vulnerability Reporting**:

👉 **[Report a vulnerability](https://github.com/Toph4er/family-library/security/advisories/new)**

This creates a private, GitHub-managed draft security advisory that only you
and the maintainer can see. If you can't use that feature, email the
maintainer directly (find a contact on the
[GitHub profile](https://github.com/Toph4er)) with the word *security* in the
subject line.

### What makes a helpful report

- The version/tag and how you deploy (Docker Compose, reverse proxy config, etc.)
- Steps to reproduce — a minimal sequence that demonstrates the problem
- Impact — what an attacker gains, and what access they'd need first
- If you have one, a suggested fix (absolutely optional)

### What to expect from the maintainer

- **First response within 14 days** — realistically much sooner, but this is a
  solo-maintained personal project and that's the honest ceiling.
- If the report is confirmed, a fix will be developed privately where possible
  and shipped in the next release.
- With your agreement, the finding will be published as a GitHub Security
  Advisory so other self-hosters know to update, with credit to you.

You will never be publicly named without your consent, and reports are handled
in good faith — never as an opportunity to ambush a volunteer project.

### Scope

**In scope:** anything reachable through the application itself —
authentication/session handling, authorization (guest vs. admin), CSRF,
SQL injection, XSS/CSP bypasses, path traversal, secrets leaking in logs or
responses, and unsafe handling of your data.

**Out of scope (not vulnerabilities):**

- Attacks that require physical/network access to the machine hosting the app
- Weak or leaked credentials configured by the operator (e.g., a guessed
  `GUEST_PASSWORD` or `ADMIN_PASSWORD` — these are yours to choose well)
- Missing hardening of your reverse proxy, TLS setup, or host OS
- Denial of service
- Findings requiring a browser with JavaScript disabled protections,
  browser extensions, or already-deprecated browsers
- Third-party service behavior (e.g., what the Open Library API does with your
  ISBN queries — see [DISCLOSURES.md](DISCLOSURES.md))

### Bug bounty

There is no bug bounty program. A thank-you in the advisory credit line is the
currency. 🙏

## Security Notes for Self-Hosters

A few operator-side things that meaningfully improve your deployment:

- **Keep `SESSION_SECRET` secret** (32+ random chars; `openssl rand -hex 32`).
  Rotating it invalidates all sessions — do so if you suspect a leak.
- **The guest password is configured only via the `GUEST_PASSWORD` environment
  variable** and is re-applied from the environment on every container start.
  There is deliberately no in-app way to change it: to rotate it, update your
  `.env` and restart. Anyone who can read your `.env` can already read your
  database, so treat that file as sensitive (`chmod 600 .env`).
- **Run it on a trusted network or behind authentication.** This app has no
  multi-tenant threat model; it's designed for family use.
- **Back up your SQLite database.** It is the entire system of record. A
  periodic `sqlite3 library.db ".backup backup.db"` from the host (or copying
  the file while the app is stopped) is sufficient for a personal install.
