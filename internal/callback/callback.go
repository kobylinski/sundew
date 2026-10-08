// Package callback delivers provider-built status webhooks for caught messages.
package callback

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/kobylinski/sundew/internal/core"
)

type engine struct {
	deps   core.Deps
	client *http.Client
	wait   func(context.Context, time.Duration) bool
}

// Start subscribes synchronously, then processes callbacks in the background.
// The returned function cancels delivery and waits for all workers to finish.
// Call it before closing the store. Canceling ctx also stops the engine.
func Start(ctx context.Context, d core.Deps) func() {
	e := &engine{deps: d, client: &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, wait: wait}
	return e.start(ctx)
}

func wait(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return ctx.Err() == nil
	}
}

func (e *engine) start(parent context.Context) func() {
	ctx, cancel := context.WithCancel(parent)
	events := e.deps.Bus.Subscribe(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		var workers sync.WaitGroup
		active := map[string]context.CancelFunc{}
		// Completions are consumed by this loop; worker state has one owner.
		finished := make(chan string)
		defer func() {
			cancel()
			for _, stop := range active {
				stop()
			}
			workers.Wait()
		}()
		for {
			select {
			case <-ctx.Done():
				return
			case id := <-finished:
				if stop, ok := active[id]; ok {
					stop()
					delete(active, id)
				}
			case ev, ok := <-events:
				if !ok {
					return
				}
				switch ev.Type {
				case core.EventReset:
					for _, stop := range active {
						stop()
					}
					// Keep canceled workers in the map until completion prevents duplicate IDs.
				case core.EventDeleted:
					if ev.Message != nil {
						if stop := active[ev.Message.ID]; stop != nil {
							stop()
						}
					}
				case core.EventCreated:
					m := ev.Message
					if m == nil || m.Direction != core.Outbound || m.Options[core.OptStatusCallback] == "" {
						continue
					}
					if _, ok := active[m.ID]; ok {
						continue
					}
					if _, ok := e.deps.Providers[m.Provider]; !ok {
						slog.Warn("callback provider unavailable", "provider", m.Provider, "message_id", m.ID)
						continue
					}
					workerCtx, stop := context.WithCancel(ctx)
					active[m.ID] = stop
					workers.Add(1)
					go func(m core.Message) {
						defer workers.Done()
						e.deliver(workerCtx, m)
						select {
						case finished <- m.ID:
						case <-ctx.Done():
						}
					}(*m)
				}
			}
		}
	}()
	return func() { cancel(); <-done }
}

func (e *engine) deliver(ctx context.Context, m core.Message) {
	final := core.StatusDelivered
	if e.deps.Config.CallbackOutcome == "failed" {
		final = core.StatusFailed
	}
	statuses := []core.Status{core.StatusQueued, core.StatusSent, final}
	p := e.deps.Providers[m.Provider]
	for i, status := range statuses {
		if i > 0 && !e.wait(ctx, e.deps.Config.CallbackDelay) {
			return
		}
		if ctx.Err() != nil {
			return
		}
		// Recheck storage when waking: a delete/reset cancellation may still be queued.
		if _, err := e.deps.Store.Get(ctx, m.ID); err != nil {
			return
		}
		if err := e.deps.Store.UpdateStatus(ctx, m.ID, status, ""); err != nil {
			if !errors.Is(err, core.ErrNotFound) && ctx.Err() == nil {
				slog.Warn("callback status update failed", "message_id", m.ID)
			}
			return
		}
		m.Status = status
		m.ErrorCode = ""
		if ctx.Err() != nil {
			return
		}
		req, err := p.StatusRequest(ctx, &m, status, "")
		if err != nil {
			slog.Warn("callback request construction failed", "message_id", m.ID)
			continue
		}
		if req == nil {
			continue
		}
		resp, err := e.client.Do(req)
		if err != nil {
			if ctx.Err() == nil {
				slog.Warn("callback delivery failed", "message_id", m.ID, "status", status)
			}
			continue
		}
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
		_ = resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			slog.Warn("callback receiver refused delivery", "message_id", m.ID, "status", status, "response_status", resp.StatusCode)
		}
	}
}
