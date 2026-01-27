package executehandler

import (
	"fmt"
	"time"

	"github.com/TechXploreLabs/seristack/internal/config"
	"github.com/TechXploreLabs/seristack/internal/shellexecutor"
)

func ExecuteStack(e *config.Executor, stack *config.Stack) *config.Result {
	start := time.Now()
	fmt.Printf("\nStarting [%s]\n", stack.Name)
	var result *config.Result
	result = shellexecutor.ExecuteShell(e, stack)
	result.Duration = time.Since(start)
	if result.Success {
		fmt.Printf("[%s] ✓ SUCCESS (%.2fs)\n", stack.Name, result.Duration.Seconds())
	} else {
		fmt.Printf("[%s] ✗ FAILED (%.2fs): %v\n", stack.Name, result.Duration.Seconds(), result.Error)
	}
	return result
}
