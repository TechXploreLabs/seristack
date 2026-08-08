# Seristack Config Reference

This document explains every supported attribute in a Seristack YAML configuration file.

Seristack configs use a root `stacks` list. Each stack describes one shell workflow that can be run from the CLI, exposed as an HTTP endpoint, or exposed as an MCP tool.

```yaml
stacks:
  - name: example
    cmds:
      - echo "hello from seristack"
```

## Root attributes

| Attribute | Type | Required | Default | Description |
|---|---:|---:|---|---|
| `stacks` | list | yes | none | List of stack/workflow definitions. |

## Stack attributes

| Attribute | Type | Required | Default | Used by | Description |
|---|---:|---:|---|---|---|
| `name` | string | yes | none | CLI, HTTP, MCP, dependencies | Unique stack name. |
| `description` | string | no | empty | MCP | Stacks with a non-empty description are registered as MCP tools. |
| `method` | string | no | empty | HTTP | HTTP method (`GET`, `POST`, `PUT`, `PATCH`, `DELETE`). If empty, the stack is not exposed as an HTTP endpoint. |
| `urlPath` | string | no | `/<name>` | HTTP | Custom HTTP route path. |
| `matchAccess` | string | no | `ANY` | HTTP | How multiple `access` rules are evaluated. `ANY` grants access if one rule matches. `ALL` requires every rule to match. |
| `access` | list | no | empty | HTTP | Per-stack authorization rules. Each rule checks an HTTP header against allowed values. |
| `workDir` | string | no | config directory | shell execution | Working directory for command execution. |
| `continueOnError` | boolean | no | `false` | execution | If `true`, records errors and continues. If `false`, stops on failure. |
| `dependsOn` | list of strings | no | `[]` | execution order | Stack names that must complete before this stack runs. |
| `vars` | list | no | empty | CLI, HTTP, MCP, templating | Variable definitions and validation rules. |
| `executionMode` | string | no | `PARALLEL` | execution | Controls concurrency. Valid values: `PARALLEL`, `STAGE`, `PIPELINE`, `SEQUENTIAL`. |
| `count` | integer | no | `0` | execution | Number of times to run the stack. `0` skips execution. |
| `timeouts` | string | no | `1h` | shell execution | Per-command timeout. Uses Go duration syntax: `30s`, `5m`, `1h30m`. |
| `shell` | string | no | mvdan shell | shell execution | External shell: `bash`, `sh`, `pwsh`, `powershell`. |
| `shellArg` | string | no | `-c` | shell execution | Argument passed to the external shell before the command script. |
| `cmds` | list of strings | no | empty | shell execution | Commands executed by the stack. |
| `output` | string | no | empty | output aggregation | Post-processing command. Can use `{{.Self.result}}` to aggregate output. |
| `discardOutput` | list of strings | no | empty | registry cleanup | Stack output keys to remove from memory after this stack completes. |

---

## `name`

```yaml
name: deploy-api
```

`name` must be unique across all stacks. It is used for:

- `seristack trigger -s <name>`
- Default HTTP path when `urlPath` is not set
- MCP tool name
- Dependency references in `dependsOn`
- Output registry keys shared between stacks

---

## `description`

```yaml
description: Deploy the application to a target environment
```

In MCP mode, stacks with a non-empty `description` are registered as MCP tools. If the description is empty, the stack is not added.

---

## `method` and `urlPath`

```yaml
method: POST
urlPath: /deploy
```

Set `method` to expose a stack as an HTTP endpoint. If `urlPath` is omitted, Seristack uses `/<stack-name>`.

```bash
curl -X POST 'http://127.0.0.1:8080/deploy' \
  -H 'Content-Type: application/json' \
  -d '{"env": "staging", "version": "v1.2.3"}'
```

---

## `access` and `matchAccess`

Per-stack authorization. Seristack checks HTTP headers forwarded by nginx or Caddy after they validate the user against your IdP.

```yaml
matchAccess: ANY
access:
  - headerName: "X-Auth-Request-Groups"
    headerValue: ["sre", "platform"]
  - headerName: "X-Auth-Request-Roles"
    headerValue: ["admin"]
```

### `access` attributes

| Attribute | Type | Required | Description |
|---|---:|---:|---|
| `headerName` | string | yes | The exact HTTP header name to read. |
| `headerValue` | list of strings | yes | Allowed values. Access is granted if the header contains any of these values. |

### `matchAccess` values

| Value | Behaviour |
|---|---|
| `ANY` | Access granted if **any one** rule matches. Default. |
| `ALL` | Access granted only if **every** rule matches. |

### How it works

Your nginx or Caddy config forwards identity headers after validating the user's JWT or token:

```nginx
auth_request_set $groups $upstream_http_x_auth_request_groups;
proxy_set_header X-Auth-Request-Groups $groups;
```

Seristack reads those headers and compares them to the `access` block. Header values can be comma-separated (e.g. `sre,platform,devops`) and Seristack splits them automatically.

### No `access` block

If a stack has no `access` block, any authenticated user reaching seristack can execute it. Use nginx or Caddy authentication to ensure only authenticated users can reach seristack at all.

### Examples

**Any member of sre or platform can run:**
```yaml
access:
  - headerName: "X-Auth-Request-Groups"
    headerValue: ["sre", "platform"]
```

**Must be in devops group AND have admin role:**
```yaml
matchAccess: ALL
access:
  - headerName: "X-Auth-Request-Groups"
    headerValue: ["devops"]
  - headerName: "X-Auth-Request-Roles"
    headerValue: ["admin"]
```

**Specific user only:**
```yaml
access:
  - headerName: "X-Auth-Request-Email"
    headerValue: ["oncall@company.com"]
```

### Identity header names by IdP

| IdP | Proxy | Group header | Role header | Email header |
|---|---|---|---|---|
| Entra ID / Azure AD | oauth2-proxy | `X-Auth-Request-Groups` | `X-Auth-Request-Roles` | `X-Auth-Request-Email` |
| GCP IAM | IAP | — | — | `X-Goog-Authenticated-User-Email` |
| AWS Cognito | ALB | `X-Amzn-Oidc-Data` | — | — |
| OCI IAM | nginx + oauth2-proxy | `X-Auth-Request-Groups` | — | `X-Auth-Request-Email` |
| Okta / Auth0 | oauth2-proxy | `X-Auth-Request-Groups` | `X-Auth-Request-Roles` | `X-Auth-Request-Email` |

---

## `workDir`

```yaml
workDir: ./scripts
```

Sets the working directory for shell command execution, resolved relative to the Seristack process working directory.

---

## `continueOnError`

```yaml
continueOnError: true
```

- `false` — stop execution on command failure (default)
- `true` — record the error and continue

---

## `dependsOn`

```yaml
dependsOn: [build, test]
```

Runs the current stack after the listed stacks complete. Seristack resolves dependencies topologically.

```yaml
stacks:
  - name: build
    cmds:
      - go build ./...

  - name: test
    dependsOn: [build]
    cmds:
      - go test ./...

  - name: deploy
    dependsOn: [test]
    cmds:
      - ./deploy.sh
```

---

## `vars`

Variables are declared as a list. Declared variables can be overridden at runtime from HTTP, CLI, or MCP. Undeclared variable names from external inputs are dropped.

```yaml
vars:
  - name: env
    value: dev
```

Use variables in commands:

```text
{{.Vars.env}}
```

### Runtime variable sources

| Source | How |
|---|---|
| CLI | `--vars key=value` or `--vars-json '{"key":"value"}'` |
| HTTP query params | `?env=staging` |
| HTTP form body | `application/x-www-form-urlencoded` |
| HTTP JSON body | `application/json` with `{"env": "staging"}` |
| HTTP headers | Any header starting with `X-` |
| MCP | Tool arguments matching declared variable names |

### Variable attributes

| Attribute | Type | Required | Default | Description |
|---|---:|---:|---|---|
| `name` | string | yes | none | Variable name. Must be unique within the stack. |
| `value` | string | no | empty | Default value. |
| `required` | boolean | no | `false` | If `true`, the final value must not be empty. |
| `allowed_value` | list of strings | no | empty | Allows only values from this list. |
| `denied_value` | list of strings | no | empty | Rejects values in this list. |
| `allowed_regex` | string | no | empty | Allows only values matching the pattern. |
| `denied_regex` | string | no | empty | Rejects values matching the pattern. |

Use only one rule per variable: `allowed_value`, `denied_value`, `allowed_regex`, or `denied_regex`. `required` can be combined with any of them.

### Variable validation examples

```yaml
vars:
  - name: env
    value: staging
    required: true
    allowed_value: [dev, staging, production]
```

```yaml
vars:
  - name: version
    required: true
    allowed_regex: regex("^[a-zA-Z0-9._-]+$")
```

```yaml
vars:
  - name: command
    denied_regex: regex("(?i)rm|delete|drop")
```

> **Note:** Do not pass secrets as stack vars. They will appear in the audit log and in logs. Secrets should come from environment variables or a secrets manager inside the shell script.

---

## `executionMode`

```yaml
executionMode: SEQUENTIAL
```

| Value | Count iterations | Commands inside each iteration |
|---|---|---|
| `PARALLEL` | concurrent | concurrent |
| `STAGE` | concurrent | sequential |
| `PIPELINE` | sequential | concurrent |
| `SEQUENTIAL` | sequential | sequential |

Default is `PARALLEL`.

---

## `count`

```yaml
count: 3
```

Number of times to run the stack commands.

- `count: 0` — skip execution
- `count: 1` — run once
- `count: 3` — run three times

Use the current iteration index in commands:

```text
{{.Count.index}}
```

---

## `timeouts`

```yaml
timeouts: 30s
```

Maximum duration for each command execution in the stack. Default: `1h`.

Uses Go duration syntax:

| Unit | Meaning |
|---|---|
| `ns` | nanoseconds |
| `us` or `µs` | microseconds |
| `ms` | milliseconds |
| `s` | seconds |
| `m` | minutes |
| `h` | hours |

```yaml
timeouts: 500ms
timeouts: 30s
timeouts: 5m
timeouts: 1h
timeouts: 1h30m
timeouts: 2.5h
timeouts: 24h
```

Invalid values: `0s`, `-1m`, `1d`, `never`. Use `24h` instead of `1d`.

---

## `shell` and `shellArg`

```yaml
shell: bash
shellArg: -c
```

If `shell` is omitted, Seristack uses the built-in mvdan shell interpreter. `shellArg` defaults to `-c` when using an external shell.

```yaml
shell: powershell
shellArg: /C
```

---

## `cmds`

```yaml
cmds:
  - echo "hello"
  - |
    echo "starting"
    echo "finished"
```

Commands run in the stack's `workDir`. Use `{{.Vars.key}}` for variable substitution and `{{.Self.result}}` to access output from previous commands in the same stack.

---

## `output`

```yaml
output: |
  echo '{{.Self.result}}' | jq -s '.'
```

Optional post-processing command. Runs after all `cmds` complete. The accumulated output from `cmds` is available through `{{.Self.result}}`.

---

## `discardOutput`

```yaml
discardOutput: [build, test]
```

Removes the named stack outputs from the in-memory registry after the current stack completes. Use this to free memory when downstream stacks no longer need earlier outputs.

---

## Complete example

```yaml
stacks:
  - name: deploy
    description: Deploy the application to a target environment
    method: POST
    urlPath: /deploy
    matchAccess: ANY
    access:
      - headerName: "X-Auth-Request-Groups"
        headerValue: ["sre", "platform"]
      - headerName: "X-Auth-Request-Roles"
        headerValue: ["admin"]
    count: 1
    timeouts: 10m
    executionMode: SEQUENTIAL
    vars:
      - name: env
        value: staging
        required: true
        allowed_value: [staging, production]
      - name: version
        required: true
        allowed_regex: regex("^[a-zA-Z0-9._-]+$")
    cmds:
      - |
        echo "Deploying {{.Vars.version}} to {{.Vars.env}}"
        kubectl set image deployment/app app={{.Vars.version}}

  - name: smoke-test
    description: Run smoke tests against a deployed environment
    method: POST
    urlPath: /smoke-test
    dependsOn: [deploy]
    access:
      - headerName: "X-Auth-Request-Groups"
        headerValue: ["sre", "platform", "qa"]
    count: 1
    timeouts: 5m
    vars:
      - name: env
        value: staging
        allowed_value: [staging, production]
    cmds:
      - |
        echo "Running smoke tests against {{.Vars.env}}"
        curl -sf https://app-{{.Vars.env}}.internal/health

  - name: notify
    description: Send Slack notification
    dependsOn: [smoke-test]
    count: 1
    timeouts: 10s
    vars:
      - name: message
        value: "Deployment complete"
    cmds:
      - |
        curl -s -X POST "$SLACK_WEBHOOK" \
          -H 'Content-Type: application/json' \
          -d "{\"text\": \"{{.Vars.message}}\"}"
```

Start the server with audit logging:

```bash
seristack run \
  --config config.yaml \
  --addr 127.0.0.1 \
  --port 8080 \
  --audit-log /var/log/seristack/audit.log \
  --identity-header "user=X-Auth-Request-Email" \
  --identity-header "groups=X-Auth-Request-Groups"
```