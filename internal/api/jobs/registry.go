// Package jobs provides run-time control (pause/resume/cancel) for active generation jobs.
package jobs

import (
	"context"
	"sync"
	"sync/atomic"
)

// Control manages pause/resume/cancel signals for one running generation job.
// Workers call CheckPoint() between file tasks — it blocks while paused and
// returns an error when the job is cancelled.
type Control struct {
	ctx    context.Context
	cancel context.CancelFunc
	gate   sync.RWMutex
	paused atomic.Bool
}

// NewControl creates a Control derived from the given parent context.
func NewControl(parent context.Context) *Control {
	ctx, cancel := context.WithCancel(parent)
	return &Control{ctx: ctx, cancel: cancel}
}

// Context returns the control's context, which is cancelled when Cancel() is called.
func (c *Control) Context() context.Context {
	return c.ctx
}

// CheckPoint is called by workers between file tasks.
// Blocks while the job is paused; returns context.Canceled when cancelled.
func (c *Control) CheckPoint() error {
	if err := c.ctx.Err(); err != nil {
		return err
	}
	c.gate.RLock()
	c.gate.RUnlock()
	return c.ctx.Err()
}

// Pause suspends all workers after their current task completes.
// Safe to call multiple times — subsequent calls are no-ops.
func (c *Control) Pause() {
	if c.paused.CompareAndSwap(false, true) {
		c.gate.Lock()
	}
}

// Resume unblocks paused workers.
// Safe to call when not paused — it is a no-op.
func (c *Control) Resume() {
	if c.paused.CompareAndSwap(true, false) {
		c.gate.Unlock()
	}
}

// Cancel stops all workers. Safe to call while paused.
func (c *Control) Cancel() {
	c.cancel()
	if c.paused.Load() {
		c.Resume() // unblock workers so they observe ctx.Done()
	}
}

// IsPaused reports whether the job is currently paused.
func (c *Control) IsPaused() bool {
	return c.paused.Load()
}

// Registry maps scenario IDs to their active job controls.
type Registry struct {
	mu   sync.Mutex
	jobs map[string]*Control
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{jobs: make(map[string]*Control)}
}

// Register associates a Control with a scenario ID.
func (r *Registry) Register(scenarioID string, ctrl *Control) {
	r.mu.Lock()
	r.jobs[scenarioID] = ctrl
	r.mu.Unlock()
}

// Get returns the Control for a scenario, if one is active.
func (r *Registry) Get(scenarioID string) (*Control, bool) {
	r.mu.Lock()
	ctrl, ok := r.jobs[scenarioID]
	r.mu.Unlock()
	return ctrl, ok
}

// Remove deregisters a scenario's control.
func (r *Registry) Remove(scenarioID string) {
	r.mu.Lock()
	delete(r.jobs, scenarioID)
	r.mu.Unlock()
}
