# Seristack mcp stdio server

```yaml
#config.yaml
stacks:
  - name: sre-incident-response
    description: Diagnose production health and collect incident diagnostics
    # access:
    #   - headerName: X-Auth-Request-Groups
    #     headerValue:
    #       - "groups:sre"
    #       - "groups:platform-admin"
    count: 1
    vars:
    - name: key
      value: "true"
      allowed_value: ["true", "false"]
      required: true
    cmds:
      - echo "Production health check HEALTHY {{.Vars.key}}"
      - echo "Incident diagnostics collected"

  - name: devops-deployment
    description: Inspect and manage application deployments
    # access:
    #   - headerName: X-Auth-Request-Groups
    #     headerValue:
    #       - "groups:devops"
    #       - "groups:platform-admin"
    count: 1
    cmds:
      - echo "Deployment status RUNNING"
      - echo "Application version v2.4.1"

  - name: cloud-infrastructure
    description: Inspect cloud compute, networking and infrastructure state
    # access:
    #   - headerName: X-Auth-Request-Groups
    #     headerValue:
    #       - "groups:cloud-infra"
    #       - "groups:platform-admin"
    count: 1
    cmds:
      - echo "Cloud infrastructure HEALTHY"
      - echo "Compute RUNNING"
      - echo "Network HEALTHY"

  - name: platform-admin
    description: Execute privileged platform operations
    # access:
    #   - headerName: X-Auth-Request-Groups
    #     headerValue:
    #       - "groups:platform-admin"
    count: 1
    cmds:
      - echo "Privileged platform operation authorized"
```

to test the above config.yaml, run below cmd

```bash
seristack.exe trigger -c seristack-config.yaml
```

Add below config in the cline_mcp_settings.json

```json
{
  "mcpServers": {
    "seristack-stdio": {
      "autoApprove": [],
      "disabled": true,
      "timeout": 60,
      "type": "stdio",
      "command": "seristack",
      "args": [
        "mcp",
        "-t",
        "stdio",
        "-c",
        "E:\\vscode\\seristack\\seristack-config.yaml"
      ]
    },
}
```

