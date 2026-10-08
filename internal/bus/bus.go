// Package bus provides bounded, in-process event subscriptions.
package bus

import (
	"context"
	"sync"

	"github.com/kobylinski/sundew/internal/core"
)

// Bus drops new events for a subscriber whose 64-event buffer is full.
// The lock protects channel lifetime; it is never held waiting for a reader.
type Bus struct {
	mu   sync.Mutex
	subs map[chan core.Event]struct{}
}

var _ core.Bus = (*Bus)(nil)

func New() *Bus { return &Bus{subs: make(map[chan core.Event]struct{})} }

func (b *Bus) Publish(e core.Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for ch := range b.subs {
		select {
		case ch <- e:
		default:
		}
	}
}

func (b *Bus) Subscribe(ctx context.Context) <-chan core.Event {
	ch := make(chan core.Event, 64)
	if ctx.Err() != nil {
		close(ch)
		return ch
	}
	b.mu.Lock()
	b.subs[ch] = struct{}{}
	b.mu.Unlock()
	go func() {
		<-ctx.Done()
		b.mu.Lock()
		delete(b.subs, ch)
		close(ch)
		b.mu.Unlock()
	}()
	return ch
}
