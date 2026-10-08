// Package bus provides reliable, in-process event subscriptions.
package bus

import (
	"context"
	"sync"

	"github.com/kobylinski/sundew/internal/core"
)

// Bus gives every subscriber its own unbounded queue. Publishing only appends
// to queues; it never waits for consumers. Cancel subscriptions when finished
// so their queues and delivery goroutines can be released.
type Bus struct {
	mu   sync.Mutex
	subs map[*subscription]struct{}
}

type subscription struct {
	mu    sync.Mutex
	queue []core.Event
	head  int
	wake  chan struct{}
	out   chan core.Event
}

var _ core.Bus = (*Bus)(nil)

func New() *Bus { return &Bus{subs: make(map[*subscription]struct{})} }

func (b *Bus) Publish(e core.Event) {
	// This lock makes simultaneous publishers agree on one order for every
	// subscriber. No consumer send or wait happens while holding it.
	b.mu.Lock()
	defer b.mu.Unlock()
	for sub := range b.subs {
		sub.mu.Lock()
		sub.queue = append(sub.queue, e)
		sub.mu.Unlock()
		select {
		case sub.wake <- struct{}{}:
		default:
		}
	}
}

func (b *Bus) Subscribe(ctx context.Context) <-chan core.Event {
	sub := &subscription{wake: make(chan struct{}, 1), out: make(chan core.Event)}
	if ctx.Err() != nil {
		close(sub.out)
		return sub.out
	}
	b.mu.Lock()
	b.subs[sub] = struct{}{}
	b.mu.Unlock()
	go func() {
		defer func() {
			b.mu.Lock()
			delete(b.subs, sub)
			b.mu.Unlock()
			close(sub.out)
		}()
		for {
			sub.mu.Lock()
			var e core.Event
			available := sub.head < len(sub.queue)
			if available {
				e = sub.queue[sub.head]
				sub.queue[sub.head] = core.Event{}
				sub.head++
				if sub.head == len(sub.queue) {
					sub.queue = nil
					sub.head = 0
				}
			}
			sub.mu.Unlock()
			if available {
				select {
				case <-ctx.Done():
					return
				case sub.out <- e:
				}
			} else {
				select {
				case <-ctx.Done():
					return
				case <-sub.wake:
				}
			}
		}
	}()
	return sub.out
}
