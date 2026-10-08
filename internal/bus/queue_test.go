package bus

import (
	"context"
	"testing"
	"time"

	"github.com/kobylinski/sundew/internal/core"
)

// A subscriber that stays behind must retain its backlog, not every slot used
// during the process lifetime. A burst-then-drain test misses this regression.
func TestQueueStorageTracksBacklog(t *testing.T) {
	b := New()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := b.Subscribe(ctx)
	for range 64 {
		b.Publish(core.Event{})
	}
	deadline := time.After(3 * time.Second)
	for range 100 {
		for range 32 {
			b.Publish(core.Event{})
		}
		for range 32 {
			select {
			case <-ch:
			case <-deadline:
				t.Fatal("delivery stalled")
			}
		}
	}
	b.mu.Lock()
	var sub *subscription
	for candidate := range b.subs {
		sub = candidate
	}
	b.mu.Unlock()
	if sub == nil {
		t.Fatal("live subscription disappeared")
	}
	sub.mu.Lock()
	slots, pending := len(sub.queue), len(sub.queue)-sub.head
	sub.mu.Unlock()
	if pending < 63 || pending > 64 {
		t.Fatalf("expected constant backlog, got %d", pending)
	}
	// Allow batching space without depending on the exact compaction threshold.
	// The append-only history retained over 3,200 slots for this workload.
	if slots > 2048 {
		t.Fatalf("retained historical slots=%d with backlog=%d", slots, pending)
	}
}
