# seristack

**Run shell workflows via CLI or HTTP — no CI, no cron, no Makefiles.**

Seristack is a lightweight automation engine designed to bridge the gap between local task execution and remote triggers. Define your stacks in YAML, manage dependencies, and expose your automation via a built-in HTTP server.

# Features

    🚀 Run multiple command stacks from a single config
    🔁 Repeat stacks with serial or concurrent execution
    🔗 Define dependencies between stacks
    🧩 Variable substitution using templates
    📦 Share output between stacks
    🌐 Expose stacks as HTTP endpoints
    🛠 Works with Bash, sh, and PowerShell

## Installation

### Using Homebrew (Mac and linux)

```bash
brew install TechXploreLabs/tap/seristack
```

# Sample stack yaml file

```yaml
# description about seristack

# `seristack trigger -c config.yaml`  will run the entire stack.
# `seristack trigger -c config.yaml -s stack3` will run the particular stack
# `seristack run -c config.yaml` will init the http server with endpoint. ctrl+c will stop the server process.

stacks:
- name: stack1                # name of the stack(REQUIRED)
  workDir: ./                 # working directory to execute the cmds. default is "./"
  continueOnError: false      # if the cmds has any error, setting this flag true will not stop entir stacks execution, seeting false will stop all other rest of the executions. default is false.
  count: 3                    # if setting count = 0 , will not run cmds, count = 3 will run entire cmds three times. default is 0.
  isSerial: TRUE              # if the count = 3 and isSerial = false will let run set of cmds concurrently(thrice). default is false.
  vars:                       # vars take key=value pair of variables. default is empty
    samplekey: samplevalue
  shell: bash                 # shell takes sh as default for linux, darwin and powershell for windows.
  shellArg: -c                # shellArg take -c as default for linux, darwin and -Command for windows.
  dependsOn: []               # dependsOn takes list of stack to start after them. default is [].
  cmds:                       # cmds takes list of shell cmds(linux, powershell)
  - |
    export samplekey={{.Vars.samplekey}}    # to use vars for substitution
    echo $samplekey
    echo "count={{.Count.index}}"         # index of count iterations
    echo "Hey i'm seristack an modern task automation tool."
- name: stack2
  workDir: ./
  continueOnError: false
  count: 3
  isSerial: TRUE
  dependsOn: [stack1]
  cmds:
  - |
    echo "Hey i can also take output from any of the previous stack, which completed before i start"
    echo "count={{.Result.stack1}}"     # to use result of previous batch stack output for substitution
- name: stack3
  workDir: ./
  count: 1
  vars:
    invite: hello engineers
  cmds:
  - |
    echo "Hey, i'm gonna show the date we met!!"
    echo "`date`"


server:
  host: 127.0.0.1      # default is 27.0.0.1 
  port: 8080           # default is 8080
  endpoint:            # endpoint will connect the path to particular stack and run the cmds publish output.
  - path: /show
    method: GET
    stackName: stack3

```

# License

Apache License
