# Seristack Config Reference

This document explains the supported attributes in a Seristack YAML configuration file.

Seristack uses a single YAML configuration to define shell workflows that can be:

* Run from the CLI
* Exposed as HTTP endpoints
* Exposed as MCP tools
* Connected through dependencies
* Protected with per-stack authorization
* Audited with structured JSON logs

## Basic configuration

A Seristack configuration contains a root `stacks` list:

```yaml
stacks:
  - name: example
    cmds:
      - echo "hello from seristack"
```

---

## Root attributes

| Attribute | Type | Required | Default | Description                         |
| --------- | ---- | -------- | ------- | ----------------------------------- |
| `stacks`  | list | yes      | none    | List of stack/workflow definitions. |

---

# Stack attributes

| Attribute         | Type            | Required | Default                          | Used by                      | Description                                                                                           |
| ----------------- | --------------- | -------- | -------------------------------- | ---------------------------- | ----------------------------------------------------------------------------------------------------- |
| `name`            | string          | yes      | none                             | CLI, HTTP, MCP, dependencies | Unique stack name.                                                                                    |
| `description`     | string          | no       | empty                            | MCP                          | Human-readable description. A non-empty description allows the stack to be registered as an MCP tool. |
| `method`          | string          | no       | empty                            | HTTP                         | HTTP method such as `GET`, `POST`, `PUT`, `PATCH`, or `DELETE`.                                       |
| `urlPath`         | string          | no       | stack-dependent                  | HTTP                         | HTTP route path.                                                                                      |
| `matchAccess`     | string          | no       | `ANY`                            | HTTP                         | Controls how multiple access rules are combined.                                                      |
| `access`          | list            | no       | empty                            | HTTP                         | Per-stack authorization rules based on HTTP identity headers.                                         |
| `workDir`         | string          | no       | process/config working directory | execution                    | Working directory for command execution.                                                              |
| `continueOnError` | boolean         | no       | `false`                          | execution                    | Determines whether execution continues after a command failure.                                       |
| `dependsOn`       | list of strings | no       | empty                            | dependency resolution        | Names of stacks that must execute before this stack.                                                  |
| `vars`            | list            | no       | empty                            | CLI, HTTP, MCP, templating   | Declared variables and validation rules.                                                              |
| `executionMode`   | string          | no       | implementation default           | execution                    | Controls concurrency strategy.                                                                        |
| `count`           | integer         | no       | `0`                              | execution                    | Number of executions/iterations. `0` means the stack is skipped.                                      |
| `shell`           | string          | no       | mvdan shell                      | execution                    | Optional external shell.                                                                              |
| `shellArg`        | string          | no       | `-c`                             | execution                    | Argument passed to an external shell.                                                                 |
| `cmds`            | list of strings | no       | empty                            | execution                    | Commands/scripts executed by the stack.                                                               |
| `timeouts`        | string          | no       | normalized timeout               | execution                    | Maximum duration for command execution.                                                               |
| `output`          | string          | no       | empty                            | output processing            | Optional command used to post-process accumulated output.                                             |
| `discardOutput`   | list of strings | no       | empty                            | registry                     | Removes selected stack results from the in-memory registry.                                           |

---

# `name`

```yaml
name: deploy-api
```

`name` identifies the stack and must be unique within the configuration.

It is used for:

* `seristack trigger -s <name>`
* HTTP endpoint identification
* MCP tool registration
* Dependency references in `dependsOn`
* Registry/output references

Example:

```yaml
stacks:
  - name: deploy-api
    cmds:
      - ./deploy.sh
```

---

# `description`

```yaml
description: Deploy the application to a target environment
```

A non-empty `description` allows a stack to be registered as an MCP tool.

For example:

```yaml
stacks:
  - name: system-health
    description: Check system health
    cmds:
      - uptime
      - df -h
```

The description should explain what the tool does because MCP clients and AI agents can use it when presenting available tools.

---

# `method` and `urlPath`

Set `method` to expose a stack through the HTTP server.

```yaml
name: deploy
method: POST
urlPath: /deploy
cmds:
  - ./deploy.sh
```

Typical methods include:

```text
GET
POST
PUT
PATCH
DELETE
```

A request can then be sent to the configured endpoint:

```bash
curl -X POST http://127.0.0.1:8080/deploy \
  -H 'Content-Type: application/json' \
  -d '{"env":"staging","version":"v1.2.3"}'
```

`urlPath` allows the HTTP route to differ from the stack name.

---

# `access` and `matchAccess`

The `access` configuration provides per-stack HTTP authorization.

Seristack evaluates identity headers supplied to the HTTP request. In a production deployment, those headers should normally be produced by an authentication layer such as nginx, Caddy, oauth2-proxy, an identity-aware proxy, or another trusted authentication system.

Example:

```yaml
name: deploy-production
method: POST
urlPath: /deploy/production

matchAccess: ANY

access:
  - headerName: "X-Auth-Request-Groups"
    headerValue: ["sre", "platform"]

  - headerName: "X-Auth-Request-Roles"
    headerValue: ["admin"]

cmds:
  - ./deploy-production.sh
```

## `access` attributes

| Attribute     | Type            | Required | Description                      |
| ------------- | --------------- | -------- | -------------------------------- |
| `headerName`  | string          | yes      | HTTP identity header to inspect. |
| `headerValue` | list of strings | yes      | Values accepted for the header.  |

For example:

```yaml
access:
  - headerName: "X-Auth-Request-Groups"
    headerValue:
      - sre
      - platform
```

A header containing multiple comma-separated values can be matched against the configured values.

For example:

```text
X-Auth-Request-Groups: developers,platform,sre
```

can satisfy a rule containing:

```yaml
headerValue: ["sre"]
```

## `matchAccess`

### `ANY`

```yaml
matchAccess: ANY
```

Access is granted when at least one configured access rule matches.

Example:

```yaml
matchAccess: ANY

access:
  - headerName: "X-Auth-Request-Groups"
    headerValue: ["sre"]

  - headerName: "X-Auth-Request-Roles"
    headerValue: ["admin"]
```

This represents:

```text
group == sre OR role == admin
```

### `ALL`

```yaml
matchAccess: ALL
```

Every configured rule must match.

```yaml
matchAccess: ALL

access:
  - headerName: "X-Auth-Request-Groups"
    headerValue: ["devops"]

  - headerName: "X-Auth-Request-Roles"
    headerValue: ["admin"]
```

This represents:

```text
group == devops AND role == admin
```

## No `access` block

If a stack does not define an `access` block, Seristack does not apply stack-specific identity-header authorization to that stack.

For production deployments, put an authentication layer in front of Seristack so that unauthenticated users cannot reach the server directly.

---

# Identity headers

The exact identity headers depend on the authentication infrastructure in front of Seristack.

Common examples include:

| Identity system     | Proxy                | Example identity information                                            |
| ------------------- | -------------------- | ----------------------------------------------------------------------- |
| Entra ID / Azure AD | oauth2-proxy         | `X-Auth-Request-Groups`, `X-Auth-Request-Roles`, `X-Auth-Request-Email` |
| GCP                 | IAP                  | `X-Goog-Authenticated-User-Email`                                       |
| AWS Cognito         | ALB                  | `X-Amzn-Oidc-Data`                                                      |
| OCI                 | nginx + oauth2-proxy | `X-Auth-Request-Groups`, `X-Auth-Request-Email`                         |
| Okta / Auth0        | oauth2-proxy         | Depends on configured claims and forwarded headers                      |

The table is illustrative. Your identity provider and reverse proxy configuration determine which headers are actually available.

---

# `workDir`

```yaml
workDir: ./scripts
```

Sets the working directory used when executing the stack's commands.

Example:

```yaml
stacks:
  - name: build
    workDir: ./backend
    cmds:
      - go build ./...
```

Use this when commands depend on a particular filesystem location.

---

# `continueOnError`

```yaml
continueOnError: true
```

Controls whether execution continues after a command fails.

### `false`

```yaml
continueOnError: false
```

The default behavior. A command failure stops the remaining execution for the stack according to the execution flow.

### `true`

```yaml
continueOnError: true
```

The failure is recorded and execution continues.

Example:

```yaml
name: diagnostics
continueOnError: true

cmds:
  - echo "Starting diagnostics"
  - ./check-service-a.sh
  - ./check-service-b.sh
  - echo "Diagnostics complete"
```

---

# `dependsOn`

Defines dependencies between stacks.

```yaml
dependsOn:
  - build
  - test
```

Seristack resolves stack dependencies before execution.

Example:

```yaml
stacks:

  - name: build
    cmds:
      - go build ./...

  - name: test
    dependsOn:
      - build
    cmds:
      - go test ./...

  - name: deploy
    dependsOn:
      - test
    cmds:
      - ./deploy.sh
```

The execution relationship is:

```text
build
  ↓
test
  ↓
deploy
```

Multiple dependencies can be specified:

```yaml
name: deploy
dependsOn:
  - build
  - test
  - security-scan
```

---

# `vars`

Variables are declared as a list.

```yaml
vars:
  - name: env
    value: staging
```

Variables can be referenced from commands:

```yaml
cmds:
  - echo "Environment: {{.Vars.env}}"
```

Only variables declared in the stack configuration are eligible for runtime overrides.

Undeclared external variable names are not accepted as stack variables.

## Variable attributes

| Attribute       | Type            | Required | Default | Description                                     |
| --------------- | --------------- | -------- | ------- | ----------------------------------------------- |
| `name`          | string          | yes      | none    | Variable name.                                  |
| `value`         | string          | no       | empty   | Default value.                                  |
| `required`      | boolean         | no       | false   | Requires the final value to be non-empty.       |
| `allowed_value` | list of strings | no       | empty   | Restricts the variable to listed values.        |
| `denied_value`  | list of strings | no       | empty   | Rejects listed values.                          |
| `allowed_regex` | string          | no       | empty   | Restricts the value using a regular expression. |
| `denied_regex`  | string          | no       | empty   | Rejects values matching a regular expression.   |

## Runtime variable sources

Variables may be supplied by supported runtime interfaces such as:

| Source             | Example                                    |
| ------------------ | ------------------------------------------ |
| CLI                | `--vars key=value`                         |
| HTTP query         | `?env=staging`                             |
| HTTP form          | `env=staging`                              |
| HTTP JSON          | `{"env":"staging"}`                        |
| HTTP `X-*` headers | `X-Env: staging`                           |
| MCP                | Tool argument matching a declared variable |

## Allowed values

```yaml
vars:
  - name: env
    value: staging
    required: true
    allowed_value:
      - development
      - staging
      - production
```

Only the listed values are accepted.

## Denied values

```yaml
vars:
  - name: environment
    denied_value:
      - production
```

## Allowed regular expression

```yaml
vars:
  - name: version
    required: true
    allowed_regex: regex("^[a-zA-Z0-9._-]+$")
```

## Denied regular expression

```yaml
vars:
  - name: command
    denied_regex: regex("(?i)rm|delete|drop")
```

## Variable validation

Use the validation rules carefully.

Do not pass secrets as stack variables.

Stack variables may appear in execution/audit output depending on how the stack is executed. Secrets should instead come from environment variables, a secrets manager, or another protected runtime mechanism.

---

# `executionMode`

Controls how stack iterations and commands are scheduled.

```yaml
executionMode: SEQUENTIAL
```

Supported modes:

| Mode         | Count iterations | Commands inside each iteration |
| ------------ | ---------------- | ------------------------------ |
| `PARALLEL`   | concurrent       | concurrent                     |
| `STAGE`      | concurrent       | sequential                     |
| `PIPELINE`   | sequential       | concurrent                     |
| `SEQUENTIAL` | sequential       | sequential                     |

## `PARALLEL`

Runs iterations concurrently and allows commands within an iteration to execute concurrently.

```yaml
executionMode: PARALLEL
count: 3
```

## `STAGE`

Runs iterations concurrently while keeping commands inside each iteration sequential.

```yaml
executionMode: STAGE
count: 3
```

## `PIPELINE`

Runs iterations sequentially while allowing commands inside each iteration to execute concurrently.

```yaml
executionMode: PIPELINE
count: 3
```

## `SEQUENTIAL`

Runs everything sequentially.

```yaml
executionMode: SEQUENTIAL
count: 3
```

Use this when command ordering is important or when commands modify shared state.

---

# `count`

Controls how many times a stack is executed.

```yaml
count: 3
```

Examples:

```yaml
count: 1
```

Runs once.

```yaml
count: 3
```

Runs three iterations.

```yaml
count: 0
```

The stack is skipped.

This distinction is important when writing tests or simple configurations. A stack intended to execute at least once should explicitly use:

```yaml
count: 1
```

The current iteration can be referenced from commands using:

```text
{{.Count.index}}
```

Example:

```yaml
count: 3

cmds:
  - echo "Iteration {{.Count.index}}"
```

---

# `timeouts`

Controls the maximum execution duration for commands.

```yaml
timeouts: 30s
```

Seristack uses Go duration syntax.

Supported units include:

| Unit | Meaning      |
| ---- | ------------ |
| `ns` | nanoseconds  |
| `us` | microseconds |
| `µs` | microseconds |
| `ms` | milliseconds |
| `s`  | seconds      |
| `m`  | minutes      |
| `h`  | hours        |

Examples:

```yaml
timeouts: 500ms
```

```yaml
timeouts: 30s
```

```yaml
timeouts: 5m
```

```yaml
timeouts: 1h
```

```yaml
timeouts: 1h30m
```

```yaml
timeouts: 24h
```

Use `24h` instead of `1d`.

---

# `shell` and `shellArg`

By default, Seristack uses the built-in mvdan shell interpreter.

An external shell can be selected with:

```yaml
shell: bash
shellArg: -c
```

Examples include:

```yaml
shell: bash
shellArg: -c
```

```yaml
shell: sh
shellArg: -c
```

On Windows:

```yaml
shell: powershell
shellArg: -Command
```

The exact shell executable and arguments depend on the operating system and installed shell.

---

# `cmds`

`cmds` contains the commands executed by a stack.

A command can be a single line:

```yaml
cmds:
  - echo "hello"
```

Or a multi-line shell script:

```yaml
cmds:
  - |
    echo "starting"
    echo "checking service"
    systemctl status nginx
    echo "finished"
```

Variables can be substituted:

```yaml
cmds:
  - echo "Deploying {{.Vars.version}}"
```

The current iteration can be accessed with:

```text
{{.Count.index}}
```

Output from previous execution steps can be referenced where supported through:

```text
{{.Self.result}}
```

---

# `output`

`output` defines optional post-processing for accumulated stack output.

Example:

```yaml
output: |
  echo '{{.Self.result}}' | jq -s '.'
```

A common use case is aggregating JSON emitted by multiple command executions:

```yaml
cmds:
  - |
    echo '{"status":"ok","service":"api"}'

  - |
    echo '{"status":"ok","service":"worker"}'

output: |
  echo '{{.Self.result}}' | jq -s '{
    total: length,
    results: .
  }'
```

---

# `discardOutput`

Removes selected stack outputs from the in-memory registry.

```yaml
discardOutput:
  - build
  - test
```

This is useful when downstream stacks no longer need earlier results and you want to reduce retained registry data.

Example:

```yaml
stacks:

  - name: build
    cmds:
      - go build ./...

  - name: test
    dependsOn: [build]
    cmds:
      - go test ./...

  - name: cleanup
    dependsOn: [test]
    discardOutput:
      - build
      - test
    cmds:
      - echo "cleanup complete"
```

---

# HTTP configuration example

A stack can combine variables, HTTP exposure, and authorization:

```yaml
stacks:
  - name: deploy
    description: Deploy the application
    method: POST
    urlPath: /deploy

    matchAccess: ALL

    access:
      - headerName: "X-Auth-Request-Groups"
        headerValue:
          - sre
          - platform

      - headerName: "X-Auth-Request-Roles"
        headerValue:
          - admin

    count: 1

    timeouts: 10m

    executionMode: SEQUENTIAL

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

    cmds:
      - |
        echo "Deploying {{.Vars.version}} to {{.Vars.env}}"
        ./deploy.sh
```

---

# MCP configuration

A stack with a non-empty `description` can be registered as an MCP tool.

Example:

```yaml
stacks:
  - name: system-health
    description: Check system health
    count: 1
    cmds:
      - |
        echo "system-health is good"
```

Run the MCP server:

```bash
seristack mcp \
  -t streamableHTTP \
  --addr 127.0.0.1 \
  --port 8081
```

For a remotely accessible MCP deployment, place an authentication and TLS layer in front of Seristack.

Example architecture:

```text
AI agent / IDE
      |
      v
 HTTPS
      |
      v
nginx / Caddy / authentication proxy
      |
      v
Seristack MCP
      |
      v
Stack authorization + execution
```

For production deployments, do not expose the Seristack MCP process directly to the public internet.

---

# Audit logging

Seristack can write structured JSON audit records.

Enable audit logging with:

```bash
seristack run \
  --config config.yaml \
  --addr 127.0.0.1 \
  --port 8080 \
  --audit-log /var/log/seristack/audit.log
```

Audit records contain execution information such as:

```json
{
  "timestamp": "2026-09-19T10:30:00Z",
  "event": "stack_executed",
  "stack": "deploy",
  "path": "/deploy",
  "method": "POST",
  "success": true,
  "duration_ms": 1423
}
```

Depending on the request and configuration, additional identity, variable, output, or error information may be present.

Use `jq` to inspect records:

```bash
jq 'select(.success == false)' /var/log/seristack/audit.log
```

Find slow executions:

```bash
jq 'select(.duration_ms > 30000)' /var/log/seristack/audit.log
```

Use log rotation for long-running production deployments.

Do not place secrets in stack variables.

---

# Complete example

```yaml
stacks:

  - name: build
    description: Build the application
    count: 1
    executionMode: SEQUENTIAL
    timeouts: 10m

    cmds:
      - |
        echo "Building application"
        go build ./...

  - name: test
    description: Run application tests
    dependsOn:
      - build
    count: 1
    executionMode: SEQUENTIAL
    timeouts: 10m

    cmds:
      - |
        echo "Running tests"
        go test ./...

  - name: deploy
    description: Deploy the application
    method: POST
    urlPath: /deploy

    dependsOn:
      - test

    matchAccess: ALL

    access:
      - headerName: "X-Auth-Request-Groups"
        headerValue:
          - sre
          - platform

      - headerName: "X-Auth-Request-Roles"
        headerValue:
          - admin

    count: 1
    executionMode: SEQUENTIAL
    timeouts: 10m

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

    cmds:
      - |
        echo "Deploying {{.Vars.version}} to {{.Vars.env}}"
        ./deploy.sh

  - name: cleanup
    dependsOn:
      - deploy

    count: 1
    timeouts: 30s

    discardOutput:
      - build
      - test

    cmds:
      - echo "Cleanup complete"
```

---

# Production recommendations

Seristack executes shell commands and should be treated as a privileged automation service.

Recommended architecture:

```text
Internet
   |
   v
TLS / Authentication / Rate limiting
   |
   v
nginx / Caddy / oauth2-proxy
   |
   v
Seristack on 127.0.0.1
   |
   +---- Stack authorization
   |
   +---- Variable validation
   |
   +---- Shell execution
   |
   +---- Audit logging
```

For HTTP deployments:

```bash
seristack run \
  --config config.yaml \
  --addr 127.0.0.1 \
  --port 8080 \
  --audit-log /var/log/seristack/audit.log
```

For MCP deployments:

```bash
seristack mcp \
  -t streamableHTTP \
  --addr 127.0.0.1 \
  --port 8081
```

Keep Seristack bound to a trusted interface and use a reverse proxy or network security layer for external access.
