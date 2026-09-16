package transport

import (
	"context"
	"errors"
	"os/exec"
	"sync"
	"time"
)

// LocalExecutor is useful for development and tests. It has the same result
// semantics as SSHExecutor, without requiring an SSH server.
type LocalExecutor struct {
	DefaultTimeout time.Duration
	MaxBytes       int64
	Parallel       int
}

func (e *LocalExecutor) Run(ctx context.Context, cmd Command) Result {
	start := time.Now()
	result := Result{Key: cmd.Key, ExitCode: -1}
	timeout := cmd.Timeout
	if timeout <= 0 {
		timeout = e.DefaultTimeout
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	process := exec.CommandContext(ctx, "sh", "-c", cmd.Argv)
	stdout, stderr := newCappedBuffer(limitFor(cmd.MaxBytes, e.MaxBytes)), newCappedBuffer(limitFor(cmd.MaxBytes, e.MaxBytes))
	process.Stdout, process.Stderr = stdout, stderr
	err := process.Run()
	result.Duration, result.Stdout, result.Stderr = time.Since(start), stdout.String(), stderr.String()
	result.Truncated = stdout.Truncated() || stderr.Truncated()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		result.TimedOut, result.ExitCode = true, 124
		return result
	}
	if err == nil {
		result.ExitCode = 0
		return result
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		result.ExitCode = exitErr.ExitCode()
		return result
	}
	result.Err = err
	return result
}

func (e *LocalExecutor) RunAll(ctx context.Context, cmds []Command) []Result {
	results := make([]Result, len(cmds))
	n := e.Parallel
	if n <= 0 {
		n = 4
	}
	if n > 8 {
		n = 8
	}
	jobs := make(chan int)
	var wg sync.WaitGroup
	for worker := 0; worker < n; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				results[i] = e.Run(ctx, cmds[i])
			}
		}()
	}
	for i := range cmds {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	return results
}
func (e *LocalExecutor) Facts(ctx context.Context) (Facts, error) { return probeFacts(ctx, e) }
func (e *LocalExecutor) Close() error                             { return nil }
