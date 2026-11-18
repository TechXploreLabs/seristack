# Seristack Configuration Reference

Every field in a Seristack YAML file. For behavior, deployment, and security guidance, see the [Guide](guide.md).

A configuration is a root `stacks` list:

```yaml
stacks:
  - name: example
    count: 1
    cmds:
      - echo "hello from seristack"
```

> **A stack with `count: 0` is skipped.** Set `count: 1` for a stack that should run once.

---

## Stack fields

| Field | Type | Default | Description |
|---|---|---|---|
| `name` | string | required | Unique stack name. |
| `description` | string | empty | Human-readable description. A non-empty value exposes the stack as an MCP tool. |
| `method` | string | empty | HTTP method. A non-empty value exposes the stack as an HTTP endpoint. |
| `urlPath` | string | `/<name>` | HTTP route path. |
| `matchAccess` | string | `ANY` | How `access` rules combine: `ANY` or `ALL`. |
| `access` | list | empty | Identity-header authorization rules. |
| `workDir` | string | config directory | Working directory, relative to where Seristack was started. |
| `continueOnError` | boolean | `false` | Keep going after a command fails. |
| `dependsOn` | list of strings | empty | Stacks that must run first. |
| `vars` | list | empty | Variables and validation rules. |
| `executionMode` | string | `PARALLEL` | `SEQUENTIAL`, `PARALLEL`, `STAGE`, or `PIPELINE`. |
| `count` | integer | `0` | Number of repeats. `0` skips the stack. |
| `executor` | list of strings | built-in mvdan shell | External program used to run commands, for example `["bash", "-c"]`. |
| `cmds` | list of strings | empty | Commands or scripts to run. |
| `timeouts` | string | `1h` | Maximum duration of **each command**. Go duration syntax. |
| `output` | string | empty | Command that post-processes the stack's collected output. |
| `discardOutput` | list of strings | empty | Stack names whose stored output is released once this stack finishes. |

---

## `name`

```yaml
name: deploy-api
```

Must be unique. Used by `seristack trigger -s <name>`, `dependsOn`, HTTP routing, and MCP tool registration.

Names are also used in `{{.Result.<name>}}` templates, and Go templates reject `-` in that position. If other stacks read this stack's output with `.Result`, use letters, digits, and underscores (`deploy_api`).

---

## `description`

```yaml
description: Check system health
```

A non-empty `description` registers the stack as an MCP tool. Write it for the AI client that will read it: say what the tool does and when to use it.

---

## `method` and `urlPath`

```yaml
method: POST
urlPath: /deploy
```

`method` exposes the stack as an HTTP endpoint (`GET`, `POST`, `PUT`, `PATCH`, `DELETE`). If `urlPath` is empty, the route is `/<name>`. Requests with a different method get `405`.

```bash
curl -X POST http://127.0.0.1:8080/deploy \
  -H 'Content-Type: application/json' \
  -d '{"env":"staging","version":"v1.2.3"}'
```

---

## `access` and `matchAccess`

Restrict a stack using identity headers set by a trusted reverse proxy.

```yaml
matchAccess: ANY
access:
  - headerName: X-Auth-Request-Groups
    headerValue: [sre, platform]
  - headerName: X-Auth-Request-Roles
    headerValue: [admin]
```

| Field | Type | Description |
|---|---|---|
| `headerName` | string | Header to inspect. |
| `headerValue` | list of strings | Accepted values. |

A header with comma-separated values (`X-Auth-Request-Groups: dev,platform,sre`) matches if any listed value is present.

* `ANY`: access is granted when at least one rule matches (OR).
* `ALL`: access is granted only when every rule matches (AND).

A stack with **no `access` block has no stack-level restriction**. Seristack trusts the headers it receives, so the proxy in front must remove client-supplied copies and set its own. See [Per-stack authorization](guide.md#per-stack-authorization).

---

## `workDir`

```yaml
workDir: ./backend
```

Working directory for the stack's commands, joined to the directory Seristack was started from.

---

## `continueOnError`

```yaml
continueOnError: true
```

* `false` (default): a failed command stops the rest of that stack's work, and a failed stack stops later batches.
* `true`: the failure is recorded, and execution continues. The stack is still reported as unsuccessful.

---

## `dependsOn`

```yaml
dependsOn: [build, test]
```

Dependencies are resolved before execution. Stacks with no dependency between them can run in the same batch. Running one stack with `seristack trigger -s <name>` ignores its `dependsOn`, and so does calling it through HTTP or MCP.

---

## `vars`

```yaml
vars:
  - name: env
    value: staging
    allowed_value: [staging, production]
  - name: version
    allowed_regex: regex("^[a-zA-Z0-9._-]+$")
  - name: token
    sensitive: true
```

Use a variable in a command as `{{.Vars.env}}`.

| Field | Type | Default | Description |
|---|---|---|---|
| `name` | string | required | Variable name. |
| `value` | string | empty | Default value. |
| `required` | boolean | `false` | Value must be non-empty. |
| `allowed_value` | list of strings | empty | Value must be one of these. |
| `denied_value` | list of strings | empty | Value must not be one of these. |
| `allowed_regex` | string | empty | Value must match. Written `regex("...")`. |
| `denied_regex` | string | empty | Value must not match. Written `regex("...")`. |
| `sensitive` | boolean | `false` | Keep the value out of the audit log. |

**Where values come from**

| Source | Example |
|---|---|
| Default | `value:` in YAML |
| CLI | `--vars key=value` |
| HTTP query | `?env=staging` |
| HTTP form | `env=staging` |
| HTTP JSON body | `{"env":"staging"}` |
| MCP | a tool argument with the variable's name |

Only variables declared in `vars` can be overridden. Anything else a caller sends is ignored.

**Validation**

* Prefer `allowed_value` and `allowed_regex` over deny-lists. A deny-list such as `(?i)rm|delete|drop` is easy to bypass and also blocks harmless text like `form`.
* Anchor regex patterns with `^` and `$`.
* Values are substituted into command text before the command runs, so constrain every variable that callers can set.

**`sensitive`**

`sensitive: true` removes the variable's value from the audit log. It does not hide the value from command output (if your script prints it, it appears in the response and audit `output`), and validation error messages can include the rejected value. Prefer having scripts read secrets from their own environment.

---

## Template values

Templates can be used in `cmds` and `output`.

| Template | Meaning |
|---|---|
| `{{.Vars.name}}` | Value of a declared variable. |
| `{{.Count.index}}` | Index of the current repeat, starting at 0. |
| `{{.Self.result}}` | Output this stack has collected so far. |
| `{{.Result.stack_name}}` | Output of another stack that ran earlier. |

A stack's output is the text its commands printed. **stdout and stderr are combined**, so warnings written to stderr become part of the output. Use `2>/dev/null` to keep them out.

Output pasted into a command becomes part of that command's command line, which operating systems limit (about 32,000 characters on Windows, about 128 KB per argument on Linux). Use these templates for small values. For large data, write a file in one stack and read it in the next.

---

## `executionMode`

```yaml
executionMode: SEQUENTIAL
```

| Mode | Repeats (`count`) | Commands within one repeat |
|---|---|---|
| `PARALLEL` (default) | concurrent | concurrent |
| `STAGE` | concurrent | one after another |
| `PIPELINE` | one after another | concurrent |
| `SEQUENTIAL` | one after another | one after another |

Output order is only deterministic in `SEQUENTIAL`. Use it when order matters or commands touch shared state.

By default at most 10 commands run at the same time across the whole process.

---

## `count`

```yaml
count: 3
```

How many times the stack runs. `count: 0` (the default) skips it, and the result reads `<name> skipped`. Use `{{.Count.index}}` to tell repeats apart:

```yaml
count: 3
cmds:
  - echo "repeat {{.Count.index}}"
```

---

## `executor`

By default commands run in the built-in mvdan interpreter. To use something else, set `executor`. Each entry in `cmds` is appended as the final argument:

```yaml
executor: ["bash", "-c"]
```

```yaml
executor: ["python3", "-c"]
```

```yaml
executor: ["powershell", "-Command"]
```

For a program that takes no script, set the whole command in `executor` and leave `cmds` empty:

```yaml
executor: ["terraform", "apply", "--auto-approve"]
```

Commands run with Seristack's own permissions and environment. Each command with an external executor starts a new process, which costs a few milliseconds.

---

## `cmds`

```yaml
cmds:
  - echo "single line"
  - |
    echo "starting"
    ./script.sh
    echo "finished"
```

---

## `timeouts`

```yaml
timeouts: 30s
```

Go duration syntax: `ms`, `s`, `m`, `h`, and combinations such as `1h30m` or `2.5h`. Use `24h`, not `1d`.

The timeout applies to **each command**, not to the whole stack. When it expires, Seristack kills the command and everything it started: the whole process group on Linux and macOS, the process tree through `taskkill /T` on Windows (best effort). The kill is immediate, so tools such as Terraform cannot release state locks. Set it well above the longest expected runtime.

---

## `output`

```yaml
output: |
  echo '{{.Self.result}}' | jq -s '{total: length, results: .}'
```

Runs after all repeats finish. Its output **replaces** the stack's collected output. The single quotes around `{{.Self.result}}` break if the output contains a single quote, and this example needs `jq` installed.

---

## `discardOutput`

```yaml
discardOutput: [build, test]
```

Releases the stored output of the listed stacks from memory after this stack finishes.

---

## Complete example

```yaml
stacks:

  - name: build
    description: Build the application
    count: 1
    executionMode: SEQUENTIAL
    timeouts: 10m
    cmds:
      - echo "building"

  - name: test
    description: Run application tests
    dependsOn: [build]
    count: 1
    executionMode: SEQUENTIAL
    timeouts: 10m
    cmds:
      - echo "testing"

  - name: deploy
    description: Deploy the application
    method: POST
    urlPath: /deploy
    dependsOn: [test]
    matchAccess: ALL
    access:
      - headerName: X-Auth-Request-Groups
        headerValue: [sre, platform]
      - headerName: X-Auth-Request-Roles
        headerValue: [admin]
    count: 1
    executionMode: SEQUENTIAL
    timeouts: 10m
    vars:
      - name: env
        value: staging
        allowed_value: [staging, production]
      - name: version
        allowed_regex: regex("^[a-zA-Z0-9._-]+$")
    cmds:
      - |
        echo "Deploying {{.Vars.version}} to {{.Vars.env}}"

  - name: cleanup
    dependsOn: [deploy]
    count: 1
    timeouts: 30s
    discardOutput: [build, test]
    cmds:
      - echo "cleanup complete"
```