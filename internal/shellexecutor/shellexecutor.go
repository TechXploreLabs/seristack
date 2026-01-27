package shellexecutor

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/TechXploreLabs/seristack/internal/config"
	"github.com/TechXploreLabs/seristack/internal/function"
)

func ExecuteShell(e *config.Executor, stack *config.Stack) *config.Result {
	result := &config.Result{
		Name:    stack.Name,
		Success: true,
	}
	shell, shellArg, err := Shellargs(stack.Shell, stack.ShellArg)
	if err != nil {
		result.Success = false
		result.Error = err
		return result
	}
	workDir := filepath.Join(e.SourceDir, stack.WorkDir)
	var wg sync.WaitGroup
	errChan := make(chan error, stack.Count)
	var outputMu sync.Mutex
	var allOutput bytes.Buffer
	for index := range stack.Count {
		wg.Add(1)
		executeIteration := func() {
			defer wg.Done()
			for _, cmd := range stack.Cmds {
				indexStr := strconv.Itoa(index)
				modifiedCmd := strings.ReplaceAll(cmd, "{{.Count.index}}", indexStr)
				modifiedCmd, err := function.ReplaceVariables(stack.Vars, e, modifiedCmd)
				if err != nil {
					errChan <- fmt.Errorf("variable template error. %s", err)
					return
				}
				output, err := ShellExec(workDir, shell, shellArg, modifiedCmd)
				outputMu.Lock()
				if len(output) > 0 {
					allOutput.Write(output)
				}
				outputMu.Unlock()
				if err != nil {
					if stack.ContinueOnError {
						fmt.Printf("[%s] Warning: command failed (index %d), continuing: %v\n", stack.Name, index, err)
						continue
					}
					errChan <- fmt.Errorf("command failed (index %d): %w\n%s", index, err, output)
					return
				}
			}
		}
		if stack.IsSerial {
			executeIteration()
		} else {
			go executeIteration()
		}
	}
	wg.Wait()
	close(errChan)
	for err := range errChan {
		result.Success = false
		result.Error = err
		break
	}
	if stack.Count == 0 {
		result.Output = fmt.Sprintf("%s skipped", stack.Name)
	} else {
		result.Output = allOutput.String()
	}
	return result
}

func ShellExec(args ...string) ([]byte, error) {
	execCmd := exec.Command(args[1], args[2], args[3])
	execCmd.Dir = args[0]
	output, err := execCmd.CombinedOutput()
	return output, err
}

func Shellargs(shell string, shellArg string) (string, string, error) {
	if shell == "" {
		switch runtime.GOOS {
		case "windows":
			shell = "powershell.exe"
			shellArg = "-Command"
		case "darwin", "linux":
			shell = "sh"
			shellArg = "-c"
		default:
			return "", "", fmt.Errorf("unsupported OS: %s", runtime.GOOS)
		}
	}
	if shellArg == "" {
		shellArg = "-c"
	}
	return shell, shellArg, nil
}
