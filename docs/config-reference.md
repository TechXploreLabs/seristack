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
| `name` | string | yes | none | CLI, HTTP, MCP, dependencies | Unique stack name. Used for stack selection, default HTTP route, MCP tool name, dependencies, and registry output keys. |
| `description` | string | no | empty | MCP | Human-readable description. Stacks with a non-empty description are registered as MCP tools. |
| `method` | string | no | empty | HTTP | HTTP method for endpoint exposure, such as `GET`, `POST`, `PUT`, `PATCH`, or `DELETE`. If empty, the stack is not exposed as an HTTP endpoint. |
| `urlPath` | string | no | `/<name>` | HTTP | Custom HTTP route path. If omitted, the stack name is used as the path. |
| `workDir` | string | no | source/config directory behavior | shell execution | Working directory for command execution, resolved relative to the Seristack source/config execution directory. |
| `continueOnError` | boolean | no | `false` | execution | If `true`, Seristack records command errors and continues. If `false`, command failures stop execution. |
| `dependsOn` | list of strings | no | `[]` | execution order | Stack names that must run before this stack. |
| `vars` | list | no | empty | CLI, HTTP, MCP, templating | Variable definitions and validation rules. Runtime values can override declared variables. |
| `executionMode` | string | no | `PARALLEL` | execution | Controls concurrency between count iterations and command execution. Valid values: `PARALLEL`, `STAGE`, `PIPELINE`, `SEQUENTIAL`. |
| `count` | integer | no | `0` | execution | Number of times to run the stack commands. `0` skips command execution. |
| `timeouts` | string | no | `1h` | shell execution | Per-command timeout using Go duration syntax, such as `30s`, `5m`, or `1h30m`. |
| `shell` | string | no | built-in mvdan shell | shell execution | External shell executable, such as `bash`, `sh`, `pwsh`, or `powershell`. |
| `shellArg` | string | no | `-c` for external shells | shell execution | Argument passed to the external shell before the command script. |
| `cmds` | list of strings | no | empty | shell execution | Commands/scripts executed by the stack. Usually required for useful stacks. |
| `output` | string | no | empty | output aggregation | Optional post-processing command that can use `{{.Self.result}}` to aggregate command output. |
| `discardOutput` | list of strings | no | empty | registry cleanup | Stack output keys to remove from the in-memory registry after this stack completes. |

## `name`

```yaml
name: deploy-api
```

`name` must be unique. It is used for:

- `seristack trigger -s <name>`
- default HTTP path when `urlPath` is not set
- MCP tool name
- dependency references through `dependsOn`
- output registry keys

## `description`

```yaml
description: Restart the application service
```

In MCP mode, stacks with a non-empty `description` are registered as MCP tools. If the description is empty, the stack is not added as an MCP tool.

## `method` and `urlPath`

```yaml
method: POST
urlPath: /deploy
```

Set `method` to expose a stack as an HTTP endpoint. `urlPath` is optional. If `urlPath` is omitted, Seristack uses `/<stack-name>`.

Example:

```yaml
name: greet-api
method: GET
urlPath: /greet
```

Request example:

```bash
curl 'http://127.0.0.1:8080/greet?name=alice&env=dev'
```

## `workDir`

```yaml
workDir: ./scripts
```

Sets the working directory for shell command execution.

## `continueOnError`

```yaml
continueOnError: true
```

- `false`: stop execution on command failure
- `true`: continue execution and record the error in the result

## `dependsOn`

```yaml
dependsOn: [build, test]
```

Runs the current stack after the listed stacks complete.

```yaml
stacks:
  - name: build
    cmds:
      - go build ./...

  - name: test
    dependsOn: [build]
    cmds:
      - go test ./...
```

## `vars`

Variables are declared as a list of objects.

```yaml
vars:
  - name: env
    value: dev
```

Use variables in commands with:

```text
{{.Vars.env}}
```

Runtime values can come from:

- CLI `--vars`
- CLI `--vars-json`
- HTTP query parameters
- HTTP form values
- HTTP JSON body
- HTTP headers beginning with `X-`
- MCP tool arguments

Only variables already declared in `vars` are overridden by runtime values.

## Variable attributes

| Attribute | Type | Required | Default | Description |
|---|---:|---:|---|---|
| `name` | string | yes | none | Variable name. Must be unique within the stack. |
| `value` | string | no | empty | Default variable value. |
| `required` | boolean | no | `false` | If `true`, the final value must not be empty. |
| `allowed_value` | list of strings | no | empty | Allows only values from the list. |
| `denied_value` | list of strings | no | empty | Rejects values from the list. |
| `allowed_regex` | string | no | empty | Allows only values matching `regex(...)`. |
| `denied_regex` | string | no | empty | Rejects values matching `regex(...)`. |

### Variable validation examples

```yaml
vars:
  - name: env
    value: dev
    required: true
    allowed_value: [dev, stage, prod]
```

```yaml
vars:
  - name: service
    required: true
    allowed_regex: regex("^[a-zA-Z0-9_-]+$")
```

```yaml
vars:
  - name: command
    denied_regex: regex("(?i)rm|delete|drop")
```

Only one of these rule sets can be used for a single variable:

- `allowed_value`
- `denied_value`
- `allowed_regex`
- `denied_regex`

`required` can be combined with one of those rule sets.

## `executionMode`

```yaml
executionMode: SEQUENTIAL
```

Valid values:

| Value | Count iterations | Commands inside each iteration |
|---|---|---|
| `PARALLEL` | concurrent | concurrent |
| `STAGE` | concurrent | sequential |
| `PIPELINE` | sequential | concurrent |
| `SEQUENTIAL` | sequential | sequential |

Use `SEQUENTIAL` when ordering matters. Use `PARALLEL` when maximum concurrency is desired.

## `count`

```yaml
count: 3
```

Number of times to run the stack commands.

- `count: 0` skips command execution
- `count: 1` runs once
- `count: 3` runs three times

Inside commands, use the current iteration index with:

```text
{{.Count.index}}
```

## `timeouts`

```yaml
timeouts: 30s
```

Controls the maximum duration for each command execution in the stack. Default: `1h`.

Seristack uses Go duration syntax through `time.ParseDuration`.

Supported units:

| Unit | Meaning |
|---|---|
| `ns` | nanoseconds |
| `us` or `µs` | microseconds |
| `ms` | milliseconds |
| `s` | seconds |
| `m` | minutes |
| `h` | hours |

Examples:

```yaml
timeouts: 500ms
timeouts: 30s
timeouts: 5m
timeouts: 1h
timeouts: 1h30m
timeouts: 2.5h
timeouts: 24h
```

Invalid examples:

```yaml
timeouts: 0s
timeouts: -1m
timeouts: 1d
timeouts: never
```

Use `24h` instead of `1d`.

## `shell` and `shellArg`

```yaml
shell: bash
shellArg: -c
```

If `shell` is omitted, Seristack uses the built-in mvdan shell interpreter. If an external shell is configured, `shellArg` defaults to `-c` when omitted.

Examples:

```yaml
shell: bash
shellArg: -c
```

```yaml
shell: powershell
shellArg: /C
```

## `cmds`

```yaml
cmds:
  - echo "hello"
  - echo "world"
```

Commands can be single-line or multi-line scripts.

```yaml
cmds:
  - |
    echo "starting"
    echo "finished"
```

## `output`

```yaml
output: |
  echo '{{.Self.result}}' | jq -s '.'
```

Optional post-processing command. The accumulated command output is available through:

```text
{{.Self.result}}
```

## `discardOutput`

```yaml
discardOutput: [build, test]
```

Removes saved outputs from the in-memory registry after the current stack completes.

Use this to reduce memory usage or prevent later stacks from reading outputs that are no longer needed.

## HTTP endpoint behavior

A stack becomes an HTTP endpoint when `method` is set.

HTTP variables can come from:

- query parameters
- form values
- JSON request body
- headers beginning with `X-`

## MCP tool behavior

A stack becomes an MCP tool when `description` is set. Variables declared in `vars` become MCP tool arguments.

## Complete example

```yaml
stacks:
  - name: greet-api
    description: Greets a user from CLI, HTTP, or MCP
    method: GET
    urlPath: /greet
    workDir: ./
    count: 1
    timeouts: 30s
    executionMode: SEQUENTIAL
    vars:
      - name: name
        value: engineer
        required: true
        allowed_regex: regex("^[a-zA-Z0-9_-]+$")
      - name: env
        value: dev
        allowed_value: [dev, stage, prod]
    cmds:
      - |
        echo "Hello {{.Vars.name}} from {{.Vars.env}}"

  - name: collect-metadata
    description: Produces JSON metadata for aggregation
    dependsOn: [greet-api]
    count: 2
    timeouts: 5m
    executionMode: SEQUENTIAL
    vars:
      - name: env
        value: dev
    cmds:
      - |
        echo "{\"index\": {{.Count.index}}, \"env\": \"{{.Vars.env}}\", \"status\": \"ok\"}"
    output: |
      echo '{{.Self.result}}' | grep "^{" | jq -s '{records: ., total: length}'

  - name: cleanup-output
    dependsOn: [collect-metadata]
    discardOutput: [greet-api]
    count: 1
    timeouts: 10s
    cmds:
      - echo "discarded greet-api output from memory"
```