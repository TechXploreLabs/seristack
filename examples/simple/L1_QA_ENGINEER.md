# L1 QA Engineer

Daily grind: Set up test environments, run regression suites, reset test data, generate reports, log results.

These are the same 6 steps before every test cycle. Manually. Every time.

```yaml
stacks:
  - name: reset-test-data
    count: 1
    cmds:
      - npm run db:reset && npm run db:seed:test

  - name: run-regression
    count: 1
    dependsOn: [reset-test-data]
    cmds:
      - npm run test:regression -- --env=staging

  - name: generate-report
    count: 1
    dependsOn: [run-regression]
    cmds:
      - npm run test:report && cp report.html /reports/$(date +%F)-regression.html

  - name: notify-team
    count: 1
    dependsOn: [generate-report]
    cmds:
      - curl -X POST $SLACK_WEBHOOK --data '{"text":"🧪 Regression complete. Report ready."}'
```

```bash
seristack trigger   # full QA cycle, one command
```