# L1 Cloud Engineer

Daily grind: Provision resources, check costs, rotate credentials, clean up stale infrastructure, run compliance checks.

The kind of work that's "easy but you have to remember to do it."

```yaml
stacks:
  - name: cost-report
    description: Pull daily AWS cost summary
    count: 1
    cmds:
      - |
        aws ce get-cost-and-usage \
          --time-period Start=$(date -d yesterday +%F),End=$(date +%F) \
          --granularity DAILY \
          --metrics BlendedCost

  - name: stale-resource-cleanup
    description: Delete unattached EBS volumes older than 30 days
    count: 1
    cmds:
      - python scripts/cleanup_stale_volumes.py --dry-run=false

  - name: rotate-secrets
    description: Rotate IAM access keys expiring this week
    count: 1
    dependsOn: [stale-resource-cleanup]
    cmds:
      - python scripts/rotate_expiring_keys.py

  - name: compliance-check
    description: Run AWS Config compliance rules check
    count: 1
    dependsOn: [rotate-secrets]
    cmds:
      - aws configservice describe-compliance-by-config-rule --compliance-types NON_COMPLIANT

server:
  host: 0.0.0.0
  port: 8080
  endpoints:
    - path: /daily-ops
      method: GET
      stackName: cost-report
```