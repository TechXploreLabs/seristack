# L1 Oracle Cloud Engineer

Daily grind: Oracle Cloud (OCI) has its own ecosystem — Autonomous Databases, OCI CLI, compartments, policies, Object Storage, and a lot of manual console clicking that should never need to happen.
The pain is specific: OCI tasks are well-defined and repetitive, but the OCI CLI commands are verbose and easy to get wrong in order.

```yaml
#config.yaml
stacks:
  - name: check-compartment-health
    description: Check resource quotas and limits across compartments
    count: 1
    vars:
      compartment_id: ocid1.compartment.oc1..your-compartment-id
    cmds:
      - |
        set -e
        echo "Checking compute limits..."
        oci limits value list \
          --compartment-id={{.Vars.compartment_id}} \
          --service-name compute \
          --output table

        echo "Checking block storage usage..."
        oci bv volume list \
          --compartment-id={{.Vars.compartment_id}} \
          --output table

  - name: backup-autonomous-db
    description: Trigger manual backup of Autonomous Database
    count: 1
    dependsOn: [check-compartment-health]
    vars:
      db_id: ocid1.autonomousdatabase.oc1..your-db-id
    cmds:
      - |
      set -e
        echo "Triggering ADB backup..."
        oci db autonomous-database-backup create \
          --autonomous-database-id={{.Vars.db_id}} \
          --display-name="manual-backup-$(date +%F)"

  - name: check-backup-status
    description: Poll until backup completes
    count: 1
    dependsOn: [backup-autonomous-db]
    cmds:
      - |
        set -e
        echo "Waiting for backup to complete..."
        sleep 30
        oci db autonomous-database-backup list \
          --autonomous-database-id={{.Vars.db_id}} \
          --output table | grep manual-backup

  - name: rotate-object-storage-keys
    description: Rotate customer-managed keys for Object Storage buckets
    count: 1
    dependsOn: [check-backup-status]
    vars:
      namespace: your-namespace
      bucket: prod-data-bucket
    cmds:
      - |
        set -e
        echo "Rotating Object Storage bucket keys..."
        oci os bucket update \
          --namespace={{.Vars.namespace}} \
          --name={{.Vars.bucket}} \
          --kms-key-id=$OCI_KMS_KEY_ID

  - name: audit-iam-policies
    description: List all IAM policies and flag overly permissive ones
    count: 1
    cmds:
      - |
        echo "Auditing IAM policies..."
        oci iam policy list \
          --compartment-id={{.Vars.compartment_id}} \
          --output table
        python scripts/flag_permissive_policies.py

  - name: cost-report
    description: Pull daily OCI cost and usage report
    count: 1
    cmds:
      - |
        oci usage-api usage-summary request-summarized-usages \
          --tenant-id=$OCI_TENANT_ID \
          --time-usage-started=$(date -d yesterday +%FT00:00:00Z) \
          --time-usage-ended=$(date +%FT00:00:00Z) \
          --granularity DAILY \
          --output table

  - name: notify-team
    count: 1
    dependsOn: [cost-report]
    cmds:
      - |
        curl -X POST $SLACK_WEBHOOK \
          --data '{"text":"☁️ OCI daily ops complete. Backup done, keys rotated, IAM audited."}'

server:
  host: 0.0.0.0
  port: 8080
  endpoints:
    - path: /daily-oci-ops
      method: GET
      stackName: check-compartment-health

    - path: /backup-db
      method: GET
      stackName: backup-autonomous-db

    - path: /cost-report
      method: GET
      stackName: cost-report
```

```bash
seristack run      # expose all endpoints
seristack trigger  # run full daily ops pipeline
seristack mcp -t streamableHTTP # start mcp server and add stack as tools which has description.
```

