# seristack(v0.3.4)

[![Go Report Card](https://goreportcard.com/badge/github.com/TechXploreLabs/seristack)](https://goreportcard.com/report/github.com/TechXploreLabs/seristack)
[![Go Reference](https://pkg.go.dev/badge/github.com/TechXploreLabs/seristack.svg)](https://pkg.go.dev/github.com/TechXploreLabs/seristack)
[![Go Version](https://img.shields.io/badge/go-1.25.5-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/github/license/TechXploreLabs/seristack)](LICENSE)
[![Release](https://img.shields.io/github/v/release/TechXploreLabs/seristack?include_prereleases)](https://github.com/TechXploreLabs/seristack/releases)

**Run shell workflows via CLI or HTTP

Seristack is a lightweight automation engine designed to bridge the gap between local task execution and remote triggers. Define your stacks in YAML, manage dependencies, and expose your automation via a built-in HTTP server.

[seristack](https://github.com/TechXploreLabs/seristack)

Documentation:

- [Config Reference](docs/config-reference.md)

# Features

    🚀 Run multiple command stacks from a single config
    🔁 Repeat stacks with serial or concurrent execution
    🔗 Define dependencies between stacks
    🧩 Variable substitution using templates
    📦 Share output between stacks
    🌐 Expose stacks as HTTP endpoints
    🧠 Run as an MCP server for IDE integrations
    🛠 Works with mvdan shell (default), Bash, sh, and PowerShell


## Installation

### Using Homebrew (Mac and Linux)

```bash
brew install TechXploreLabs/tap/seristack
```

### Linux (using release archive)

1. Go to [Seristack Releases](https://github.com/TechXploreLabs/seristack/releases) and download the latest `seristack_VERSION_linux_ARCH.tar.gz` (`ARCH` matches your system, e.g., `amd64`, `arm64`).
2. Extract the archive:
   ```bash
   tar -xzf seristack_VERSION_linux_ARCH.tar.gz
   ```
3. Move the `seristack` binary to a directory in your `PATH`:
   ```bash
   sudo mv seristack /usr/local/bin/
   ```
4. Set execute permissions (just in case):
   ```bash
   sudo chmod +x /usr/local/bin/seristack
   ```
5. Verify installation:
   ```bash
   seristack --help
   ```

### Windows (using release archive)

1. Go to [Seristack Releases](https://github.com/TechXploreLabs/seristack/releases) and download the latest `seristack_VERSION_windows_ARCH.zip` or `.gz` file (where `ARCH` matches your system, e.g., `amd64`).
2. Extract the zip/gz file (Right click → Extract all, or use a tool like 7-Zip).
3. Move `seristack.exe` to a folder in your `%PATH%` (such as `C:\Windows`, or better, a custom tools folder included in PATH).
4. Open PowerShell or Command Prompt and verify installation:
   ```powershell
   seristack --help
   ```


# Sample stack yaml file

For a full explanation of every YAML attribute, see the [Config Reference](docs/config-reference.md).

```yaml
# description about seristack
# config.yaml

stacks:
  - name: stack1                # name of the stack (REQUIRED)
    workDir: ./                 # working directory to execute the cmds. default is "./"
    description: Used for printing  
                  welcome message               # used for adding the stack as tool in mcp server, if descrption is empty then 
                                                # it won't be added
    method: GET                 # Http methods needs to be added for http server
    urlPath: /show              # Optional, If not provided stack name will be added as path, ex /stack1 
    continueOnError: false      # if cmds has error, true will not stop execution, false will stop. default is false
    count: 3                    # count = 0 will not run cmds, count = 3 runs entire cmds three times. default is 0
    timeouts: 1h                # timeout for each command execution in this stack. default is 1h
                                # supports Go duration values like 500ms, 30s, 5m, 1h, 1h30m
    executionMode: PARALLEL     # if count = 3 and executionMode is PARALLEL, then all three iterations of list 
                                # cmds execute parallellely . Valid options are, [PARALLEL/STAGE/PIPELINE/SEQUENTIAL]. 
                                # STAGE = execute all iterations conncurrently, list of cmds execeuted serially
                                # PIPELINE = execute all iterations serially, list of cmds executed concurrently
                                # SEQUENTIAL = execute all iterations and theirs cmds serially. default is PARALLEL
    
    vars:                       # vars takes list of variable objects. default is empty
      - name: samplekey
        value: samplevalue
        required: true # optional
        allowed_value: [samplevalue, devvalue]  # optional
        # denied_value: [blocked]               # optional
        # allowed_regex: regex("^[a-z]+$")     # optional
        # denied_regex: regex("(?i)rm")        # optional
        # Note: only one rule set can be used among
        # allowed_value / denied_value / allowed_regex / denied_regex
    discardOutput: [stack1] # Discard the output saved in the memory, after current stack completes
    shell: bash                 # optional. if not provided, mvdan shell interpreter is used by default
    shellArg: -c                # optional for external shells
    dependsOn: []               # dependsOn takes list of stacks to start after them. default is []
    cmds:                       # cmds takes list of shell commands (linux, powershell)
      - |
        export samplekey={{.Vars.samplekey}}    # to use vars for substitution
        echo $samplekey
        echo "count={{.Count.index}}"         # index of count iterations
        echo "Hey i'm seristack!"

  - name: stack2
    workDir: ./
    continueOnError: false
    count: 3
    executionMode: SEQUENTIAL
    vars:
      - name: env
        value: Dev
    dependsOn: [stack1]          # runs after stack1 completes
    cmds:
      - |
        # Command 1: Produces metadata
        echo "{\"index\": {{.Count.index}}, \"step\": \"metadata\", \"status\": \"ok\"}"
      - |
        # Command 2: Produces metric data
        echo "{\"index\": {{.Count.index}}, \"step\": \"metrics\", \"value\": $((RANDOM % 100))}"  
    output: |  # for aggregating outputs from the cmds
      echo "--- Aggregation Summary ---"
      # We use 'grep' to find JSON lines and 'jq' to format them into an array
      echo '{{.Self.result}}' | grep "^{" | jq -s '{
        total_records: length,
        environment: "{{.Vars.env}}",
        results: .
      }'
  - name: stack3
    workDir: ./
    count: 1
    vars:
      - name: invite
        value: hello engineers
    cmds:
      - |
        echo "Current date and time:"
        echo `date`
```

# Running the stacks

1. Trigger entire stacks, default is config.yaml.

```bash
seristack trigger -c config.yaml

or

seristack trigger
```

2. Run the particular stack.

```bash
seristack trigger -c config.yaml -s stack3
```

3. Init the http server with endpoint. ctrl+c will stop the server process.

```bash
seristack run -c config.yaml
```

4. Init the mcpserver. ctrl+c will stop the server process.

```bash
seristack mcp -t streamableHTTP
```

# Production deployment and authentication

Seristack can execute shell commands, so avoid exposing it directly to the public internet.
The recommended production pattern is to run Seristack on `127.0.0.1` or a private network
and put a reverse proxy such as **Nginx** or **Caddy** in front of it.

Use the reverse proxy for:

- TLS/HTTPS termination
- authentication and authorization
- IP allowlists
- rate limiting
- request body size limits
- access logging

Seristack focuses on stack execution, variable validation, HTTP endpoint routing, and MCP tool
exposure. Authentication is intentionally best handled at the edge by a battle-tested proxy.

## HTTP API behind a reverse proxy

Start Seristack bound to localhost:

```bash
seristack run --config config.yaml --addr 127.0.0.1 --port 8080
```

Then expose it through Nginx or Caddy.

### Nginx example with Basic Auth

Create a password file:

```bash
htpasswd -c /etc/nginx/.seristack_htpasswd admin
```

Example Nginx config:

```nginx
server {
    listen 443 ssl;
    server_name seristack.example.com;

    ssl_certificate /etc/letsencrypt/live/seristack.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/seristack.example.com/privkey.pem;

    client_max_body_size 1m;

    location / {
        auth_basic "Seristack";
        auth_basic_user_file /etc/nginx/.seristack_htpasswd;

        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Request-ID $request_id;
    }
}
```

For stricter production use, add Nginx rate limiting and/or IP allowlists.

### Caddy example with Basic Auth

Generate a password hash:

```bash
caddy hash-password
```

Example `Caddyfile`:

```caddyfile
seristack.example.com {
    request_body {
        max_size 1MB
    }

    basicauth {
        admin <PASTE_CADDY_HASHED_PASSWORD_HERE>
    }

    reverse_proxy 127.0.0.1:8080 {
        header_up X-Request-ID {http.request.uuid}
    }
}
```

## MCP server security

The same rule applies to MCP transports. Start MCP on localhost/private networking unless you
are deliberately exposing it through a protected proxy:

```bash
seristack mcp --type streamableHTTP --addr 127.0.0.1 --port 8080
```

If exposing MCP externally, protect it with Nginx/Caddy authentication, TLS, rate limits, and
network restrictions. MCP clients can trigger the configured stack tools, so treat MCP endpoints
with the same security posture as the HTTP API.

## Request validation

Seristack supports variable-level validation in `vars` using:

- `required`
- `allowed_value`
- `denied_value`
- `allowed_regex`
- `denied_regex`

Use these rules to restrict inputs accepted by HTTP endpoints and MCP tools.

## Command timeout values

The stack-level `timeouts` field controls the maximum duration allowed for each command execution
inside that stack.

If `timeouts` is not set, Seristack uses the default command timeout: **`1h`**.

Seristack uses Go duration syntax through `time.ParseDuration`, so timeout values support these
units:

- `ns` — nanoseconds
- `us` or `µs` — microseconds
- `ms` — milliseconds
- `s` — seconds
- `m` — minutes
- `h` — hours

Examples:

```yaml
timeouts: 500ms   # 500 milliseconds
timeouts: 30s     # 30 seconds
timeouts: 5m      # 5 minutes
timeouts: 1h      # 1 hour
timeouts: 1h30m   # 1 hour and 30 minutes
timeouts: 2.5h    # 2 hours and 30 minutes
```

Timeouts must be greater than zero. Values like `0s`, `-1m`, `1d`, or `never` are invalid.
Use `24h` instead of `1d` if you need a one-day timeout.

# Support the project

If Seristack helps you turn shell scripts or runbooks into internal APIs and MCP tools, consider
supporting the project by starring the repository, sharing feedback, opening issues, contributing
examples, or sponsoring future development.


# License

Apache License
