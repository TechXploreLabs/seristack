# seristack

[![Go Reference](https://pkg.go.dev/badge/github.com/TechXploreLabs/seristack.svg)](https://pkg.go.dev/github.com/TechXploreLabs/seristack)
[![Go Version](https://img.shields.io/badge/go-1.25.5-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/github/license/TechXploreLabs/seristack)](LICENSE)
[![Release](https://img.shields.io/github/v/release/TechXploreLabs/seristack?include_prereleases)](https://github.com/TechXploreLabs/seristack/releases)

**Run shell workflows via CLI, HTTP, or AI agents**

Seristack is a lightweight automation engine designed for DevOps, Platform, SRE, and Cloud teams. Define shell workflows in YAML, manage dependencies, expose them as HTTP endpoints, and let AI agents call them as MCP tools.

[seristack](https://github.com/TechXploreLabs/seristack)

Documentation:

- [Config Reference](docs/config-reference.md)

## Features

    🚀 Run multiple command stacks from a single config
    🔁 Repeat stacks with serial or concurrent execution
    🔗 Define dependencies between stacks
    🧩 Variable substitution with validation rules
    📦 Share output between stacks
    🌐 Expose stacks as HTTP endpoints
    🔐 Per-stack authorization via identity headers
    📋 Structured audit log for every stack execution
    🧠 Run as an MCP server for AI agent and IDE integrations
    🛠 Works with mvdan shell (default), Bash, sh, and PowerShell


## Installation

### Using Homebrew (Mac and Linux)

```bash
brew install TechXploreLabs/tap/seristack
```

### Linux (using release archive)

1. Go to [Seristack Releases](https://github.com/TechXploreLabs/seristack/releases) and download the latest `seristack_VERSION_linux_ARCH.tar.gz`.
2. Extract the archive:
   ```bash
   tar -xzf seristack_VERSION_linux_ARCH.tar.gz
   ```
3. Move the binary to your PATH:
   ```bash
   sudo mv seristack /usr/local/bin/
   sudo chmod +x /usr/local/bin/seristack
   ```
4. Verify:
   ```bash
   seristack --help
   ```

### Windows (using release archive)

1. Go to [Seristack Releases](https://github.com/TechXploreLabs/seristack/releases) and download the latest `seristack_VERSION_windows_ARCH.zip`.
2. Extract and move `seristack.exe` to a folder in your `%PATH%`.
3. Verify:
   ```powershell
   seristack --help
   ```


## Sample config

For a full explanation of every YAML attribute, see the [Config Reference](docs/config-reference.md).

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
        allowed_value: [samplevalue, devvalue]
    cmds:
      - |
        export samplekey={{.Vars.samplekey}}
        echo $samplekey
        echo "count={{.Count.index}}"
        echo "Hey i'm seristack!"

  - name: stack2
    workDir: ./
    continueOnError: false
    count: 3
    executionMode: SEQUENTIAL
    vars:
      - name: env
        value: Dev
    dependsOn: [stack1]
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

## Running stacks

```bash
# Trigger all stacks
seristack trigger -c config.yaml

# Trigger a specific stack
seristack trigger -c config.yaml -s stack1

# Start the HTTP server
seristack run -c config.yaml

# Start the MCP server
seristack mcp -t streamableHTTP
```


## Production deployment

Seristack executes shell commands. Never expose it directly to the public internet.

The production pattern is:

```
Client → nginx / caddy (TLS, AuthN) → seristack on 127.0.0.1 (AuthZ, execution)
```

Start seristack bound to localhost:

```bash
seristack run --config config.yaml --addr 127.0.0.1 --port 8080
```

The reverse proxy handles TLS, authentication, and rate limiting. Seristack handles per-stack authorization and execution.


## Per-stack authorization

Seristack checks identity headers forwarded by nginx/caddy after they validate the user. Every IdP (Entra ID, OCI IAM, GCP IAM, AWS Cognito, Okta) forwards group and role information as HTTP headers via oauth2-proxy or a native OIDC integration.

Add an `access` block to any stack to restrict who can execute it:

```yaml
stacks:
  - name: deploy-production
    method: POST
    urlPath: /deploy/production
    matchAccess: ANY        # ANY (default) or ALL
    access:
      - headerName: "X-Auth-Request-Groups"
        headerValue: ["sre", "platform"]
      - headerName: "X-Auth-Request-Roles"
        headerValue: ["admin"]
    count: 1
    cmds:
      - ./deploy.sh
```

- `matchAccess: ANY` — access is granted if any one rule matches (OR logic). Default.
- `matchAccess: ALL` — every rule must match (AND logic).
- No `access` block — any authenticated user can execute the stack.

The header names depend on your IdP and proxy:

| IdP | Proxy | Headers forwarded |
|---|---|---|
| Entra ID / Azure AD | oauth2-proxy | `X-Auth-Request-Groups`, `X-Auth-Request-Roles`, `X-Auth-Request-Email` |
| GCP IAM | IAP | `X-Goog-Authenticated-User-Email` |
| AWS Cognito | ALB | `X-Amzn-Oidc-Data` |
| OCI IAM | nginx + oauth2-proxy | `X-Auth-Request-Groups`, `X-Auth-Request-Email` |
| Okta / Auth0 | oauth2-proxy | `X-Auth-Request-Groups`, `X-Auth-Request-Roles` |


## Audit log

Enable a structured JSON audit trail for every stack execution:

```bash
seristack run \
  --audit-log /var/log/seristack/audit.log \
  --identity-header "user=X-Auth-Request-Email" \
  --identity-header "groups=X-Auth-Request-Groups" \
  --identity-header "roles=X-Auth-Request-Roles"
```

Every execution — success or failure — writes one JSON line:

```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "event": "stack_executed",
  "request_id": "abc-123",
  "stack": "deploy-production",
  "path": "/deploy/production",
  "method": "POST",
  "source_ip": "127.0.0.1:54231",
  "identity": {
    "user": "alice@company.com",
    "groups": "sre,platform"
  },
  "vars": {"env": "production", "version": "v1.2.3"},
  "success": true,
  "duration_ms": 1423
}
```

Query the audit log with `jq`:

```bash
# Who ran deploys today
jq 'select(.stack == "deploy-production")' /var/log/seristack/audit.log

# All failures
jq 'select(.success == false)' /var/log/seristack/audit.log

# Everything a specific user ran
jq 'select(.identity.user == "alice@company.com")' /var/log/seristack/audit.log

# Executions over 30 seconds
jq 'select(.duration_ms > 30000)' /var/log/seristack/audit.log
```

Use `logrotate` to manage audit log rotation. Do not pass secrets as stack vars — they will appear in the audit log. Secrets should come from environment variables or a secrets manager inside the shell script.


## nginx example with IdP integration

```nginx
server {
    listen 443 ssl;
    server_name seristack.example.com;

    ssl_certificate     /etc/letsencrypt/live/seristack.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/seristack.example.com/privkey.pem;

    location /oauth2/ {
        proxy_pass http://127.0.0.1:4180;
        proxy_set_header Host $host;
    }

    location / {
        # Validate identity via oauth2-proxy
        auth_request /oauth2/auth;

        # Forward identity headers from oauth2-proxy to seristack
        auth_request_set $user   $upstream_http_x_auth_request_email;
        auth_request_set $groups $upstream_http_x_auth_request_groups;

        proxy_set_header X-Auth-Request-Email  $user;
        proxy_set_header X-Auth-Request-Groups $groups;
        proxy_set_header X-Request-ID          $request_id;

        proxy_pass http://127.0.0.1:8080;
    }
}
```

## Caddy example with IdP integration

```caddyfile
seristack.example.com {
    forward_auth http://oauth2-proxy:4180 {
        uri /oauth2/auth
        copy_headers X-Auth-Request-Email X-Auth-Request-Groups X-Auth-Request-Roles
    }

    reverse_proxy 127.0.0.1:8080 {
        header_up X-Request-ID {http.request.uuid}
    }
}
```

## MCP server

```bash
seristack mcp -t streamableHTTP --addr 127.0.0.1 --port 8081
```

Stacks with a `description` field are registered as MCP tools. AI agents (Claude, Cursor, Copilot) can call them directly. Apply the same nginx/caddy security posture in front of the MCP server as you would for the HTTP server.


## Variable validation

```yaml
vars:
  - name: env
    value: staging
    required: true
    allowed_value: [staging, production]

  - name: version
    required: true
    allowed_regex: regex("^[a-zA-Z0-9._-]+$")

  - name: command
    denied_regex: regex("(?i)rm|delete|drop")
```

Only variables declared in `vars` can be overridden by HTTP requests or MCP arguments. Undeclared variable names from HTTP inputs are dropped.


## run command reference

```
seristack run [flags]

Flags:
  -c, --config string              config file (default "config.yaml")
  -a, --addr string                bind address (default "127.0.0.1")
  -p, --port string                server port (default "8080")
      --audit-log string           path to audit log file (enables audit logging when set)
      --identity-header strings    map identity header to a key: "user=X-Auth-Request-Email" (repeatable)
```


# Support the project

If Seristack helps your team turn shell scripts or runbooks into internal APIs and MCP tools, consider supporting the project by starring the repository, sharing feedback, opening issues, or contributing examples.


# License

Apache License