# Host SeriStack MCP with Auth0, Nginx, and oauth2-proxy

Give your team one MCP URL, browser sign-in, and access to a defined set of operational tools. SeriStack executes those tools on a VM that holds the cloud, Kubernetes, monitoring, or CI/CD credentials the tools need.

Users connect from Cline, Codex, or Claude. They authenticate with Auth0 and receive tools according to their permissions. They do not need the server's Auth0 client secret, a shared password, or a manually distributed access token.

**Guide baseline:** Ubuntu 22.04/24.04 LTS, SeriStack **v0.4.6**, oauth2-proxy **v7.15.4**, and Streamable HTTP. Documentation checked on **23 September 2026**. The versions are pinned for reproducibility; evaluate newer releases separately. Complete the deployment checks below against your own tenant and VM before sharing the endpoint.

**Contents:** [Architecture](#1-architecture-and-authorization-model) · [Prerequisites](#2-prerequisites-and-values-to-replace) · [Auth0](#3-configure-auth0) · [Installation](#4-install-packages-and-pinned-binaries) · [Service users](#5-create-service-users-and-directories) · [oauth2-proxy](#6-configure-oauth2-proxy) · [SeriStack](#7-define-the-sample-tools-and-start-seristack) · [TLS](#8-obtain-a-certificate) · [Nginx](#9-configure-nginx-for-mcp-authentication-and-discovery) · [Verification](#10-verify-the-deployment) · [Clients](#11-connect-users-clients) · [Operations](#12-operate-the-service) · [Operational tools](#13-expand-into-operational-tools) · [Troubleshooting](#14-troubleshooting) · [Acceptance](#15-deployment-acceptance)

## 1. Architecture and authorization model

| Component | Responsibility |
|---|---|
| MCP client | Discovers OAuth metadata, opens browser sign-in, obtains and renews tokens, and calls tools. |
| Auth0 | Authenticates users and issues signed tokens containing their API permissions. |
| Nginx | Terminates TLS, sends an authentication subrequest, and replaces identity headers before forwarding requests. |
| oauth2-proxy | Validates supported bearer tokens and maps the `permissions` claim to `X-Auth-Request-Groups`. |
| SeriStack | Filters tool discovery and checks access again when a tool is called. Executes permitted commands as the service user. |
| VM service account and downstream credentials | Determine what those commands can do in cloud accounts, clusters, and external services. |

An unauthenticated request to `/mcp` receives **401** with a `WWW-Authenticate` header pointing to protected-resource metadata. The client discovers Auth0, registers an OAuth client where supported, and performs authorization code login with PKCE. After login it sends a bearer access token on MCP requests.

This guide deliberately enforces **user API permissions** through Auth0's `permissions` claim. That claim can contain all permissions assigned to the user for the API. It is different from `scope`, which reflects requested and granted permissions. The configuration below does not additionally restrict tool execution to a narrower client-specific scope grant. If that is a requirement, enforce the intersection of the required user permission and granted scope in a validating gateway or server. [Auth0 RBAC behavior](https://auth0.com/docs/manage-users/access-control/configure-core-rbac/enable-role-based-access-control-for-apis)

The execution VM is a trusted host. Binding SeriStack to loopback prevents direct remote access, but other local processes can still connect and supply identity headers. Use a dedicated VM without untrusted local users or workloads. All tool executions share the service user's operating-system access and the configured downstream identities; an Auth0 user does not automatically become a distinct cloud IAM principal.

Tool filtering is present in the pinned release. Older v0.4.4 deployments checked calls but did not filter discovery. Hiding tools improves discovery; execution-time authorization remains the control that denies unauthorized calls. It does not make authorized tools immune to prompt injection. [SeriStack v0.4.6 implementation](https://github.com/TechXploreLabs/seristack/blob/v0.4.6/internal/mcpserver/mcpserver.go)

## 2. Prerequisites and values to replace

Prepare an Ubuntu VM with a reserved public IP, outbound HTTPS, and a DNS record pointing to it. Allow inbound TCP 80/443 in the cloud firewall and host firewall. Restrict SSH to your administrator network or bastion. Do not expose 4180 or 8081. Publish an AAAA record only if IPv6 routing and firewall rules work too.

Use these values consistently:

| Placeholder | Meaning | Example for your deployment |
|---|---|---|
| `mcp.example.com` | Public MCP hostname | `seristackmcp.getsaas.in` |
| `YOUR_TENANT.us.auth0.com` | Auth0 tenant domain, without a scheme or trailing slash | Your Auth0 tenant domain |
| `https://mcp.example.com/mcp` | MCP endpoint and Auth0 API identifier | `https://seristackmcp.getsaas.in/mcp` |
| `REPLACE_PROXY_CLIENT_ID` | Client ID of the Regular Web Application | Copy from Auth0 |
| `REPLACE_PROXY_CLIENT_SECRET` | Secret of that same application | Keep on the VM |

**Replace the literal placeholders in configuration blocks before saving them.** Setting shell variables does not substitute values inside pasted Nginx, YAML, or systemd files.

Run the server commands in Bash on the VM. Set:

```bash
export MCP_HOST='mcp.example.com'
export AUTH0_DOMAIN='YOUR_TENANT.us.auth0.com'
```

The hostname has no scheme or path. The Auth0 **issuer URL**, used later, has both `https://` and a trailing `/`.

## 3. Configure Auth0

### 3.1 Enable MCP discovery and registration support

In **Settings → Advanced**, enable:

- **Dynamic Client Registration** for clients that register themselves.
- **Resource Parameter Compatibility Profile** so the MCP `resource` parameter can select the API audience.
- **Include Issuer in Authorization Responses** for authorization-server identification.

This guide uses DCR for URL-based onboarding. DCR allows clients to obtain an application registration; it does not assign users permissions. Treat open registration as a tenant policy decision and monitor application counts. Clients that require pre-registration need their own registration and exact callback URLs. [Auth0 MCP setup](https://auth0.com/ai/docs/mcp/get-started/authorization-for-your-mcp-server), [Auth0 DCR](https://auth0.com/docs/get-started/applications/dynamic-client-registration)

### 3.2 Enable the login connection for third-party clients

Open **Authentication → Database → Username-Password-Authentication** and enable **Promote Connection to Domain Level**. This exposes the selected connection to third-party clients created through DCR. The standard Auth0 database connection stores users in Auth0; you do not need your own database.

For an internal team, enable **Disable Sign Ups** and provision users deliberately. If you use an enterprise connection instead, configure that connection for the same third-party login requirement. Promote only the connection you intend these clients to use. [Third-party connection setup](https://auth0.com/ai/docs/mcp/get-started/authorization-for-your-mcp-server)

### 3.3 Create one API for the MCP resource

Under **Applications → APIs**, create:

| Setting | Value |
|---|---|
| Name | `SeriStack MCP` |
| Identifier | `https://mcp.example.com/mcp` |
| Signing algorithm | `RS256` |
| Enable RBAC | On |
| Add Permissions in the Access Token | On |
| Allow Offline Access | On |
| User-delegated application access | Per-app authorization |
| Client access | No apps allowed, unless you separately implement a machine-to-machine flow |

Add these API permissions:

```text
groups:developers
groups:platform
groups:platform-admin
```

Under **Default Permissions for third-party applications**, set **User-delegated Access → Authorized** and select those three permissions. Leave third-party **Client Access → Unauthorized**. This allows new third-party clients to request the permitted user-delegated access; it does not make every user an administrator. User permission assignments remain necessary. Review overrides in **Application Access** if a particular client is denied. [Auth0 application registration and API access](https://auth0.com/docs/get-started/applications/dynamic-client-registration)

Choose an access-token lifetime consistent with how quickly revoked access must stop working. Shorter lifetimes reduce the period an already-issued token can remain usable; refresh support reduces repeated interactive sign-in.

**Migrating an existing deployment:** a second API identified by `https://mcp.example.com` is not required for this fresh setup. Keep it only while existing clients still request that old audience. During migration, oauth2-proxy can accept both explicitly configured audiences; assign permissions on the correct API and retire the old audience when those clients have moved.

### 3.4 Create roles and assign users

Create roles under **User Management → Roles**:

| Role | Permission from the `/mcp` API |
|---|---|
| `devops` | `groups:devops` |
| `sre` | `groups:sre` |
| `cloud-infra` | `groups:cloud-infra` |
| `platform-admin` | `groups:platform-admin` |

Assign the required role to each user. A role name has no implicit hierarchy. A user can have multiple roles or directly assigned permissions, so effective access is based on the resulting permission values, not only the displayed role name.

No custom Post Login Action is needed to copy permissions into a `groups` claim. This setup reads Auth0's built-in `permissions` claim. Remove the earlier experimental action that wrote `groups` and `permissions` if that was its only purpose; retain unrelated action behavior.

### 3.5 Create the proxy's Regular Web Application

Under **Applications → Applications**, create a **Regular Web Application** named `SeriStack OAuth2 Proxy`.

- Register `https://mcp.example.com/oauth2/callback` as its allowed callback URL.
- Enable your chosen login connection for the application.
- Record its Client ID and Client Secret for the server environment file.

This is oauth2-proxy's confidential OIDC client configuration. MCP clients obtain their own OAuth registrations and callbacks. Do not distribute this application's secret or enter it in Cline, Codex, or Claude. A Native application and password grant are not needed for the browser OAuth flow described here.

The Nginx configuration below exposes only the proxy's internal authentication check. It does not publish its optional cookie-login endpoints. The MCP endpoint uses bearer tokens and its own OAuth discovery challenge.

### 3.6 Let users set their own passwords

For an Auth0 database connection:

1. Create the user with their real email address and a unique random initial password. Do not distribute that initial password as a shared credential.
2. Use Auth0's password reset email workflow so the user chooses their password through an expiring link.
3. Assign the user's role from the MCP API.
4. Confirm that the user can sign in and sees the intended tools.

Configure a production email provider and test delivery before onboarding the team. A password reset is not automatically a role assignment. For federated users, password setup and reset belong to their identity provider. [Auth0 password reset documentation](https://auth0.com/docs/authenticate/database-connections/password-change)

## 4. Install packages and pinned binaries

Install the host packages:

```bash
sudo apt update
sudo apt install -y nginx curl ca-certificates tar jq python3 openssl logrotate certbot
sudo systemctl enable --now nginx
```

If this VM already uses Certbot from Snap, keep that installation instead of installing a second copy. Configure the existing firewall manager to persist the required rules; do not mix independent UFW, nftables, and iptables configurations. With an already enabled UFW configuration, the web rule is:

```bash
sudo ufw allow 'Nginx Full'
```

Do not enable or reset a firewall during SSH access without preserving its SSH rule.

### 4.1 Install SeriStack v0.4.6

This downloads a pinned release, checks its archive against the release checksum file, and installs the binary:

```bash
(
  set -euo pipefail
  SERISTACK_VERSION='0.4.6'
  SERISTACK_ARCH="$(dpkg --print-architecture)"
  case "$SERISTACK_ARCH" in
    amd64|arm64) ;;
    *) printf 'Unsupported architecture: %s\n' "$SERISTACK_ARCH" >&2; exit 1 ;;
  esac
  SERISTACK_DOWNLOAD_DIR="$(mktemp -d)"
  trap 'rm -rf "$SERISTACK_DOWNLOAD_DIR"' EXIT
  cd "$SERISTACK_DOWNLOAD_DIR"
  SERISTACK_ARCHIVE="seristack_${SERISTACK_VERSION}_linux_${SERISTACK_ARCH}.tar.gz"
  SERISTACK_RELEASE="https://github.com/TechXploreLabs/seristack/releases/download/v${SERISTACK_VERSION}"
  curl -fsSLO "$SERISTACK_RELEASE/$SERISTACK_ARCHIVE"
  curl -fsSLO "$SERISTACK_RELEASE/seristack_${SERISTACK_VERSION}_checksums.txt"
  awk -v file="$SERISTACK_ARCHIVE" '$2 == file {print; found=1} END {if (!found) exit 1}' \
    "seristack_${SERISTACK_VERSION}_checksums.txt" > selected.sha256
  sha256sum -c selected.sha256
  tar -xzf "$SERISTACK_ARCHIVE"
  sudo install -o root -g root -m 0755 seristack /usr/local/bin/seristack
)
seristack -v
```

Release: [SeriStack v0.4.6](https://github.com/TechXploreLabs/seristack/releases/tag/v0.4.6).

### 4.2 Install oauth2-proxy v7.15.4

```bash
(
  set -euo pipefail
  OAUTH2_PROXY_VERSION='v7.15.4'
  OAUTH2_PROXY_ARCH="$(dpkg --print-architecture)"
  case "$OAUTH2_PROXY_ARCH" in
    amd64|arm64) ;;
    *) printf 'Unsupported architecture: %s\n' "$OAUTH2_PROXY_ARCH" >&2; exit 1 ;;
  esac
  OAUTH2_PROXY_DOWNLOAD_DIR="$(mktemp -d)"
  trap 'rm -rf "$OAUTH2_PROXY_DOWNLOAD_DIR"' EXIT
  cd "$OAUTH2_PROXY_DOWNLOAD_DIR"
  OAUTH2_PROXY_DIR="oauth2-proxy-${OAUTH2_PROXY_VERSION}.linux-${OAUTH2_PROXY_ARCH}"
  OAUTH2_PROXY_ARCHIVE="${OAUTH2_PROXY_DIR}.tar.gz"
  OAUTH2_PROXY_RELEASE="https://github.com/oauth2-proxy/oauth2-proxy/releases/download/${OAUTH2_PROXY_VERSION}"
  curl -fsSLO "$OAUTH2_PROXY_RELEASE/$OAUTH2_PROXY_ARCHIVE"
  curl -fsSLO "$OAUTH2_PROXY_RELEASE/${OAUTH2_PROXY_ARCHIVE}-sha256sum.txt"
  sha256sum -c "${OAUTH2_PROXY_ARCHIVE}-sha256sum.txt"
  tar -xzf "$OAUTH2_PROXY_ARCHIVE"
  sudo install -o root -g root -m 0755 "$OAUTH2_PROXY_DIR/oauth2-proxy" /usr/local/bin/oauth2-proxy
)
oauth2-proxy --version
```

Release: [oauth2-proxy v7.15.4](https://github.com/oauth2-proxy/oauth2-proxy/releases/tag/v7.15.4). Checksums detect a mismatched download; they do not independently establish trust if both archive and checksum source are compromised.

## 5. Create service users and directories

```bash
id -u oauth2-proxy >/dev/null 2>&1 || \
  sudo useradd --system --no-create-home --shell /usr/sbin/nologin oauth2-proxy
id -u seristack >/dev/null 2>&1 || \
  sudo useradd --system --home-dir /var/lib/seristack --no-create-home \
    --shell /usr/sbin/nologin seristack

sudo install -d -o root -g root -m 0700 /etc/oauth2-proxy
sudo install -d -o root -g seristack -m 0750 /etc/seristack
sudo install -d -o root -g root -m 0755 /opt/seristack
sudo install -d -o root -g seristack -m 0750 /opt/seristack/scripts
sudo install -d -o seristack -g seristack -m 0750 /var/lib/seristack
sudo install -d -o seristack -g seristack -m 0750 /var/log/seristack
```

Configuration and trusted scripts belong to root. The service may write runtime data and logs, but must not be able to replace its own configuration or scripts through a writable parent directory. Do not give `seristack` blanket sudo access.

## 6. Configure oauth2-proxy

Create `/etc/oauth2-proxy/oauth2-proxy.env` with these contents, replacing the tenant, host, Client ID, and Client Secret:

```ini
OAUTH2_PROXY_PROVIDER=oidc
OAUTH2_PROXY_OIDC_ISSUER_URL=https://YOUR_TENANT.us.auth0.com/
OAUTH2_PROXY_CLIENT_ID=REPLACE_PROXY_CLIENT_ID
OAUTH2_PROXY_CLIENT_SECRET=REPLACE_PROXY_CLIENT_SECRET
OAUTH2_PROXY_REDIRECT_URL=https://mcp.example.com/oauth2/callback

OAUTH2_PROXY_HTTP_ADDRESS=127.0.0.1:4180
OAUTH2_PROXY_REVERSE_PROXY=true
OAUTH2_PROXY_SET_XAUTHREQUEST=true
OAUTH2_PROXY_UPSTREAMS=static://202

OAUTH2_PROXY_EMAIL_DOMAINS=*
OAUTH2_PROXY_SCOPE="openid profile email"
OAUTH2_PROXY_SKIP_CLAIMS_FROM_PROFILE_URL=false

OAUTH2_PROXY_SKIP_JWT_BEARER_TOKENS=true
OAUTH2_PROXY_OIDC_EXTRA_AUDIENCES=https://mcp.example.com/mcp
OAUTH2_PROXY_OIDC_GROUPS_CLAIM=permissions

OAUTH2_PROXY_COOKIE_SECURE=true
OAUTH2_PROXY_COOKIE_HTTPONLY=true
OAUTH2_PROXY_COOKIE_SAMESITE=lax
```

Use `sudoedit` to enter the client secret rather than putting it into a shell command. Lock down the file and append a newly generated cookie secret once:

```bash
sudo chown root:root /etc/oauth2-proxy/oauth2-proxy.env
sudo chmod 0600 /etc/oauth2-proxy/oauth2-proxy.env
python3 -c 'import secrets; print("OAUTH2_PROXY_COOKIE_SECRET=" + secrets.token_urlsafe(32))' \
  | sudo tee -a /etc/oauth2-proxy/oauth2-proxy.env >/dev/null
```

On later edits, keep exactly one `OAUTH2_PROXY_COOKIE_SECRET` entry. The proxy requires this setting even though MCP authentication here uses bearer tokens.

`SKIP_JWT_BEARER_TOKENS=true` means verified bearer tokens can bypass the proxy's interactive cookie login. It does not disable signature checking. `OIDC_EXTRA_AUDIENCES` adds the MCP audience accepted by the provider, and `OIDC_GROUPS_CLAIM=permissions` supplies the groups header. Do not add unrelated audiences. `PASS_ACCESS_TOKEN` and a synthetic `groups` scope are not needed for this header-based authorization path. [oauth2-proxy configuration](https://oauth2-proxy.github.io/oauth2-proxy/configuration/overview/)

Create `/etc/systemd/system/oauth2-proxy.service`:

```ini
[Unit]
Description=OAuth2 Proxy for SeriStack MCP
Wants=network-online.target
After=network-online.target

[Service]
Type=simple
User=oauth2-proxy
Group=oauth2-proxy
EnvironmentFile=/etc/oauth2-proxy/oauth2-proxy.env
ExecStart=/usr/local/bin/oauth2-proxy
Restart=on-failure
RestartSec=5
UMask=0077
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true

[Install]
WantedBy=multi-user.target
```

systemd reads the root-only environment file before starting the unprivileged service.

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now oauth2-proxy
sudo systemctl status oauth2-proxy --no-pager
```

## 7. Define the sample tools and start SeriStack

Create `/etc/seristack/config.yaml`:

```yaml
stacks:
  - name: sre-incident-response
    description: Diagnose production health and collect incident diagnostics
    access:
      - headerName: X-Auth-Request-Groups
        headerValue:
          - "groups:sre"
          - "groups:platform-admin"
    count: 1
    cmds:
      - echo "Production health check HEALTHY"
      - echo "Incident diagnostics collected"

  - name: devops-deployment
    description: Inspect and manage application deployments
    access:
      - headerName: X-Auth-Request-Groups
        headerValue:
          - "groups:devops"
          - "groups:platform-admin"
    count: 1
    cmds:
      - echo "Deployment status RUNNING"
      - echo "Application version v2.4.1"

  - name: cloud-infrastructure
    description: Inspect cloud compute, networking and infrastructure state
    access:
      - headerName: X-Auth-Request-Groups
        headerValue:
          - "groups:cloud-infra"
          - "groups:platform-admin"
    count: 1
    cmds:
      - echo "Cloud infrastructure HEALTHY"
      - echo "Compute RUNNING"
      - echo "Network HEALTHY"

  - name: platform-admin
    description: Execute privileged platform operations
    access:
      - headerName: X-Auth-Request-Groups
        headerValue:
          - "groups:platform-admin"
    count: 1
    cmds:
      - echo "Privileged platform operation authorized"

```

All three tools only print text. Their names do not imply that they inspect or change real infrastructure.

This preserves the example's deliberately non-hierarchical policy:

| User's API permissions | `sre-incident-response` | `devops-deployment` | `cloud-infrastructure` | `platform-admin` |
|---|---|---|---|
| `groups:sre` | Allow | Deny | Deny | Deny |
| `groups:devops` | Deny | Allow | Deny | Deny |
| `groups:cloud-infra` | Deny | Deny | Allow | Deny |
| `groups:platform-admin` | Allow | Allow | Allow | Allow |
| None of these permissions | Deny | Deny | Deny | Deny |

If production operations should instead require administrators, change that tool's rule and the verification matrix together. A stack without an `access` block has no tool-level restriction. Every remotely exposed operational stack should declare its intended access.

With several access rules, `matchAccess: ANY` is the default; `ALL` requires every rule to match at least one allowed value. Matching is exact. An email-domain restriction needs a separately validated domain value or explicit full-address values, not an assumed wildcard match. Trust additional identity headers only if Nginx overwrites them from verified claims.

Set file ownership:

```bash
sudo chown root:seristack /etc/seristack/config.yaml
sudo chmod 0640 /etc/seristack/config.yaml
```

Create `/etc/systemd/system/seristack-mcp.service`:

```ini
[Unit]
Description=SeriStack MCP Server
Wants=network-online.target
After=network-online.target

[Service]
Type=simple
User=seristack
Group=seristack
WorkingDirectory=/var/lib/seristack
Environment=PATH=/usr/local/bin:/usr/bin:/bin
ExecStart=/usr/local/bin/seristack mcp \
  -t streamableHTTP \
  -a 127.0.0.1 \
  -p 8081 \
  -l 10 \
  -c /etc/seristack/config.yaml \
  --audit-log /var/log/seristack/mcp-audit.log \
  --exclude-identity-headers Authorization \
  --exclude-identity-headers Cookie
Restart=on-failure
RestartSec=5
UMask=0027
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/seristack /var/log/seristack

[Install]
WantedBy=multi-user.target
```

The service account's home directory is `/var/lib/seristack`. Existing CLI configuration in an administrator's home directory is not inherited. The systemd filesystem restrictions allow writes in the two declared directories and private temporary storage. Keep reviewed code outside those writable directories; add narrowly scoped writable paths only when a real tool needs them.

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now seristack-mcp
sudo systemctl status seristack-mcp --no-pager
```

Install any cloud CLIs, Python dependencies, Kubernetes configuration, and CI/CD credentials required by later tools for this service identity. Test their access as `seristack`, not only in your interactive administrator shell.

## 8. Obtain a certificate

Create a webroot for ACME challenges:

```bash
sudo install -d -o root -g root -m 0755 /var/www/letsencrypt/.well-known/acme-challenge
```

For initial certificate issuance, create `/etc/nginx/sites-available/seristack-mcp` with this temporary HTTP-only configuration. Replace the hostname:

```nginx
server {
    listen 80;
    listen [::]:80;
    server_name mcp.example.com;

    location ^~ /.well-known/acme-challenge/ {
        root /var/www/letsencrypt;
        default_type text/plain;
        try_files $uri =404;
    }

    location / {
        return 404;
    }
}
```

Activate the site, confirm DNS points to the VM, and request a certificate:

```bash
sudo ln -s /etc/nginx/sites-available/seristack-mcp /etc/nginx/sites-enabled/seristack-mcp
sudo nginx -t && sudo systemctl reload nginx
sudo certbot certonly --webroot -w /var/www/letsencrypt -d "$MCP_HOST"
```

If the site symlink already exists, reuse it. Resolve duplicate `server_name` entries instead of deleting unrelated sites. The certificate will be stored beneath `/etc/letsencrypt/live/<hostname>/`.

Configure the renewal hook at `/etc/letsencrypt/renewal-hooks/deploy/reload-nginx`:

```sh
#!/bin/sh
set -eu
/usr/sbin/nginx -t
/usr/bin/systemctl reload nginx
```

```bash
sudo chmod 0755 /etc/letsencrypt/renewal-hooks/deploy/reload-nginx
sudo systemctl enable --now certbot.timer
sudo certbot renew --dry-run
```

The timer command above is for the Ubuntu package installation used in this guide. With Snap, check its existing scheduled renewal mechanism instead. The final Nginx configuration retains the ACME location, so renewals continue to work.

If inbound port 80 is unavailable, use a Certbot DNS plugin for your DNS provider. Manual DNS challenges require manual renewal unless automated hooks are configured. [Certbot usage and renewal](https://eff-certbot.readthedocs.io/en/stable/using.html)

## 9. Configure Nginx for MCP authentication and discovery

Create `/etc/nginx/conf.d/seristack-mcp-log.conf`. Ubuntu loads this inside the `http` context:

```nginx
log_format seristack_mcp escape=json
    '{"time":"$time_iso8601","request_id":"$request_id",'
    '"method":"$request_method","path":"$uri","status":$status,'
    '"user":"$auth_user","groups":"$auth_groups"}';
```

Replace the temporary `/etc/nginx/sites-available/seristack-mcp` contents with the configuration below. Replace **every** `mcp.example.com` and `YOUR_TENANT.us.auth0.com` occurrence, including certificate paths and JSON strings.

```nginx
server {
    listen 80;
    listen [::]:80;
    server_name mcp.example.com;

    location ^~ /.well-known/acme-challenge/ {
        root /var/www/letsencrypt;
        default_type text/plain;
        try_files $uri =404;
    }

    location / {
        return 301 https://mcp.example.com$request_uri;
    }
}

server {
    listen 443 ssl;
    listen [::]:443 ssl;
    server_name mcp.example.com;

    ssl_certificate /etc/letsencrypt/live/mcp.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/mcp.example.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;

    access_log /var/log/nginx/seristackmcp_access.log seristack_mcp;
    error_log /var/log/nginx/seristackmcp_error.log;

    # Only Nginx subrequests can invoke this endpoint.
    location = /oauth2/auth {
        internal;
        proxy_pass http://127.0.0.1:4180/oauth2/auth;
        proxy_pass_request_body off;
        proxy_set_header Content-Length "";
        proxy_set_header Authorization $http_authorization;

        # MCP uses bearer authentication; ignore proxy login cookies.
        proxy_set_header Cookie "";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $remote_addr;
        proxy_set_header X-Forwarded-Proto https;
        proxy_set_header X-Forwarded-Host $host;
        proxy_set_header X-Forwarded-Uri $request_uri;
        proxy_set_header X-Request-ID $request_id;
    }

    location = /mcp {
        auth_request /oauth2/auth;
        error_page 401 = @mcp_oauth_required;

        auth_request_set $auth_user $upstream_http_x_auth_request_user;
        auth_request_set $auth_email $upstream_http_x_auth_request_email;
        auth_request_set $auth_groups $upstream_http_x_auth_request_groups;
        auth_request_set $auth_username $upstream_http_x_auth_request_preferred_username;

        # Replace client-supplied values with verified proxy output.
        proxy_set_header X-Auth-Request-User $auth_user;
        proxy_set_header X-Auth-Request-Email $auth_email;
        proxy_set_header X-Auth-Request-Groups $auth_groups;
        proxy_set_header X-Auth-Request-Preferred-Username $auth_username;

        # SeriStack authorizes through the verified identity headers.
        proxy_set_header Authorization "";
        proxy_set_header Cookie "";

        proxy_set_header X-Request-ID $request_id;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $remote_addr;
        proxy_set_header X-Forwarded-Proto https;
        proxy_set_header Host $host;

        proxy_http_version 1.1;
        proxy_set_header Connection "";
        proxy_buffering off;
        proxy_cache off;
        proxy_read_timeout 3600s;
        proxy_send_timeout 3600s;

        proxy_pass http://127.0.0.1:8081;
    }

    location = /.well-known/oauth-protected-resource/mcp {
        default_type application/json;
        return 200 '{
            "resource": "https://mcp.example.com/mcp",
            "authorization_servers": [
                "https://YOUR_TENANT.us.auth0.com/"
            ],
            "bearer_methods_supported": ["header"],
            "scopes_supported": [
                "openid",
                "profile",
                "email",
                "groups:sre",
                "groups:devops",
                "groups:cloud-infra",
                "groups:platform-admin",
                "offline_access"
            ]
        }';
    }

    location @mcp_oauth_required {
        default_type application/json;
        add_header Cache-Control "no-store" always;
        add_header WWW-Authenticate 'Bearer resource_metadata="https://mcp.example.com/.well-known/oauth-protected-resource/mcp"' always;
        return 401 '{"error":"unauthorized"}';
    }

    location = /healthz {
        access_log off;
        default_type text/plain;
        return 200 "OK\n";
    }

    location / {
        return 404;
    }
}
```

This exposes the exact `/mcp` endpoint. Configure clients without a trailing slash. `scopes_supported` advertises available scopes; it does not grant them. The verified token claims and stack rules govern access.

For this direct-to-VM topology, Nginx supplies the client address itself instead of trusting an incoming `X-Forwarded-For` chain. If you later add a load balancer, configure trusted real-IP sources explicitly.

Do not replace the MCP 401 response with a redirect to `/oauth2/sign_in`. MCP clients need the challenge and discovery document to start their own OAuth flow. Metadata is public; tool access is authenticated. `/healthz` checks Nginx only, not the health of Auth0 or SeriStack. [Nginx authentication subrequests](https://nginx.org/en/docs/http/ngx_http_auth_request_module.html), [MCP authorization](https://modelcontextprotocol.io/specification/2025-11-25/basic/authorization)

Apply the configuration:

```bash
sudo nginx -t && sudo systemctl reload nginx
```

The proxy's identity headers replace corresponding caller-supplied headers. Empty verified values are not forwarded. If you add an access rule based on another header later, add a corresponding verified mapping; an arbitrary incoming header is not a trusted identity attribute.

## 10. Verify the deployment

### 10.1 Check services and listeners

```bash
sudo systemctl is-active nginx oauth2-proxy seristack-mcp
sudo ss -ltnp | grep -E ':(80|443|4180|8081)\b'
```

Confirm that 4180 and 8081 listen on `127.0.0.1`, while public web traffic reaches Nginx. If a service fails:

```bash
sudo journalctl -u oauth2-proxy -u seristack-mcp -n 100 --no-pager
```

### 10.2 Check the authentication challenge

Run from your workstation or the VM:

```bash
curl -sS -i --max-time 15 \
  -H 'Accept: application/json, text/event-stream' \
  "https://$MCP_HOST/mcp"
```

Expected: HTTP **401**, a `WWW-Authenticate: Bearer resource_metadata="..."` header, and `{"error":"unauthorized"}`. A 302 login redirect is not the expected result.

Check discovery:

```bash
curl -fsS --max-time 15 \
  "https://$MCP_HOST/.well-known/oauth-protected-resource/mcp" | jq .

curl -fsS --max-time 15 \
  "https://$AUTH0_DOMAIN/.well-known/openid-configuration" \
  | jq '{issuer, authorization_endpoint, token_endpoint, jwks_uri, registration_endpoint, code_challenge_methods_supported, authorization_response_iss_parameter_supported, client_id_metadata_document_supported}'
```

Confirm that the resource matches the Auth0 API identifier and that the issuer matches the configured tenant. Discovery alone does not prove that registration, user login, or API access is authorized; complete a real client login next.

### 10.3 Check that an unauthenticated caller cannot supply a group

```bash
curl -sS -i --max-time 15 \
  -H 'Accept: application/json, text/event-stream' \
  -H 'X-Auth-Request-Groups: groups:platform' \
  "https://$MCP_HOST/mcp"
```

Expected: **401**. Also repeat an authenticated denied-tool test with a low-privilege user's token plus a forged groups header. It must still be denied according to the authenticated user's actual permissions. Use an authorized diagnostic client such as MCP Inspector for that test; never send real tokens to an unrelated echo service.

### 10.4 Check discovery and execution with separate users

Use the clients in the next section or [MCP Inspector](https://github.com/modelcontextprotocol/inspector). Initialize a new session for each user and check the matrix in section 7.

- A developer should see and successfully call `system-health`.
- A platform user should see and successfully call all three harmless examples.
- A user with only `platform-admin` should see `system-health` and `inspect-runtime`.
- A user with none of the sample permissions should see none of these tools.
- Submit `production-operation` directly as a developer and as a user with only `platform-admin`. Both must be denied, even when the name is known.
- Test a user with multiple roles to confirm the combined permissions behave as intended.

Check the MCP response body; an HTTP 200 can carry an application-level tool error. In the pinned release, a call to a filtered-out tool can return `tool not found`. That is a rejected call, and it can happen before the tool handler writes an execution audit event. Confirm successful executions in the audit log; do not assume every rejected RPC produces one. Do not automatically run every tool after replacing these examples with real operational commands.

### 10.5 Check token renewal and permission changes

Confirm that the client requests `offline_access`, that Auth0 allows it for the API and client, and that a subsequent request after access-token expiry succeeds through refresh or prompts for sign-in as expected. Enabling offline access does not guarantee that every client requests or receives a refresh token.

After changing a role, obtain a new token and repeat the relevant tests. Existing JWTs can retain their old permissions until they expire; disconnecting a UI session or changing a role is not an immediate revocation mechanism for an already-issued token validated locally.

Inspect activity without recording tokens:

```bash
sudo tail -n 30 /var/log/nginx/seristackmcp_access.log
sudo tail -n 30 /var/log/seristack/mcp-audit.log
```

The `X-Auth-Request-User` value is a better stable identifier than a presumed email. Depending on bearer-token claims, the proxy's email field can contain a subject identifier rather than a usable email address.

## 11. Connect users' clients

Give users the endpoint **`https://mcp.example.com/mcp`** and their account onboarding instructions. Client identifiers are not secrets; the proxy's confidential client secret stays on the server.

### 11.1 Cline in VS Code

Open **Cline → MCP Servers → Remote Servers**. Enter a name such as `seristack`, set the URL, choose **Streamable HTTP**, and add the server. Complete the authentication prompt in your browser. When VS Code asks to open an external site, verify that it is your Auth0 domain and choose **Open**.

Alternatively, open Cline's **Configure MCP Servers** JSON and merge this entry into `mcpServers`:

```json
{
  "mcpServers": {
    "seristack": {
      "type": "streamableHttp",
      "url": "https://mcp.example.com/mcp",
      "disabled": false,
      "autoApprove": []
    }
  }
}
```

Use Cline's OAuth sign-in action when prompted. Leave out a static `Authorization` header for this flow. This is Cline's configuration, not GitHub Copilot's `.vscode/mcp.json` format. [Cline MCP configuration](https://docs.cline.bot/mcp/mcp-overview)

### 11.2 Codex CLI and local desktop/IDE configuration

On the user's machine:

```bash
codex mcp add seristack --url https://mcp.example.com/mcp
codex mcp login seristack
codex mcp list
```

Or merge this into `~/.codex/config.toml`, then run `codex mcp login seristack`:

```toml
[mcp_servers.seristack]
url = "https://mcp.example.com/mcp"
```

In a desktop or IDE version with MCP settings, add a Streamable HTTP server with that URL and select **Authenticate**. No password or shared client secret belongs in the MCP configuration.

Current Codex can choose CIMD registration when advertised by the issuer. To explicitly test this guide's DCR path, supported CLI versions offer:

```bash
codex mcp login seristack --oauth-client-registration dcr
```

If your version does not recognize that option, check `codex mcp login --help` and update or use its supported registration flow. These local configuration instructions do not configure hosted ChatGPT web. [Official Codex MCP documentation](https://learn.chatgpt.com/docs/extend/mcp?surface=cli)

### 11.3 Claude Code

```bash
claude mcp add --transport http --scope user seristack https://mcp.example.com/mcp
```

Inside Claude Code, run `/mcp`, select the server, and authenticate. The `user` scope makes the registration available across that user's projects. Check that the expected tools appear and call `system-health`. [Claude Code MCP documentation](https://code.claude.com/docs/en/mcp)

### 11.4 Claude Desktop and Claude web

Where custom remote connectors are available, open **Settings → Connectors**, add a custom connector with the MCP URL, and complete Auth0 sign-in. Organization policy may require an administrator to enable the connector first. For this hosted HTTP endpoint, use the remote connector flow rather than assuming the local-process `claude_desktop_config.json` format accepts a URL directly. [Claude remote connector instructions](https://support.claude.com/en/articles/11175166-get-started-with-custom-connectors-using-remote-mcp)

### Why sign-in sometimes goes straight to consent

An existing Auth0 browser session can authenticate the user through SSO, so the next screen may be consent rather than a password prompt. That is normal. Use a separate browser profile or an appropriate IdP sign-out flow when testing another identity. Clearing a client's saved tokens is not the same as ending the browser's Auth0 session.

## 12. Operate the service

### 12.1 Logs and rotation

The Nginx access format records the verified user and groups, request ID, path, and status without a bearer token or query string. SeriStack's audit log records handler execution activity. Other incoming headers in an audit record remain caller-supplied data unless explicitly mapped from verified identity. Outputs and arguments can also contain sensitive information; review tool output before sending it to an AI client or log system.

Create `/etc/logrotate.d/seristack-mcp`:

```conf
/var/log/seristack/mcp-audit.log {
    daily
    maxsize 20M
    rotate 14
    missingok
    notifempty
    compress
    delaycompress
    copytruncate
    su seristack seristack
}
```

Validate without rotating:

```bash
sudo logrotate --debug /etc/logrotate.d/seristack-mcp
```

`copytruncate` avoids assuming that SeriStack reopens its log on a signal, but it has a small copy/truncate race in which entries can be lost. For durable security auditing, forward logs to a protected central destination and use a logging strategy appropriate to your retention requirements. A service-writable local log is not tamper-proof. Confirm that the distribution's existing Nginx rotation covers `/var/log/nginx/*.log`; do not add duplicate rotation rules for the same files.

Monitor certificate expiry, failed authentication, service restarts, disk space, and tool errors. Include an authenticated synthetic call to a harmless tool: Nginx `/healthz` alone cannot confirm the whole execution path.

### 12.2 DCR registrations

Different users, machines, or client profiles may create different Auth0 registrations. Keep registrations while their clients use them. Before deleting one, identify its owner and confirm it is unused; deleting an active registration can break login or refresh.

Monitor your plan's current application limit. Auth0's published limits distinguish Free from paid Self-service plans; do not assume that all tenants allow 100 applications or that a warning will arrive before the limit. Evaluate registration restrictions against each client because hosted connectors may register from their provider's infrastructure. [Auth0 entity limits](https://auth0.com/docs/troubleshoot/customer-support/operational-policies/entity-limit-policy)

### 12.3 Role changes and offboarding

After granting or changing access, ask the client to re-authenticate and verify a fresh token. For offboarding, remove relevant API permissions, block the user where appropriate, and revoke their refresh/session credentials using the tenant's controls. Already-issued JWT access tokens can remain valid until expiry at this locally validating proxy. If immediate revocation is required, add an enforcement mechanism that checks revocation or a deny list on requests; do not promise immediate revocation from a role edit alone.

Keep privileged roles deliberate. Granting an AI client a tool exposes that tool's operational authority through the client, subject to its own approval behavior and the server's constraints.

### 12.4 Secret rotation

Rotate the Regular Web Application secret in Auth0, update the root-only proxy environment file, and restart oauth2-proxy. Coordinate the change with any other service using that same registration. Never share the replacement secret with users.

To rotate the cookie secret, generate a fresh value using the earlier command and **replace** the existing setting. Cookie rotation invalidates proxy cookies; it does not revoke Auth0 access tokens. The bearer-only MCP path does not rely on these cookies.

```bash
sudo systemctl restart oauth2-proxy
sudo systemctl status oauth2-proxy --no-pager
```

Rotate any secret that was exposed in a transcript, screenshot, repository, or terminal capture. Excluding Authorization and Cookie from audit records does not protect unrelated credentials printed by stack commands.

### 12.5 Upgrade and rollback

Before changing binaries or configuration, create a root-only backup:

```bash
SERISTACK_BACKUP_DIR="/var/backups/seristack-mcp/$(date -u +%Y%m%dT%H%M%SZ)"
sudo install -d -o root -g root -m 0700 "$SERISTACK_BACKUP_DIR"
sudo cp -a /usr/local/bin/seristack "$SERISTACK_BACKUP_DIR/seristack"
sudo cp -a /usr/local/bin/oauth2-proxy "$SERISTACK_BACKUP_DIR/oauth2-proxy"
sudo cp -a /etc/seristack/config.yaml "$SERISTACK_BACKUP_DIR/config.yaml"
sudo cp -a /etc/oauth2-proxy/oauth2-proxy.env "$SERISTACK_BACKUP_DIR/oauth2-proxy.env"
sudo cp -a /etc/systemd/system/seristack-mcp.service "$SERISTACK_BACKUP_DIR/seristack-mcp.service"
sudo cp -a /etc/systemd/system/oauth2-proxy.service "$SERISTACK_BACKUP_DIR/oauth2-proxy.service"
sudo cp -a /etc/nginx/sites-available/seristack-mcp "$SERISTACK_BACKUP_DIR/nginx-site"
sudo cp -a /etc/nginx/conf.d/seristack-mcp-log.conf "$SERISTACK_BACKUP_DIR/nginx-log.conf"
printf 'Backup: %s\n' "$SERISTACK_BACKUP_DIR"
```

Validate a new release in staging using the allow/deny matrix. Stop the affected service before replacing its running binary, install the verified release, and restart it. A SeriStack restart interrupts in-memory MCP sessions, so clients may need to reconnect. Run `systemctl daemon-reload` when unit files change, and `nginx -t` before reloading Nginx.

For rollback, set `SERISTACK_BACKUP_DIR` to the reviewed backup you intend to restore. This restores server-side files, not Auth0 tenant changes:

```bash
sudo systemctl stop seristack-mcp oauth2-proxy
sudo install -o root -g root -m 0755 "$SERISTACK_BACKUP_DIR/seristack" /usr/local/bin/seristack
sudo install -o root -g root -m 0755 "$SERISTACK_BACKUP_DIR/oauth2-proxy" /usr/local/bin/oauth2-proxy
sudo install -o root -g seristack -m 0640 "$SERISTACK_BACKUP_DIR/config.yaml" /etc/seristack/config.yaml
sudo install -o root -g root -m 0600 "$SERISTACK_BACKUP_DIR/oauth2-proxy.env" /etc/oauth2-proxy/oauth2-proxy.env
sudo install -o root -g root -m 0644 "$SERISTACK_BACKUP_DIR/seristack-mcp.service" /etc/systemd/system/seristack-mcp.service
sudo install -o root -g root -m 0644 "$SERISTACK_BACKUP_DIR/oauth2-proxy.service" /etc/systemd/system/oauth2-proxy.service
sudo install -o root -g root -m 0644 "$SERISTACK_BACKUP_DIR/nginx-site" /etc/nginx/sites-available/seristack-mcp
sudo install -o root -g root -m 0644 "$SERISTACK_BACKUP_DIR/nginx-log.conf" /etc/nginx/conf.d/seristack-mcp-log.conf
sudo systemctl daemon-reload
sudo systemctl start oauth2-proxy seristack-mcp
sudo nginx -t && sudo systemctl reload nginx
```

Repeat section 10 after an upgrade or rollback. If the Auth0 client secret was rotated after the backup, update the restored environment file with the active secret before starting the proxy.

## 13. Expand into operational tools

Once authentication and authorization are verified, replace or extend the harmless examples with reviewed commands:

| Team | Useful first tools | Execution prerequisites |
|---|---|---|
| Cloud | Instance inventory, storage inspection, network configuration summaries | Cloud CLI/SDK, an attached workload identity, and scoped IAM permissions |
| SRE | Recent alerts, error-rate queries, bounded log searches | Monitoring endpoints, credentials, query limits, and output redaction |
| Platform | Kubernetes rollout status, cluster events, Terraform plan summaries | Explicit kubeconfig/context, namespace access, reviewed IaC, and state access |
| DevOps | Pipeline status, failed-job summaries, deployment history | CI/CD API access scoped to the intended organizations and repositories |

Start with observational tools. For changes, expose specific actions with constrained resource identifiers and server-side validation. Put critical approval requirements in an enforceable workflow; a model-facing description or a client approval dialog is not a server-side authorization rule. Avoid generic remote-shell tools that accept arbitrary command strings.

Keep separate credentials or instances for environments that need different trust boundaries. Set explicit project, account, cluster, region, namespace, and repository targets rather than inheriting an administrator's defaults. Protect Terraform code and command scripts as well as the YAML that invokes them.

The service sets `-l 10`, the concurrent shell-execution limit in the pinned CLI. It applies to work scheduled through SeriStack's executor, including Python-backed execution. It is not a container sandbox, a rate limit, or a limit on child processes, threads, or cloud API requests created inside a script. Set tool timeouts, output limits, and downstream rate limits separately as needed.

If a tool needs Python, install a controlled Python runtime and dependencies and use the supported interpreter configuration for your pinned SeriStack version. Do not assume that choosing Python adds Bash or mvdan semantics. Validate workflow/dependency behavior specifically through MCP before documenting it as a supported remote workflow.

## 14. Troubleshooting

| Symptom | Check |
|---|---|
| `/mcp` returns 302 | Replace the browser sign-in redirect with the MCP 401 metadata challenge. |
| `/mcp/` returns 404 | Use the exact `/mcp` URL configured in this guide. |
| Auth0 says “Unauthorized or unknown client” | Verify the client ID, tenant, registration method, and applicable grants. Watch for copied `0` versus `O` characters. |
| Auth0 says the client cannot access the resource server | Check API Application Access and third-party user-delegated defaults. |
| Auth0 says “Service not found” | Compare the requested resource with the API identifier, including path and trailing slash; check Resource Parameter Compatibility Profile. |
| Login screen offers no usable connection | Promote the intended connection to domain level for third-party clients. |
| Registration fails | Inspect the actual client/tenant error, DCR policy, and application quota. A discovery endpoint alone does not prove registration is allowed. |
| Codex uses an unexpected registration flow | Check issuer-advertised CIMD support and the client's registration options. |
| 401 after sign-in | Check token issuer, expiry, API audience, proxy configuration, and oauth2-proxy logs. |
| No tools appear | Confirm permissions on the correct API, the user's effective permissions, and the verified groups header. |
| Tool call says “tool not found” | It may have been filtered by access rules; compare the authenticated user's permissions with the tool definition. |
| A role change has no effect | Obtain a fresh access token and reconnect; old JWTs can retain earlier claims. |
| Cookie-secret startup error | Regenerate a 32-byte URL-safe secret and retain only one setting in the environment file. |
| Nginx returns 500 during authentication | Check oauth2-proxy reachability and subrequest errors; `auth_request` expects a successful or explicit denied response. |
| Nginx returns 502 for a verified request | Check the SeriStack process, port, configuration, and service logs. |
| Cloud or Kubernetes commands work over SSH but fail through MCP | Check the service user's identity, paths, filesystem restrictions, CLI credentials, and explicit resource context. |
| Replies stall | Check streaming configuration, proxy timeouts, client tool timeout, and the underlying command. |
| Certificate renewal fails | Check DNS, port 80, the ACME webroot location, the timer, and the renewal logs. |

Decode JWT claims locally only when needed. Decoding is diagnostic and does not verify the signature. Do not paste bearer tokens into third-party token inspectors or support screenshots.

## 15. Deployment acceptance

Before giving the endpoint to the team, confirm:

- [ ] The endpoint, protected-resource metadata, Auth0 API identifier, and accepted audience use the same `/mcp` URL.
- [ ] Auth0 registration, the intended login connection, RBAC, permissions, and user onboarding work.
- [ ] Users connect through browser OAuth without receiving a shared client secret.
- [ ] Ports 4180 and 8081 are loopback-only; the VM has no untrusted local users or workloads.
- [ ] Nginx replaces each identity header used for authorization and returns the expected 401 challenge.
- [ ] The sample permission matrix passes both discovery and direct-call tests, including forged-header attempts.
- [ ] Configuration, parent directories, and invoked scripts cannot be replaced by the service user.
- [ ] The actual service identity has only the required downstream permissions.
- [ ] Refresh behavior, role changes, and the limits of JWT revocation are understood.
- [ ] Certificate renewal, monitoring, log handling, backup, and rollback have been exercised.

## Reference links

- [SeriStack source and documentation](https://github.com/TechXploreLabs/seristack)
- [SeriStack v0.4.6 release](https://github.com/TechXploreLabs/seristack/releases/tag/v0.4.6)
- [oauth2-proxy configuration](https://oauth2-proxy.github.io/oauth2-proxy/configuration/overview/)
- [Auth0 authorization for MCP](https://auth0.com/ai/docs/mcp/get-started/authorization-for-your-mcp-server)
- [MCP authorization specification](https://modelcontextprotocol.io/specification/2025-11-25/basic/authorization)
- [Cline MCP](https://docs.cline.bot/mcp/mcp-overview)
- [Codex MCP](https://learn.chatgpt.com/docs/extend/mcp?surface=cli)
- [Claude Code MCP](https://code.claude.com/docs/en/mcp)
- [Claude remote connectors](https://support.claude.com/en/articles/11175166-get-started-with-custom-connectors-using-remote-mcp)
