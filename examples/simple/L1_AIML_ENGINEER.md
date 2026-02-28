# L1 AI/ML Engineer

Daily grind: Kick off training runs, monitor GPU usage, evaluate model metrics, push models to registry, notify the team.
The pipeline between "new data arrived" and "model is in staging" is always the same — but it's fragile when done manually.

```yaml
stacks:
  - name: preprocess-data
    description: Validate and preprocess new training data
    count: 1
    vars:
      dataset: s3://your-bucket/datasets/latest
    cmds:
      - python scripts/preprocess.py --input={{.Vars.dataset}} --output=./data/processed

  - name: train-model
    description: Launch training job on GPU instance
    count: 1
    dependsOn: [preprocess-data]
    cmds:
      - python train.py --config=configs/prod.yaml --data=./data/processed

  - name: evaluate-model
    description: Run evaluation metrics on trained model
    count: 1
    dependsOn: [train-model]
    cmds:
      - python evaluate.py --threshold=0.85 || exit 1   # fails pipeline if below threshold

  - name: push-to-registry
    description: Push model to MLflow registry if evaluation passes
    count: 1
    dependsOn: [evaluate-model]
    cmds:
      - python scripts/register_model.py --stage=staging

  - name: notify-team
    count: 1
    dependsOn: [push-to-registry]
    cmds:
      - curl -X POST $SLACK_WEBHOOK --data '{"text":"🤖 New model passed eval and is in staging."}'
```

Your ML pipeline shouldn't live in a Jupyter notebook someone has to manually run top-to-bottom. Make it triggerable, version it, and let evaluation be the gatekeeper — not a human.
