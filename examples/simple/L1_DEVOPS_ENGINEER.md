
# L1 DevOps Engineer

Your Deployment Runbook

Every DevOps and cloud engineer has a runbook — a doc, a Notion page, a sticky note — that says something like:

"Before deploying: check cluster health, scale up nodes, run smoke tests, deploy, verify pods, notify Slack, scale back down."

It's always partially out of date. It's always in someone's head. And at 2am during an incident, that's a problem.

seristack turns that runbook into executable YAML:
```yaml
#config.yaml
stacks:
  - name: check-cluster-health
    description: Verify all nodes are healthy before deployment
    count: 1
    cmds:
      - |
        set -e
        echo "Checking node health..."
        kubectl get nodes
        kubectl get pods --all-namespaces | grep -v Running | grep -v Completed

  - name: scale-up
    description: Scale up node group before deployment
    count: 1
    dependsOn: [check-cluster-health]
    cmds:
      - |
        aws eks update-nodegroup-config \
          --cluster-name prod-cluster \
          --nodegroup-name main \
          --scaling-config minSize=4,maxSize=10,desiredSize=6

  - name: run-smoke-tests
    description: Run pre-deployment smoke tests against staging
    count: 1
    dependsOn: [scale-up]
    cmds:
      - |
        set -e
        echo "Running smoke tests..."
        ./scripts/smoke-test.sh --env=staging

  - name: deploy
    description: Deploy latest image to production cluster
    count: 1
    dependsOn: [run-smoke-tests]
    vars:
      image_tag: latest
    cmds:
      - |
        set -e
        kubectl set image deployment/api \
          api=your-registry/api:{{.Vars.image_tag}} \
          --namespace=production
        kubectl rollout status deployment/api --namespace=production

  - name: verify-deployment
    description: Confirm pods are healthy post-deploy
    count: 1
    dependsOn: [deploy]
    cmds:
      - |
        set -e
        echo "Verifying deployment..."
        kubectl get pods --namespace=production
        curl -sf https://api.yourapp.com/health || exit 1

  - name: notify-slack
    description: Notify team of successful deployment
    count: 1
    dependsOn: [verify-deployment]
    cmds:
      - |
        curl -X POST $SLACK_WEBHOOK_URL \
          -H 'Content-type: application/json' \
          --data '{"text":"✅ Production deployment complete. All pods healthy."}'

  - name: scale-down
    description: Return cluster to normal size after deployment
    count: 1
    dependsOn: [notify-slack]
    cmds:
      - |
        aws eks update-nodegroup-config \
          --cluster-name prod-cluster \
          --nodegroup-name main \
          --scaling-config minSize=2,maxSize=6,desiredSize=3

server:
  host: 0.0.0.0
  port: 8080
  endpoints:
    - path: /deploy
      method: GET
      stackName: deploy
```
# Three Ways This Pays Off:

1. CLI — for engineers doing it themselves

```bash
seristack trigger -s check-cluster-health    # just validate
seristack trigger                            # full deployment pipeline
```
The dependsOn chain means if smoke tests fail, the deploy never runs. No more "oops I skipped that step."

2. HTTP — for triggering from CI/CD pipelines

Start the server — on localhost for internal use, or bind to 0.0.0.0 to expose it externally:

```bash
seristack run      # start the http server 
```
This spins up your configured endpoints. Now from anywhere — a CI pipeline, a webhook, a teammate's browser — you just curl it:

```bash
# Run the full deployment pipeline
curl http://your-deploy-server:8080/deploy

# Or trigger everything
curl http://your-deploy-server:8080/trigger
```

Inside your GitHub Actions or GitLab CI, instead of duplicating shell logic across pipelines:

```yaml
# .github/workflows/deploy.yml
- name: Trigger deployment
  run: curl http://your-deploy-server:8080/deploy
```

One source of truth for deployment logic. Your CI just calls it. No more copy-pasted bash across 6 workflow files that slowly drift apart.

3. MCP — for AI-assisted incident response

When something breaks at 2am and you're using an AI-assisted terminal or IDE, you can just say:

```yaml
"Check the cluster health and show me what's wrong"

"Run the deployment verification steps"
```

The AI calls the right stacks as tools, with full output — without you remembering flags and commands while half asleep.