# Security Policy

## Reporting a vulnerability

Please **do not** open a public issue for security problems.

Use GitHub's private reporting:
**[Report a vulnerability](https://github.com/Ritiksuman07/Vajra/security/advisories/new)**
(Security tab → *Report a vulnerability*), or open a
[draft security advisory](https://github.com/Ritiksuman07/Vajra/security/advisories).

Include:

1. What the issue is and where (file/package).
2. Steps to reproduce, or a minimal proof of concept.
3. Impact — what an attacker could do.
4. Affected version(s) and platform.
5. Suggested fix, if you have one.

We aim to acknowledge within **72 hours** and to ship a fix for high-severity
issues within **7 days**. Credit is given in the release notes unless you ask
to stay anonymous.

## Supported versions

Only the latest release receives security fixes.

## Threat model & scope

Vajra is a local-first desktop tool. Understanding its boundaries helps you
judge what is (and isn't) a vulnerability:

- **The HTTP server is unauthenticated and binds to `0.0.0.0:<port>`** when
  you run `vajra serve`/`vajra web`. It is intended for `localhost`. Binding
  it to a public interface exposes it to your network — use a firewall,
  reverse proxy with auth, or SSH tunnel. Reports about the *absence of
  built-in auth* are known and expected, not vulnerabilities.
- **API keys** are stored in plain text in `~/.vajra/config.json`, like most
  local CLI tools (`gh`, `aws`, …). The HTTP API never returns key material.
  Protect the file's permissions.
- **Model output is untrusted.** Vajra does not execute model output. The
  verification pipeline is a set of heuristics and must not be relied on as a
  security control.
- **Provider traffic** goes only to endpoints you configure.

### In scope

- Leaking API keys through the API, logs or the UI.
- Remote code execution, path traversal, or unsafe file handling.
- Supply-chain issues in dependencies.
- Crashes or resource exhaustion triggerable locally from untrusted model
  output or crafted input.

### Out of scope

- Missing authentication on the local server (see above).
- Anything requiring an attacker to already control your machine.
- Inaccurate or hallucinated model answers (that's what verification is for).
- Vulnerabilities in Ollama, a model, or a third-party provider — report those
  upstream.

## Hardening tips

- Keep the server on localhost; don't port-forward it.
- Prefer a separate OS user or sandbox for untrusted models.
- Rotate any cloud key you have pasted into a shared shell.
