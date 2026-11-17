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

// noScript is a sentinel meaning "run the executor with no script argument".
const noScript = "nil"

var (
	semaphoreMu sync.RWMutex
	semaphoreCh = make(chan struct{}, 10)
)

// SetConcurrencyLimit must only be called at startup, before any command runs.
// Swapping the channel while commands are in flight would mismatch acquire/release.
func SetConcurrencyLimit(limit int) {
	if limit <= 0 {
		limit = 1
	}
	semaphoreMu.Lock()
	semaphoreCh = make(chan struct{}, limit)
	semaphoreMu.Unlock()
}

// acquire returns the channel it acquired from so release uses the same one.
func acquire() chan struct{} {
	semaphoreMu.RLock()
	ch := semaphoreCh
	semaphoreMu.RUnlock()
	ch <- struct{}{}
	return ch
}

func release(ch chan struct{}) {
	<-ch
}

// runState holds buffers shared by concurrent goroutines.
// All access goes through the mutex.
type runState struct {
	mu     sync.Mutex
	output bytes.Buffer
	errors bytes.Buffer
}

func (s *runState) appendOutput(b []byte) {
	if len(b) == 0 {
		return
	}
	s.mu.Lock()
	s.output.Write(b)
	s.mu.Unlock()
}

func (s *runState) replaceOutput(b []byte) {
	s.mu.Lock()
	s.output.Reset()
	s.output.Write(b)
	s.mu.Unlock()
}

func (s *runState) appendError(err error) {
	s.mu.Lock()
	if s.errors.Len() > 0 {
		s.errors.WriteByte('\n')
	}
	s.errors.WriteString(err.Error())
	s.mu.Unlock()
}

func (s *runState) outputString() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.output.String()
}

func (s *runState) errorString() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.errors.String()
}

func replaceVars(e *config.Executor, stack *config.Stack, s string) (string, error) {
	if e.Registry != nil {
		return function.ReplaceVariables(stack.Vars, e, s)
	}
	return function.ReplaceVariables(stack.Vars, nil, s)
}

func ExecuteShell(e *config.Executor, stack *config.Stack) *config.Result {
	result := &config.Result{
		Name:    stack.Name,
		Success: true,
	}

	executor := stack.Executor
	if len(executor) == 0 {
		executor = []string{"__mvdan__"}
	}

	// Local copy so we never write to the shared stack.
	mode := stack.ExecutionMode
	if mode == "" {
		mode = "PARALLEL"
	}
	if !slices.Contains([]string{"PARALLEL", "PIPELINE", "STAGE", "SEQUENTIAL"}, mode) {
		result.Error = fmt.Sprintf("invalid executionMode: %s", mode)
		result.Success = false
		return result
	}

	commandTimeout, err := commandTimeoutDuration(stack.Timeouts)
	if err != nil {
		result.Error = err.Error()
		result.Success = false
		return result
	}
	timeoutStr := commandTimeout.String()
	workDir := filepath.Join(e.SourceDir, stack.WorkDir)

	state := &runState{}
	countConcurrent := mode == "PARALLEL" || mode == "STAGE"
	commandsConcurrent := mode == "PARALLEL" || mode == "PIPELINE"

	// run executes one command and records its output/error. Safe for concurrent use.
	run := func(idx int, command string) error {
		script := command
		if command != noScript {
			cmd := strings.ReplaceAll(command, "{{.Count.index}}", strconv.Itoa(idx))
			cmd = strings.ReplaceAll(cmd, "{{.Self.result}}", state.outputString())
			var replaceErr error
			script, replaceErr = replaceVars(e, stack, cmd)
			if replaceErr != nil {
				cmdErr := fmt.Errorf("variable template error : %w", replaceErr)
				state.appendError(cmdErr)
				return cmdErr
			}
		}
		output, execErr := ShellExec(workDir, executor, script, timeoutStr)
		state.appendOutput(output)
		if execErr != nil {
			cmdErr := fmt.Errorf("command failed : %w\n%s", execErr, output)
			state.appendError(cmdErr)
			return cmdErr
		}
		return nil
	}

	runIteration := func(idx int) error {
		// No cmds: run the executor itself once.
		if len(stack.Cmds) == 0 {
			if err := run(idx, noScript); err != nil && !stack.ContinueOnError {
				return err
			}
			return nil
		}

		// SEQUENTIAL / STAGE: commands one after another.
		if !commandsConcurrent {
			for _, cmd := range stack.Cmds {
				if err := run(idx, cmd); err != nil && !stack.ContinueOnError {
					return err
				}
			}
			return nil
		}

		// PARALLEL / PIPELINE: commands concurrently within the iteration.
		var cmdWg sync.WaitGroup
		errs := make(chan error, len(stack.Cmds))
		for _, cmd := range stack.Cmds {
			cmdWg.Add(1)
			go func(command string) {
				defer cmdWg.Done()
				if err := run(idx, command); err != nil {
					errs <- err
				}
			}(cmd)
		}
		cmdWg.Wait()
		close(errs)
		if !stack.ContinueOnError {
			if err, ok := <-errs; ok {
				return err
			}
		}
		return nil
	}

	var wg sync.WaitGroup
	for index := range stack.Count {
		if countConcurrent {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				_ = runIteration(idx) // errors are already recorded in state
			}(index)
		} else if err := runIteration(index); err != nil {
			break
		}
	}
	wg.Wait()

	if stack.Count == 0 {
		result.Output = fmt.Sprintf("%s skipped", stack.Name)
		return result
	}

	if stack.Output != "" {
		tmpl := strings.ReplaceAll(stack.Output, "{{.Self.result}}", state.outputString())
		script, replaceErr := replaceVars(e, stack, tmpl)
		if replaceErr != nil {
			// Don't run a half-rendered command.
			state.appendError(fmt.Errorf("variable template error : %w", replaceErr))
		} else {
			output, execErr := ShellExec(workDir, executor, script, timeoutStr)
			state.replaceOutput(output)
			if execErr != nil {
				state.appendError(fmt.Errorf("command failed : %w\n%s", execErr, output))
			}
		}
	}

	result.Output = state.outputString()
	result.Error = state.errorString()
	result.ContinueOnError = stack.ContinueOnError
	if result.Error != "" {
		result.Success = false
	}
	return result
}

func ShellExec(workdir string, args []string, script string, timeoutVal string) ([]byte, error) {
	timeout := config.DefaultCommandTimeout
	if strings.TrimSpace(timeoutVal) != "" {
		parsed, err := time.ParseDuration(timeoutVal)
		if err != nil {
			return nil, fmt.Errorf("invalid command timeout: %w", err)
		}
		if parsed <= 0 {
			return nil, fmt.Errorf("invalid command timeout: timeout must be greater than 0")
		}
		timeout = parsed
	}
	if len(args) == 0 {
		return nil, fmt.Errorf("no executor configured")
	}

	ch := acquire()
	defer release(ch)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if args[0] == "__mvdan__" {
		if script == noScript {
			return nil, fmt.Errorf("no commands to run")
		}
		return ShellExecMvdan(ctx, workdir, script, timeout)
	}

	if script != noScript {
		args = append(slices.Clone(args), script)
	}
	execCmd := exec.CommandContext(ctx, args[0], args[1:]...) // #nosec G204 -- shell binary and args come from operator YAML config
	execCmd.Dir = workdir
	setupProcessGroup(execCmd)
	output, err := execCmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return output, fmt.Errorf("command timed out after %s", timeout)
	}
	return output, err
}

// syncBuffer lets stdout and stderr share one ordered, goroutine-safe buffer.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) Bytes() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	return slices.Clone(b.buf.Bytes())
}

func ShellExecMvdan(ctx context.Context, workDir string, script string, timeout time.Duration) ([]byte, error) {
	file, err := syntax.NewParser().Parse(strings.NewReader(script), "")
	if err != nil {
		return nil, err
	}

	var out syncBuffer
	runner, err := interp.New(
		interp.Dir(workDir),
		interp.StdIO(nil, &out, &out),
	)
	if err != nil {
		return nil, err
	}

	err = runner.Run(ctx, file)
	if ctx.Err() == context.DeadlineExceeded {
		return out.Bytes(), fmt.Errorf("command timed out after %s", timeout)
	}
	return out.Bytes(), err
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
