# Seristack

[![Go Reference](https://pkg.go.dev/badge/github.com/TechXploreLabs/seristack.svg)](https://pkg.go.dev/github.com/TechXploreLabs/seristack)
[![Go Version](https://img.shields.io/badge/go-1.26.6-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/github/license/TechXploreLabs/seristack)](LICENSE)
[![Release](https://img.shields.io/github/v/release/TechXploreLabs/seristack?include_prereleases)](https://github.com/TechXploreLabs/seristack/releases)

**One YAML configuration. CLI commands, HTTP endpoints, and MCP tools.**

Seristack is a lightweight automation engine for DevOps, Platform, SRE, and Cloud teams.

Define shell-based workflows in YAML, manage dependencies and variables, execute them locally or through HTTP, and expose selected stacks as MCP tools for AI agents and IDE integrations.

[GitHub Repository](https://github.com/TechXploreLabs/seristack)

## Why Seristack?

Operational workflows often live as shell scripts, runbooks, CI jobs, and undocumented procedures.

Seristack provides a single configuration layer for turning those workflows into reusable execution stacks that can be:

* Run from the CLI
* Exposed as HTTP endpoints
* Exposed as MCP tools
* Protected with per-stack authorization
* Audited with structured JSON logs
* Composed using dependencies and shared results

## Documentation

* [Configuration Reference](docs/config-reference.md)

## Features

* 🚀 Run multiple command stacks from a single YAML configuration
* 🔁 Execute stacks sequentially or concurrently
* 🔢 Repeat stack execution with configurable counts
* 🔗 Define dependencies between stacks
* 🧩 Variable substitution with validation rules
* 📦 Share output and results between dependent stacks
* 🌐 Expose stacks as HTTP endpoints
* 🔐 Per-stack authorization using identity headers
* 📋 Structured JSON audit logging
* 🧠 Run as an MCP server for AI agents and IDE integrations
* 🛠 Support for mvdan shell, Bash, sh, and PowerShell
* ⏱ Per-stack execution timeouts
* 📁 Configurable working directories
* 🛡 Allow and deny rules for stack variables

---

## Installation

### Homebrew — macOS and Linux

```bash
brew install TechXploreLabs/tap/seristack
```

### Linux — installer

```bash
curl -fsSL https://raw.githubusercontent.com/TechXploreLabs/seristack/main/install.sh | bash
```

### Linux — release archive

1. Go to [Seristack Releases](https://github.com/TechXploreLabs/seristack/releases).
2. Download the latest:

```text
seristack_VERSION_linux_ARCH.tar.gz
```

For example:

```text
seristack_0.4.1_linux_amd64.tar.gz
```

3. Extract the archive:

```bash
tar -xzf seristack_VERSION_linux_ARCH.tar.gz
```

4. Install the binary:

```bash
sudo mv seristack /usr/local/bin/
sudo chmod +x /usr/local/bin/seristack
```

5. Verify:

```bash
seristack --help
```

### Windows — installer

Run PowerShell as a user with permission to install the binary:

```powershell
irm https://raw.githubusercontent.com/TechXploreLabs/seristack/main/install.ps1 | iex
```

### Windows — release archive

1. Go to [Seristack Releases](https://github.com/TechXploreLabs/seristack/releases).
2. Download the Windows release archive:

```text
seristack_VERSION_windows_ARCH.tar.gz
```

For example:

```text
seristack_0.4.1_windows_amd64.tar.gz
```

3. Extract the archive with a tool that supports `.tar.gz`.

4. Place `seristack.exe` in a directory included in your `%PATH%`.

5. Verify:

```powershell
seristack --help
```

---

## Configuration

Seristack uses YAML to define execution stacks.

For a complete description of all configuration fields, see the [Configuration Reference](docs/config-reference.md).

### Example

```yaml
stacks:

  - name: stack1
    workDir: ./
    description: Print welcome message
    method: GET
    urlPath: /show
    continueOnError: false
    count: 3
    timeouts: 1h
    executionMode: PARALLEL

    vars:
      - name: samplekey
        value: samplevalue
        required: true
        allowed_value:
          - samplevalue
          - devvalue

    cmds:
      - |
        export samplekey={{.Vars.samplekey}}
        echo $samplekey
        echo "count={{.Count.index}}"
        echo "Hey I'm Seristack!"

  - name: stack2
    workDir: ./
    continueOnError: false
    count: 3
    executionMode: SEQUENTIAL

    vars:
      - name: env
        value: Dev

    dependsOn:
      - stack1

    cmds:
      - |
        echo "{\"index\": {{.Count.index}}, \"step\": \"metadata\", \"status\": \"ok\"}"

      - |
        echo "{\"index\": {{.Count.index}}, \"step\": \"metrics\", \"value\": $((RANDOM % 100))}"

    output: |
      echo "--- Aggregation Summary ---"
      echo '{{.Self.result}}' | grep "^{" | jq -s '{
        total_records: length,
        environment: "{{.Vars.env}}",
        results: .
      }'
```

---

## Running stacks

### Trigger all stacks

```bash
seristack trigger -c config.yaml
```

### Trigger a specific stack

```bash
seristack trigger -c config.yaml -s stack1
```

### Start the HTTP server

```bash
seristack run -c config.yaml
```

### Start the MCP server

```bash
seristack mcp -t streamableHTTP
```

---

## HTTP server

The HTTP server exposes configured stacks as HTTP endpoints.

For example:

```yaml
stacks:
  - name: system-health
    method: GET
    urlPath: /health
    cmds:
      - echo "system-health is good"
```

Start the server:

```bash
seristack run \
  --config config.yaml \
  --addr 127.0.0.1 \
  --port 8080
```

A reverse proxy such as nginx or Caddy can expose the service externally while Seristack remains bound to localhost.

---

## Production deployment

Seristack executes shell commands and should **not be exposed directly to the public internet**.

A recommended architecture is:

```text
Client
  │
  ▼
nginx / Caddy
  │
  ├── TLS termination
  ├── Authentication
  ├── Rate limiting
  │
  ▼
Seristack
127.0.0.1
  │
  ├── Authorization
  └── Command execution
```

Start Seristack on localhost:

```bash
seristack run \
  --config config.yaml \
  --addr 127.0.0.1 \
  --port 8080
```

For a remote MCP deployment, the same pattern can be used:

```text
AI Agent / IDE
      │
      ▼
    HTTPS
      │
      ▼
nginx / oauth2-proxy
      │
      ├── TLS
      ├── OIDC authentication
      └── Identity headers
      │
      ▼
Seristack MCP
127.0.0.1:8081
      │
      ├── Stack authorization
      └── Command execution
```

Authentication should be handled by the identity-aware proxy or gateway, while Seristack performs authorization at the individual stack level.

---

## Per-stack authorization

Seristack can restrict individual stacks using identity headers forwarded by an authenticated reverse proxy.

For example:

```yaml
stacks:

  - name: deploy-production
    method: POST
    urlPath: /deploy/production

    matchAccess: ANY

    access:
      - headerName: X-Auth-Request-Groups
        headerValue:
          - sre
          - platform

      - headerName: X-Auth-Request-Roles
        headerValue:
          - admin

    count: 1

    cmds:
      - ./deploy.sh
```

### Access matching

`matchAccess` controls how multiple access rules are evaluated.

#### ANY

```yaml
matchAccess: ANY
```

Access is granted when **at least one** access rule matches.

This is OR logic.

#### ALL

```yaml
matchAccess: ALL
```

Access is granted only when **every** access rule matches.

This is AND logic.

#### No access rules

If a stack does not define an `access` block, there is no stack-level access restriction.

The authentication layer should still protect the service itself in production.

### Identity headers

The actual headers depend on the authentication provider and reverse proxy.

| Identity provider   | Common proxy         | Example identity information                                            |
| ------------------- | -------------------- | ----------------------------------------------------------------------- |
| Entra ID / Azure AD | oauth2-proxy         | `X-Auth-Request-Groups`, `X-Auth-Request-Roles`, `X-Auth-Request-Email` |
| GCP                 | IAP                  | `X-Goog-Authenticated-User-Email`                                       |
| AWS Cognito         | ALB                  | `X-Amzn-Oidc-Data`                                                      |
| OCI IAM             | nginx + oauth2-proxy | `X-Auth-Request-Groups`, `X-Auth-Request-Email`                         |
| Okta / Auth0        | oauth2-proxy         | Identity headers configured by the proxy                                |

The headers must be configured and trusted only when they originate from your authenticated proxy.

Do not allow untrusted external clients to directly supply authorization headers.

---

## Audit logging

Seristack can write a structured JSON audit log for stack executions.

Enable audit logging:

```bash
seristack run \
  --config config.yaml \
  --audit-log /var/log/seristack/audit.log
```

Audit entries include information such as:

* Timestamp
* Event
* Stack name
* HTTP path and method
* Source IP when available
* Identity headers
* Variables
* Success/failure
* Execution duration
* Output
* Error information

Example:

```json
{
  "timestamp": "2026-09-19T10:00:00Z",
  "event": "stack_executed",
  "stack": "system-health",
  "success": true,
  "duration_ms": 42,
  "output": "system-health is good"
}
```

Use `logrotate` or an equivalent logging system to manage log rotation.

### Do not put secrets in stack variables

Variables may be included in audit records.

Avoid passing passwords, tokens, API keys, or other secrets as stack variables.

Prefer:

* Environment variables
* Secret managers
* Workload identity
* External credential providers

---

## MCP server

Seristack can expose configured stacks as MCP tools.

Start a Streamable HTTP MCP server:

```bash
seristack mcp \
  -t streamableHTTP \
  --addr 127.0.0.1 \
  --port 8081
```

Stacks with a `description` can be exposed as MCP tools.

AI agents and MCP-compatible IDEs can then discover and invoke the available stacks.

A production deployment should place the MCP server behind HTTPS and an authentication layer:

```text
MCP Client
    │
    ▼
HTTPS
    │
    ▼
nginx / oauth2-proxy
    │
    ├── Authentication
    ├── TLS
    └── Identity headers
    │
    ▼
Seristack MCP
127.0.0.1:8081
    │
    ▼
Per-stack authorization
    │
    ▼
Shell execution
```

---

## Variable validation

Variables can define validation and authorization rules.

```yaml
vars:

  - name: env
    value: staging
    required: true
    allowed_value:
      - staging
      - production

  - name: version
    required: true
    allowed_regex: regex("^[a-zA-Z0-9._-]+$")

  - name: command
    denied_regex: regex("(?i)rm|delete|drop")
```

Only variables declared in `vars` can be overridden by HTTP requests or MCP arguments.

Undeclared variables supplied through external inputs are ignored.

Validation rules can include:

* `required`
* `allowed_value`
* `denied_value`
* `allowed_regex`
* `denied_regex`

---

## Dependencies

Stacks can depend on other stacks:

```yaml
stacks:

  - name: build
    cmds:
      - ./build.sh

  - name: test
    dependsOn:
      - build
    cmds:
      - ./test.sh

  - name: deploy
    dependsOn:
      - test
    cmds:
      - ./deploy.sh
```

Seristack resolves the dependency order before execution.

This allows larger workflows to be composed from smaller reusable stacks.

---

## Execution modes

Stacks can be configured to execute sequentially or concurrently.

### Sequential

```yaml
executionMode: SEQUENTIAL
```

Commands or repeated executions run in sequence.

### Parallel

```yaml
executionMode: PARALLEL
```

Independent executions can run concurrently.

Example:

```yaml
stacks:

  - name: health-check
    executionMode: PARALLEL
    count: 3

    cmds:
      - |
        echo "health check {{.Count.index}}"
```

---

## Command execution

Seristack supports shell-based execution using the configured shell execution mechanisms.

Examples include:

```yaml
cmds:
  - echo "hello"
```

Multi-line commands:

```yaml
cmds:
  - |
    echo "Starting"
    ./script.sh
    echo "Finished"
```

The configured execution environment determines which shell syntax is available.

---

## Testing

Run the complete test suite:

```bash
go test ./...
```

To force tests to execute without using Go's test cache:

```bash
go test -count=1 ./...
```

For verbose output and individual test execution times:

```bash
go test -count=1 -v ./...
```

Run tests for a specific package:

```bash
go test -count=1 -v ./pkg/opsy
```

Run the race detector:

```bash
CGO_ENABLED=1 go test -race ./...
```

The race detector requires a working C compiler/toolchain on supported platforms.

---

## `seristack run` command reference

```text
seristack run [flags]

Flags:

  -c, --config string
        config file (default "config.yaml")

  -a, --addr string
        bind address (default "127.0.0.1")

  -p, --port string
        server port (default "8080")

      --audit-log string
        path to audit log file
        enables audit logging when set

      --identity-header strings
        map identity header to a key:
        "user=X-Auth-Request-Email"
        repeatable
```

Run:

```bash
seristack run --help
```

for the current command and flag reference.

---

## Security considerations

Seristack executes commands on the host where it runs.

Before deploying it in a production environment:

1. Keep Seristack bound to localhost or a private network.
2. Put an authenticated reverse proxy or gateway in front of externally accessible endpoints.
3. Use HTTPS/TLS for remote access.
4. Do not trust identity headers supplied directly by clients.
5. Restrict which users or groups can execute sensitive stacks.
6. Avoid putting secrets into stack variables.
7. Review and rotate audit logs.
8. Run Seristack with the minimum operating-system privileges required.
9. Restrict filesystem and network permissions where possible.
10. Review shell commands carefully before exposing them through HTTP or MCP.

MCP access should be treated with the same security considerations as an HTTP API because MCP clients can invoke the tools exposed by the server.

---

## Support the project

If Seristack helps your team turn shell scripts or operational runbooks into reusable CLI commands, HTTP APIs, or MCP tools, consider:

* ⭐ Starring the repository
* 🐛 Opening issues
* 💡 Sharing feedback
* 🧪 Adding examples and tests
* 🔧 Contributing improvements

[GitHub Repository](https://github.com/TechXploreLabs/seristack)

---

## License

Apache License 2.0
