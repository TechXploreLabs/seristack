package shellexecutor

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/TechXploreLabs/seristack/internal/config"
	"github.com/TechXploreLabs/seristack/internal/function"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

var (
	semaphoreMu sync.RWMutex
	semaphoreCh = make(chan struct{}, 10)
)

func SetConcurrencyLimit(limit int) {
	if limit <= 0 {
		limit = 1
	}
	semaphoreMu.Lock()
	semaphoreCh = make(chan struct{}, limit)
	semaphoreMu.Unlock()
}

func acquire() {
	semaphoreMu.RLock()
	ch := semaphoreCh
	semaphoreMu.RUnlock()
	ch <- struct{}{}
}

func release() {
	semaphoreMu.RLock()
	ch := semaphoreCh
	semaphoreMu.RUnlock()
	<-ch
}

func ExecuteShell(e *config.Executor, stack *config.Stack) *config.Result {
	result := &config.Result{
		Name:    stack.Name,
		Success: true,
	}
	shell, shellArg, err := Shellargs(stack.Shell, stack.ShellArg)
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		return result
	}
	if stack.ExecutionMode == "" {
		stack.ExecutionMode = "PARALLEL"
	}
	ExecutionMode := []string{"PARALLEL", "PIPELINE", "STAGE", "SEQUENTIAL"}
	if !slices.Contains(ExecutionMode, stack.ExecutionMode) {
		result.Error = fmt.Sprintf("invalid executionMode: %s", stack.ExecutionMode)
		result.Success = false
		return result
	}
	commandTimeout, err := commandTimeoutDuration(stack.Timeouts)
	if err != nil {
		result.Error = err.Error()
		result.Success = false
		return result
	}
	workDir := filepath.Join(e.SourceDir, stack.WorkDir)
	var wg sync.WaitGroup
	var outputMu sync.Mutex
	var allOutput bytes.Buffer
	var errorMsg bytes.Buffer
	countConcurrent := stack.ExecutionMode == "PARALLEL" || stack.ExecutionMode == "STAGE"
	commandsConcurrent := stack.ExecutionMode == "PARALLEL" || stack.ExecutionMode == "PIPELINE"
	for index := range stack.Count {
		executeIteration := func(idx int) error {
			if commandsConcurrent {
				var cmdWg sync.WaitGroup
				var iterationErrors []error
				var errorMu sync.Mutex
				for cmdIndex, cmd := range stack.Cmds {
					cmdWg.Add(1)
					go func(cIdx int, command string) {
						defer cmdWg.Done()
						indexStr := strconv.Itoa(idx)
						modifiedCmd := strings.ReplaceAll(command, "{{.Count.index}}", indexStr)
						modifiedCmd = strings.ReplaceAll(modifiedCmd, "{{.Self.result}}", allOutput.String())
						var replaceErr error
						if e.Registry != nil {
							modifiedCmd, replaceErr = function.ReplaceVariables(stack.Vars, e, modifiedCmd)
						} else {
							modifiedCmd, replaceErr = function.ReplaceVariables(stack.Vars, nil, modifiedCmd)
						}
						if replaceErr != nil {
							cmdErr := fmt.Errorf("variable template error : %w", replaceErr)
							errorMu.Lock()
							errorMsg.Write([]byte(cmdErr.Error()))
							iterationErrors = append(iterationErrors, cmdErr)
							errorMu.Unlock()
							return
						}
						output, execErr := ShellExec(workDir, shell, shellArg, modifiedCmd, commandTimeout.String())
						outputMu.Lock()
						if len(output) > 0 {
							allOutput.Write(output)
						}
						outputMu.Unlock()
						if execErr != nil {
							cmdErr := fmt.Errorf("command failed : %w\n%s", execErr, output)
							errorMu.Lock()
							errorMsg.Write([]byte(cmdErr.Error()))
							errorMu.Unlock()
							if !stack.ContinueOnError {
								errorMu.Lock()
								iterationErrors = append(iterationErrors, cmdErr)
								errorMu.Unlock()
							}
						}
					}(cmdIndex, cmd)
				}
				cmdWg.Wait()
				if len(iterationErrors) > 0 && !stack.ContinueOnError {
					return iterationErrors[0]
				}
			} else {
				for _, cmd := range stack.Cmds {
					indexStr := strconv.Itoa(idx)
					modifiedCmd := strings.ReplaceAll(cmd, "{{.Count.index}}", indexStr)
					modifiedCmd = strings.ReplaceAll(modifiedCmd, "{{.Self.result}}", allOutput.String())
					var replaceErr error
					if e.Registry != nil {
						modifiedCmd, replaceErr = function.ReplaceVariables(stack.Vars, e, modifiedCmd)
					} else {
						modifiedCmd, replaceErr = function.ReplaceVariables(stack.Vars, nil, modifiedCmd)
					}
					if replaceErr != nil {
						cmdErr := fmt.Errorf("variable template error : %w", replaceErr)
						errorMsg.Write([]byte(cmdErr.Error()))
						return cmdErr
					}
					output, execErr := ShellExec(workDir, shell, shellArg, modifiedCmd, commandTimeout.String())
					outputMu.Lock()
					if len(output) > 0 {
						allOutput.Write(output)
					}
					outputMu.Unlock()
					if execErr != nil {
						cmdErr := fmt.Errorf("command failed : %w\n%s", execErr, output)
						errorMsg.Write([]byte(cmdErr.Error()))
						if !stack.ContinueOnError {
							return cmdErr
						}
					}
				}
			}
			return nil
		}
		if countConcurrent {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				executeIteration(idx)
			}(index)
		} else {
			if err := executeIteration(index); err != nil {
				break
			}
		}
	}
	if countConcurrent {
		wg.Wait()
	}
	if stack.Count == 0 {
		result.Output = fmt.Sprintf("%s skipped", stack.Name)
	} else {
		if stack.Output != "" {
			modifiedCmd := strings.ReplaceAll(stack.Output, "{{.Self.result}}", allOutput.String())
			var replaceErr error
			if e.Registry != nil {
				modifiedCmd, replaceErr = function.ReplaceVariables(stack.Vars, e, modifiedCmd)
			} else {
				modifiedCmd, replaceErr = function.ReplaceVariables(stack.Vars, nil, modifiedCmd)
			}
			if replaceErr != nil {
				cmdErr := fmt.Errorf("variable template error : %w", replaceErr)
				errorMsg.Write([]byte(cmdErr.Error()))
			}
			output, execErr := ShellExec(workDir, shell, shellArg, modifiedCmd, commandTimeout.String())
			allOutput.Reset()
			allOutput.Write(output)
			if execErr != nil {
				cmdErr := fmt.Errorf("command failed : %w\n%s", execErr, output)
				errorMsg.Write([]byte(cmdErr.Error()))
			}
		}
		result.Output = allOutput.String()
		result.Error = errorMsg.String()
		result.ContinueOnError = stack.ContinueOnError
		if result.Error != "" {
			result.Success = false
		}
	}
	return result
}

func ShellExec(args ...string) ([]byte, error) {
	acquire()
	defer release()
	timeout := config.DefaultCommandTimeout
	if len(args) > 4 && strings.TrimSpace(args[4]) != "" {
		parsedTimeout, err := time.ParseDuration(args[4])
		if err != nil {
			return nil, fmt.Errorf("invalid command timeout: %w", err)
		}
		if parsedTimeout <= 0 {
			return nil, fmt.Errorf("invalid command timeout: timeout must be greater than 0")
		}
		timeout = parsedTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if args[1] == "__mvdan__" {
		return ShellExecMvdan(ctx, args[0], args[3], timeout)
	}
	execCmd := exec.CommandContext(ctx, args[1], args[2], args[3])
	execCmd.Dir = args[0]
	output, err := execCmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return output, fmt.Errorf("command timed out after %s", timeout)
	}
	return output, err
}

func ShellExecMvdan(ctx context.Context, workDir string, script string, timeout time.Duration) ([]byte, error) {
	parser := syntax.NewParser()
	file, err := parser.Parse(strings.NewReader(script), "")
	if err != nil {
		return nil, err
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner, err := interp.New(
		interp.Dir(workDir),
		interp.StdIO(nil, &stdout, &stderr),
	)
	if err != nil {
		return nil, err
	}

	err = runner.Run(ctx, file)
	if stderr.Len() > 0 {
		stdout.Write(stderr.Bytes())
	}
	if ctx.Err() == context.DeadlineExceeded {
		return stdout.Bytes(), fmt.Errorf("command timed out after %s", timeout)
	}
	if err != nil {
		return stdout.Bytes(), err
	}
	return stdout.Bytes(), nil
}

func Shellargs(shell string, shellArg string) (string, string, error) {
	if shell == "" {
		return "__mvdan__", "", nil
	}
	if shellArg == "" {
		shellArg = "-c"
	}
	return shell, shellArg, nil
}

func commandTimeoutDuration(timeoutValue string) (time.Duration, error) {
	if strings.TrimSpace(timeoutValue) == "" {
		return config.DefaultCommandTimeout, nil
	}
	duration, err := time.ParseDuration(timeoutValue)
	if err != nil {
		return 0, fmt.Errorf("invalid command timeout: %w", err)
	}
	if duration <= 0 {
		return 0, fmt.Errorf("invalid command timeout: timeout must be greater than 0")
	}
	return duration, nil
}
