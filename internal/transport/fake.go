package transport

import (
	"context"
	"fmt"
	"sync"
)

// FakeExecutor allows collectors to be tested without a network connection.
type FakeExecutor struct {
	Results  map[string]Result
	FactData Facts
	FactErr  error
	Runs     []Command
	mu       sync.Mutex
}

func (e *FakeExecutor) Run(_ context.Context, cmd Command) Result {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.Runs = append(e.Runs, cmd)
	r, ok := e.Results[cmd.Key]
	if !ok {
		return Result{Key: cmd.Key, Err: fmt.Errorf("no fake result for %q", cmd.Key)}
	}
	r.Key = cmd.Key
	return r
}
func (e *FakeExecutor) RunAll(ctx context.Context, cmds []Command) []Result {
	out := make([]Result, len(cmds))
	for i, cmd := range cmds {
		out[i] = e.Run(ctx, cmd)
	}
	return out
}
func (e *FakeExecutor) Facts(context.Context) (Facts, error) { return e.FactData, e.FactErr }
func (e *FakeExecutor) Close() error                         { return nil }
