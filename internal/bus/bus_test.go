package bus_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/kobylinski/sundew/internal/bus"
	"github.com/kobylinski/sundew/internal/core"
)

func TestFanoutAndCancellation(t *testing.T) {
	b := bus.New()
	b.Publish(core.Event{Type: "before"})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	a, c := b.Subscribe(ctx), b.Subscribe(ctx)
	b.Publish(core.Event{Type: core.EventCreated})
	for _, ch := range []<-chan core.Event{a, c} {
		select {
		case e := <-ch:
			if e.Type != core.EventCreated {
				t.Fatalf("unexpected event: %v", e)
			}
		case <-time.After(time.Second):
			t.Fatal("no event")
		}
	}
	cancel()
	for _, ch := range []<-chan core.Event{a, c} {
		select {
		case _, ok := <-ch:
			if ok {
				t.Fatal("channel still open")
			}
		case <-time.After(time.Second):
			t.Fatal("cancellation did not close subscription")
		}
	}
	b.Publish(core.Event{Type: "after"})
}

func TestSlowSubscriberLosesNothingAndDoesNotBlockOthers(t *testing.T) {
	b := bus.New()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	slow := b.Subscribe(ctx)
	done := make(chan struct{})
	go func() {
		for i := range 10000 {
			b.Publish(core.Event{Type: fmt.Sprint(i)})
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("publish blocked on slow subscriber")
	}
	fast := b.Subscribe(ctx)
	b.Publish(core.Event{Type: "fresh"})
	select {
	case e := <-fast:
		if e.Type != "fresh" {
			t.Fatal(e)
		}
	case <-time.After(time.Second):
		t.Fatal("fast subscriber stalled")
	}
	deadline := time.After(3 * time.Second)
	for i := range 10001 {
		want := fmt.Sprint(i)
		if i == 10000 {
			want = "fresh"
		}
		select {
		case e := <-slow:
			if e.Type != want {
				t.Fatalf("event %d: got %q want %q", i, e.Type, want)
			}
		case <-deadline:
			t.Fatalf("lost event at %d", i)
		}
	}
}

func TestCancelSlowSubscriberWithQueuedEvents(t *testing.T) {
	b := bus.New()
	ctx, cancel := context.WithCancel(context.Background())
	ch := b.Subscribe(ctx)
	for range 10000 {
		b.Publish(core.Event{})
	}
	cancel()
	deadline := time.After(time.Second)
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				return
			}
		case <-deadline:
			t.Fatal("queued events prevented cancellation")
		}
	}
}

func TestConcurrentSubscribeCancelPublish(t *testing.T) {
	b := bus.New()
	var wg sync.WaitGroup
	for range 100 {
		wg.Go(func() {
			ctx, cancel := context.WithCancel(context.Background())
			ch := b.Subscribe(ctx)
			b.Publish(core.Event{})
			cancel()
			for range ch {
			}
		})
		wg.Go(func() {
			for range 100 {
				b.Publish(core.Event{})
			}
		})
	}
	wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, ok := <-b.Subscribe(ctx); ok {
		t.Fatal("already-canceled context subscribed")
	}
}
