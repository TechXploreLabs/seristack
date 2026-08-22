# Security Policy

## Supported versions

| Version | Supported |
|---|---|
| 0.4.x (latest) | ✅ Active |
| 0.3.x | ⚠️ Critical fixes only |
| < 0.3.0 | ❌ Not supported |

Always use the latest release. Older versions do not receive security updates.

---

## Reporting a vulnerability

**Do not report security vulnerabilities through public GitHub issues.**

If you discover a security vulnerability in Seristack, please report it privately so we can address it before public disclosure.

**How to report:**

Open a private security advisory on GitHub:
https://github.com/TechXploreLabs/seristack/security/advisories/new

Include as much of the following as possible:

- Description of the vulnerability
- The component affected (HTTP server, MCP server, shellexecutor, variable validation, audit log, etc.)
- Steps to reproduce
- Potential impact
- Any suggested fix if you have one

We will acknowledge your report within **72 hours** and provide an estimated timeline for a fix. We aim to release a patch within **14 days** for critical vulnerabilities.

---

## Disclosure policy

We follow **coordinated disclosure**:

1. You report the vulnerability privately
2. We confirm and investigate
3. We develop and test a fix
4. We release the patched version
5. We publish a security advisory crediting the reporter (unless you prefer to remain anonymous)

We ask that you give us reasonable time to fix the issue before any public disclosure.

---

## Security design of seristack

Understanding how seristack is designed helps assess the attack surface.

**Seristack executes shell commands.** This is by design — it is a shell automation engine. The security model assumes:

- Seristack is **not exposed directly to the public internet**
- A reverse proxy (nginx, Caddy) handles TLS and authentication
- Seristack runs bound to `127.0.0.1` by default
- Shell commands and variables in the YAML config are operator-defined and trusted
- HTTP-sourced variable values are validated against declared `varRules` before shell execution

**Variable validation**

Variables that can be overridden via HTTP must have `varRules` defined. Use `allowed_value` or `allowed_regex` with tight patterns to prevent injection. Variable values from HTTP are only accepted for keys pre-declared in the YAML config — unknown keys are dropped.

**Authorization**

The `access` block restricts HTTP stack execution by checking identity headers forwarded by the reverse proxy. This applies to HTTP endpoints only. MCP server access is controlled at the transport level.

**Audit log**

The audit log records every HTTP stack execution including identity, variable values, and result. Do not pass secrets as stack variables — they will appear in the audit log.

**Secrets**

Seristack has no built-in secrets management. Secrets required by shell commands should come from environment variables set in the process environment or from a secrets manager (HashiCorp Vault, AWS Secrets Manager, etc.) accessed inside the shell script. Do not put secrets in `vars` values in the YAML config.

---

## Security checklist for operators

Before deploying seristack in production:

- [ ] Bind to `127.0.0.1`, not `0.0.0.0`
- [ ] Put nginx or Caddy in front for TLS and authentication
- [ ] Define `varRules` for every HTTP-exposed variable
- [ ] Use `allowed_value` or tight `allowed_regex` patterns — prefer allowlists over denylists
- [ ] Set `access` blocks on sensitive stacks
- [ ] Enable the audit log with `--audit-log`
- [ ] Use `logrotate` to manage audit log file size
- [ ] Do not pass secrets as stack `vars`
- [ ] Review shell commands for injection risks before exposing them via HTTP
- [ ] Keep seristack updated to the latest version

---

## Known limitations

- The `access` block applies to HTTP endpoints only. MCP tool calls are not subject to per-stack authorization — secure the MCP server at the network or transport level.
- Seristack does not validate that the reverse proxy is correctly configured. A misconfigured nginx or Caddy that forwards unvalidated identity headers would bypass the access control model.
- Shell commands run with the same OS user as the seristack process. Run seristack as a dedicated low-privilege user with only the permissions your stacks require.

---

## Dependency vulnerabilities

Seristack uses `govulncheck` to scan for known vulnerabilities in dependencies. Every release is gated on a clean `govulncheck` result.

To check for vulnerabilities in your own installation:

```bash
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck github.com/TechXploreLabs/seristack@latest
```