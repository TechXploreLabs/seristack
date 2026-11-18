# Seristack Guide

Detailed behavior and security notes. For installation and a quick start, see the [README](../README.md). For every YAML field, see the [Configuration Reference](config-reference.md).

## Configuration

Seristack uses YAML to define stacks. For every field, see the [Configuration Reference](docs/config-reference.md).

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

The `output` example needs `jq` installed, and the single quotes around `{{.Self.result}}` break if the output itself contains a single quote.

## Running stacks

```bash
# Run all stacks
seristack trigger -c config.yaml

# Run one stack
seristack trigger -c config.yaml -s stack1

# Start the HTTP server
seristack run -c config.yaml

# Start the MCP server
seristack mcp -t streamableHTTP
```

## Executors

By default, commands run in the **built-in mvdan shell interpreter**. It runs inside the Seristack process, so it behaves the same on every operating system and does not start a separate shell for builtins. Programs your script calls (`curl`, `git`, and so on) are still started as normal processes.

To use another shell or runtime, set `executor`. Each entry in `cmds` is appended as the final argument:

```yaml
stacks:
  - name: with-bash
    count: 1
    executor: ["bash", "-c"]
    cmds:
      - echo "runs in bash"

  - name: with-python
    count: 1
    executor: ["python3", "-c"]
    cmds:
      - print("runs in python")
```

For a tool that takes no script argument, set `executor` to the full command and leave `cmds` empty:

```yaml
stacks:
  - name: plan
    count: 1
    executor: ["terraform", "plan"]
```

Each command with an external executor starts a new process, which usually costs a few milliseconds. For thousands of tiny commands, put them in one script instead.

## Execution modes

| Mode | Repeats (`count`) | Commands within one repeat |
|---|---|---|
| `SEQUENTIAL` | one after another | one after another |
| `PARALLEL` (default) | concurrent | concurrent |
| `STAGE` | concurrent | one after another |
| `PIPELINE` | one after another | concurrent |

Output order is only deterministic in `SEQUENTIAL` mode. In the other modes, output from concurrent commands is added as each command finishes.

Concurrency is bounded: by default at most 10 commands run at the same time across the whole process.

## Timeouts

`timeouts` (for example `30s`, `5m`, `1h`) limits **each command**, not the whole stack. A stack with ten sequential commands and `timeouts: 5m` can run for up to fifty minutes.

When a command exceeds its timeout, Seristack kills it **and everything it started**:

* Linux and macOS: the whole process group is killed.
* Windows: the process tree is killed with `taskkill /T` (best effort).

The kill is immediate. Tools such as Terraform get no chance to release state locks or finish writing state. For infrastructure-changing commands, set the timeout well above the longest expected runtime.

Processes that deliberately detach themselves (daemons, `setsid`) may survive.

## Variables

Variables are declared in `vars` and used in commands as `{{.Vars.name}}`.

```yaml
vars:
  - name: env
    value: staging
    allowed_value:
      - staging
      - production
  - name: version
    allowed_regex: regex("^[a-zA-Z0-9._-]+$")
  - name: token
    sensitive: true
```

Only variables declared in `vars` can be overridden by HTTP requests or MCP arguments. Undeclared values supplied by a caller are ignored.

Available fields:

* `value`: default value
* `required`: reject empty values
* `allowed_value` / `denied_value`: exact-match lists
* `allowed_regex` / `denied_regex`: written as `regex("...")`
* `sensitive`: keep the value out of the audit log

Prefer allow-lists (`allowed_value`, `allowed_regex`) over deny-lists. A deny-list like `(?i)rm|delete|drop` is easy to bypass and also blocks harmless words such as `form`. Anchor regex patterns with `^` and `$`.

Values are substituted into command text before the command runs, so a value containing shell syntax can change what the command does. Always constrain variables that come from HTTP or MCP callers.

### Sensitive variables

`sensitive: true` removes the variable's value from the audit log. It does **not** scrub command output: if your script prints the value, it appears in the response and in the audit `output` field. Validation error messages can also include the rejected value.

Where possible, have your script read secrets from its own environment instead of receiving them as variables, and never send secrets in URL query strings, because proxies log them.

## Dependencies and sharing output

Stacks can depend on other stacks:

```yaml
stacks:
  - name: build
    count: 1
    cmds:
      - ./build.sh

  - name: test
    count: 1
    dependsOn:
      - build
    cmds:
      - ./test.sh

  - name: deploy
    count: 1
    dependsOn:
      - test
    cmds:
      - ./deploy.sh
```

Seristack resolves the order before execution. Stacks with no dependency between them can run in the same batch, and a failed stack (without `continueOnError`) stops later batches.

### Template values

| Template | Meaning |
|---|---|
| `{{.Vars.name}}` | variable value |
| `{{.Count.index}}` | index of the current repeat (0-based) |
| `{{.Self.result}}` | output collected so far by this stack |
| `{{.Result.stack_name}}` | output of another stack that ran earlier |

A stack's output is the text its commands printed. **stdout and stderr are combined** into one string, so warnings written to stderr become part of the output. Redirect stderr (`2>/dev/null`) if you need clean data.

Stack names used in `{{.Result.name}}` should use letters, digits, and underscores. Names containing `-` are rejected by Go's template parser.

`discardOutput` lists stacks whose stored output should be released from memory once the current stack finishes.

### Limits on large output

Seristack keeps a command's full output and does not truncate it. But output substituted into another command becomes part of that command's command line, and operating systems limit its length: about 32,000 characters on Windows, and about 128 KB per argument on Linux. For large data, write it to a file in one stack and read the file in the next:

```yaml
- name: export
  cmds:
    - cat bigfile.txt > /tmp/export.txt
- name: process
  dependsOn: [export]
  cmds:
    - wc -l /tmp/export.txt
```

## HTTP server

Stacks with a `method` are exposed as HTTP endpoints:

```yaml
stacks:
  - name: system-health
    count: 1
    method: GET
    urlPath: /health
    cmds:
      - echo "system-health is good"
```

```bash
seristack run --config config.yaml --addr 127.0.0.1 --port 8080
```

Variables can be supplied as query parameters, form fields, or a JSON body. The response is a JSON object with the stack's name, success flag, output, error, and duration.

Seristack serves plain HTTP. It has no TLS and no built-in authentication.

## MCP server

Stacks with a `description` can be exposed as MCP tools:

```bash
seristack mcp -t streamableHTTP --addr 127.0.0.1 --port 8081
```

Supported transports: `stdio`, `sse`, and `streamableHTTP`. Each declared variable becomes a tool argument. A variable is required unless it has a default `value` or an environment variable named `SS_VAR_<name>`.

Calling a tool runs that stack only; it does not run the stack's `dependsOn` stacks. Stacks with `access` rules are not visible over `stdio`, because there are no request headers to check.

Treat MCP access with the same care as an HTTP API: any client that can call a tool can run its commands.

## Production deployment

Seristack runs shell commands and must **not be exposed directly to the public internet**.

```text
Client
  │
  ▼
nginx / Caddy / oauth2-proxy
  ├── TLS termination
  ├── Authentication
  ├── Rate limiting
  └── Sets identity headers
  │
  ▼
Seristack (127.0.0.1)
  ├── Per-stack authorization
  └── Command execution
```

The same pattern applies to the MCP server. Notes for the proxy:

* **Bind to loopback.** Run Seristack on `127.0.0.1` so the proxy cannot be bypassed.
* **Own the identity headers.** The proxy must remove any identity headers sent by the client and set its own. Seristack trusts the headers it receives.
* **Raise read timeouts.** Seristack imposes no write timeout, so long stacks keep running, but proxies often cut idle connections early (nginx defaults to 60 seconds, set `proxy_read_timeout` above your longest stack).
* **Disable response buffering for MCP** (`proxy_buffering off;` in nginx), so streamed responses are delivered immediately.

## Per-stack authorization

Seristack can restrict individual stacks using identity headers forwarded by an authenticated proxy:

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

`matchAccess: ANY` grants access when at least one rule matches (OR). `matchAccess: ALL` requires every rule to match (AND).

A stack with **no `access` block has no stack-level restriction**: anyone who can reach the port can run it. Protect the service itself with the proxy.

| Identity provider | Common proxy | Example headers |
|---|---|---|
| Entra ID / Azure AD | oauth2-proxy | `X-Auth-Request-Groups`, `X-Auth-Request-Roles`, `X-Auth-Request-Email` |
| GCP | IAP | `X-Goog-Authenticated-User-Email` |
| AWS Cognito | ALB | `X-Amzn-Oidc-Data` |
| OCI IAM | nginx + oauth2-proxy | `X-Auth-Request-Groups`, `X-Auth-Request-Email` |
| Okta / Auth0 | oauth2-proxy | headers configured in the proxy |

Only trust these headers when they come from your authenticated proxy. If clients can reach Seristack directly, they can send any header they like.

## Audit logging

```bash
seristack run --config config.yaml --audit-log /var/log/seristack/audit.log
```

Each entry is a JSON line. Fields include the request ID, stack name, HTTP path and method, source IP, identity headers, variables (without `sensitive` ones), success flag, duration, output, and error.

Notes:

* Audit entries include the full command output, so logs can grow large. Rotate them with `logrotate` or your logging system.
* Request headers are recorded as identity unless excluded. Exclude headers that carry credentials (`Authorization`, `Cookie`, API keys) with `--identity-header`.
* Variables that are not marked `sensitive` are written to the log. Do not pass secrets through them.

## `seristack run` flags

```text
seristack run [flags]

  -c, --config string        config file (default "config.yaml")
  -a, --addr string          bind address (default "127.0.0.1")
  -p, --port string          server port (default "8080")
      --audit-log string     path to audit log file; enables audit logging when set
      --identity-header strings
                             identity header handling, repeatable
```

Run `seristack run --help` for the current list.

## Security considerations

Seristack executes commands on the host where it runs, with the permissions and environment of the Seristack process.

1. Keep Seristack bound to localhost or a private network.
2. Put an authenticating reverse proxy in front of anything reachable from outside, with TLS.
3. Do not trust identity headers that clients can set directly.
4. Restrict sensitive stacks with `access` rules. Stacks without them are open.
5. Constrain every variable that callers can set with allow-lists.
6. Do not pass secrets as variables unless marked `sensitive`, and remember output is not scrubbed.
7. Seristack is not a sandbox, and the mvdan interpreter is not either. A command can do anything the Seristack user can do, and child processes inherit its environment variables.
8. Run Seristack as a user with the minimum privileges needed, ideally in a container.
9. Review and rotate audit logs.
10. Review every command before exposing it through HTTP or MCP.

## Testing

```bash
go test ./...                     # run all tests
go test -count=1 ./...            # skip the test cache
go test -count=1 -v ./pkg/opsy    # one package, verbose
CGO_ENABLED=1 go test -race ./... # race detector (needs a C compiler)
```

Check that every release platform compiles:

```bash
GOOS=linux go build ./... && GOOS=darwin go build ./... && GOOS=windows go build ./...
```