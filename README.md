# Seristack

[![Go Reference](https://pkg.go.dev/badge/github.com/TechXploreLabs/seristack.svg)](https://pkg.go.dev/github.com/TechXploreLabs/seristack)
[![License](https://img.shields.io/github/license/TechXploreLabs/seristack)](LICENSE)
[![Release](https://img.shields.io/github/v/release/TechXploreLabs/seristack?include_prereleases)](https://github.com/TechXploreLabs/seristack/releases)

**Define shell workflows in YAML. Run them from the CLI, serve them over HTTP, or expose them as MCP tools for AI IDEs.**

```yaml
# config.yaml
stacks:
  - name: build
    count: 1
    cmds:
      - echo "building..."

  - name: test
    count: 1
    dependsOn: [build]
    cmds:
      - echo "testing..."
```

```bash
seristack trigger -c config.yaml
```

## Features

* Stacks of shell commands in one YAML file
* Dependencies between stacks
* Sequential or concurrent execution, with repeat counts
* Variables with validation rules
* Share output between stacks
* Per-command timeouts that also stop the processes a command started
* HTTP endpoints with per-stack access rules and JSON audit logs
* MCP server for AI agents and IDEs
* Built-in shell (mvdan) by default, or Bash, PowerShell, Python, and others

## Install

```bash
# macOS and Linux
brew install TechXploreLabs/tap/seristack

# Linux
curl -fsSL https://raw.githubusercontent.com/TechXploreLabs/seristack/main/install.sh | bash

# Windows (PowerShell)
irm https://raw.githubusercontent.com/TechXploreLabs/seristack/main/install.ps1 | iex
```

Or download a binary from [Releases](https://github.com/TechXploreLabs/seristack/releases).

## Three ways to run it

```bash
seristack trigger -c config.yaml                 # CLI: run stacks once
seristack run -c config.yaml                     # HTTP: serve stacks as endpoints
seristack mcp -t streamableHTTP                  # MCP: serve stacks as AI tools
```

To serve a stack over HTTP, add `method` and `urlPath`. To expose it as an MCP tool, add a `description`:

```yaml
stacks:
  - name: health
    count: 1
    description: Check system health
    method: GET
    urlPath: /health
    cmds:
      - echo "ok"
```

## Security

Seristack runs shell commands with the permissions of the user who starts it. It is not a sandbox.

* It serves plain HTTP with no built-in authentication or TLS.
* Keep it on `127.0.0.1` behind a reverse proxy that handles TLS and authentication.
* Never expose it directly to the internet.

See the [Guide](docs/guide.md#security-considerations) before deploying.

## Documentation

* [Guide](docs/guide.md): executors, execution modes, timeouts, variables, sharing output, HTTP and MCP servers, deployment, access rules, audit logging
* [Configuration Reference](docs/config-reference.md): every YAML field

## Contributing

Issues and pull requests are welcome. Run `go test ./...` before submitting.

## License

Apache License 2.0