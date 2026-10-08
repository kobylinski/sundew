package callback

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kobylinski/sundew/internal/core"
)

type fakeBus struct {
	core.Bus
	events     chan core.Event
	subscribed chan struct{}
	closed     chan struct{}
}

func newBus() *fakeBus {
	return &fakeBus{events: make(chan core.Event, 32), subscribed: make(chan struct{}), closed: make(chan struct{})}
}
func (b *fakeBus) Subscribe(ctx context.Context) <-chan core.Event {
	close(b.subscribed)
	go func() { <-ctx.Done(); close(b.closed) }()
	return b.events
}
func (b *fakeBus) Publish(e core.Event) { b.events <- e }

type fakeStore struct {
	core.Store
	mu        sync.Mutex
	messages  map[string]core.Message
	updates   []core.Status
	updateErr error
}

func (s *fakeStore) Get(_ context.Context, id string) (*core.Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.messages[id]
	if !ok {
		return nil, core.ErrNotFound
	}
	return &m, nil
}
func (s *fakeStore) UpdateStatus(ctx context.Context, id string, status core.Status, code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.updateErr != nil {
		return s.updateErr
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	m, ok := s.messages[id]
	if !ok {
		return core.ErrNotFound
	}
	m.Status = status
	m.ErrorCode = code
	s.messages[id] = m
	s.updates = append(s.updates, status)
	return nil
}
func (s *fakeStore) remove(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id == "" {
		clear(s.messages)
	} else {
		delete(s.messages, id)
	}
}
func (s *fakeStore) statuses() []core.Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]core.Status{}, s.updates...)
}

type fakeProvider struct {
	core.Provider
	status func(context.Context, *core.Message, core.Status, string) (*http.Request, error)
}

func (p fakeProvider) StatusRequest(ctx context.Context, m *core.Message, s core.Status, code string) (*http.Request, error) {
	return p.status(ctx, m, s, code)
}
func message(id string) core.Message {
	return core.Message{ID: id, Provider: "fake", Direction: core.Outbound, Status: core.StatusQueued, Options: map[string]string{core.OptStatusCallback: "http://receiver.test"}}
}
func storeFor(messages ...core.Message) *fakeStore {
	s := &fakeStore{messages: map[string]core.Message{}}
	for _, m := range messages {
		s.messages[m.ID] = m
	}
	return s
}
func receive[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(2 * time.Second):
		t.Fatal("timed out")
		var zero T
		return zero
	}
}

type tickClock struct {
	entered  chan time.Duration
	ticks    chan struct{}
	canceled chan struct{}
}

func newClock() *tickClock {
	return &tickClock{entered: make(chan time.Duration, 8), ticks: make(chan struct{}), canceled: make(chan struct{}, 8)}
}
func (c *tickClock) wait(ctx context.Context, d time.Duration) bool {
	c.entered <- d
	select {
	case <-ctx.Done():
		c.canceled <- struct{}{}
		return false
	case <-c.ticks:
		return ctx.Err() == nil
	}
}

func TestStatusSequenceAndReceiverFailure(t *testing.T) {
	for _, outcome := range []string{"delivered", "failed"} {
		for _, receiverStatus := range []int{204, 500} {
			t.Run(fmt.Sprintf("%s/%d", outcome, receiverStatus), func(t *testing.T) {
				m := message("m1")
				s := storeFor(m)
				b := newBus()
				clock := newClock()
				seen := make(chan core.Status, 8)
				built := make(chan core.Status, 8)
				receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					data, _ := io.ReadAll(r.Body)
					status := core.Status(data)
					stored, err := s.Get(r.Context(), m.ID)
					if err != nil || stored.Status != status {
						t.Errorf("status not persisted before send: %+v %v, send=%s", stored, err, status)
					}
					w.WriteHeader(receiverStatus)
					seen <- status
				}))
				defer receiver.Close()
				p := fakeProvider{status: func(ctx context.Context, m *core.Message, status core.Status, code string) (*http.Request, error) {
					if code != "" || m.ErrorCode != "" {
						t.Error("engine chose a provider error code")
					}
					if m.Status != status {
						t.Errorf("message has wrong status %s", m.Status)
					}
					built <- status
					return http.NewRequestWithContext(ctx, "POST", receiver.URL, strings.NewReader(string(status)))
				}}
				e := &engine{deps: core.Deps{Store: s, Bus: b, Providers: map[string]core.Provider{"fake": p}, Config: core.Config{CallbackDelay: 37 * time.Millisecond, CallbackOutcome: outcome}}, client: receiver.Client(), wait: clock.wait}
				stop := e.start(context.Background())
				defer stop()
				select {
				case <-b.subscribed:
				default:
					t.Fatal("Start did not subscribe synchronously")
				}
				b.Publish(core.Event{Type: core.EventCreated, Message: &m})
				if got := receive(t, seen); got != core.StatusQueued {
					t.Fatal(got)
				}
				if got := receive(t, clock.entered); got != 37*time.Millisecond {
					t.Fatal(got)
				}
				clock.ticks <- struct{}{}
				if got := receive(t, seen); got != core.StatusSent {
					t.Fatal(got)
				}
				_ = receive(t, clock.entered)
				clock.ticks <- struct{}{}
				if got := receive(t, seen); got != core.Status(outcome) {
					t.Fatal(got)
				}
				for _, want := range []core.Status{core.StatusQueued, core.StatusSent, core.Status(outcome)} {
					if got := receive(t, built); got != want {
						t.Fatalf("built=%s want %s", got, want)
					}
				}
				stop()
				if got := s.statuses(); !reflect.DeepEqual(got, []core.Status{core.StatusQueued, core.StatusSent, core.Status(outcome)}) {
					t.Fatal(got)
				}
			})
		}
	}
}

func TestCallbackCancellationByDeleteResetAndParent(t *testing.T) {
	for _, kind := range []string{core.EventDeleted, core.EventReset, "parent"} {
		t.Run(kind, func(t *testing.T) {
			m := message("m1")
			s := storeFor(m)
			b := newBus()
			clock := newClock()
			p := fakeProvider{status: func(context.Context, *core.Message, core.Status, string) (*http.Request, error) { return nil, nil }}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			e := &engine{deps: core.Deps{Store: s, Bus: b, Providers: map[string]core.Provider{"fake": p}}, client: http.DefaultClient, wait: clock.wait}
			stop := e.start(ctx)
			defer stop()
			b.Publish(core.Event{Type: core.EventCreated, Message: &m})
			_ = receive(t, clock.entered)
			switch kind {
			case core.EventDeleted:
				s.remove(m.ID)
				b.Publish(core.Event{Type: kind, Message: &m})
			case core.EventReset:
				s.remove("")
				b.Publish(core.Event{Type: kind})
			case "parent":
				cancel()
			}
			_ = receive(t, clock.canceled)
			stop()
			if got := s.statuses(); !reflect.DeepEqual(got, []core.Status{core.StatusQueued}) {
				t.Fatal(got)
			}
			_ = receive(t, b.closed)
		})
	}
}

func TestResetCancelsEveryPendingMessage(t *testing.T) {
	first, second := message("first"), message("second")
	s := storeFor(first, second)
	b := newBus()
	clock := newClock()
	p := fakeProvider{status: func(context.Context, *core.Message, core.Status, string) (*http.Request, error) { return nil, nil }}
	e := &engine{deps: core.Deps{Store: s, Bus: b, Providers: map[string]core.Provider{"fake": p}}, client: http.DefaultClient, wait: clock.wait}
	stop := e.start(context.Background())
	defer stop()
	b.Publish(core.Event{Type: core.EventCreated, Message: &first})
	b.Publish(core.Event{Type: core.EventCreated, Message: &second})
	_ = receive(t, clock.entered)
	_ = receive(t, clock.entered)
	s.remove("")
	b.Publish(core.Event{Type: core.EventReset})
	_ = receive(t, clock.canceled)
	_ = receive(t, clock.canceled)
	stop()
	if len(s.statuses()) != 2 {
		t.Fatal(s.statuses())
	}
}

func TestSkippedAndDuplicateMessages(t *testing.T) {
	valid := message("valid")
	noURL := message("no-url")
	noURL.Options = nil
	inbound := message("inbound")
	inbound.Direction = core.Inbound
	unknown := message("unknown")
	unknown.Provider = "unknown"
	s := storeFor(valid, noURL, inbound, unknown)
	b := newBus()
	clock := newClock()
	built := make(chan string, 8)
	p := fakeProvider{status: func(_ context.Context, m *core.Message, _ core.Status, _ string) (*http.Request, error) {
		built <- m.ID
		return nil, nil
	}}
	e := &engine{deps: core.Deps{Store: s, Bus: b, Providers: map[string]core.Provider{"fake": p}}, client: http.DefaultClient, wait: clock.wait}
	stop := e.start(context.Background())
	defer stop()
	for _, m := range []*core.Message{nil, &noURL, &inbound, &unknown, &valid, &valid} {
		b.Publish(core.Event{Type: core.EventCreated, Message: m})
	}
	if got := receive(t, built); got != "valid" {
		t.Fatal(got)
	}
	_ = receive(t, clock.entered)
	stop()
	select {
	case extra := <-built:
		t.Fatalf("unexpected callback %s", extra)
	default:
	}
	if len(s.statuses()) != 1 {
		t.Fatal(s.statuses())
	}
}

type roundTripper func(*http.Request) (*http.Response, error)

func (f roundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestDeliveryErrorsDoNotStopSequence(t *testing.T) {
	for _, failure := range []string{"build", "transport", "timeout"} {
		t.Run(failure, func(t *testing.T) {
			m := message("m1")
			s := storeFor(m)
			b := newBus()
			seen := make(chan core.Status, 8)
			clock := newClock()
			p := fakeProvider{status: func(ctx context.Context, m *core.Message, status core.Status, _ string) (*http.Request, error) {
				seen <- status
				if failure == "build" {
					return nil, errors.New("provider")
				}
				return http.NewRequestWithContext(ctx, "POST", m.Options[core.OptStatusCallback], nil)
			}}
			client := &http.Client{Timeout: time.Millisecond, Transport: roundTripper(func(r *http.Request) (*http.Response, error) {
				if failure == "timeout" {
					<-r.Context().Done()
					return nil, r.Context().Err()
				}
				return nil, errors.New("transport")
			})}
			e := &engine{deps: core.Deps{Store: s, Bus: b, Providers: map[string]core.Provider{"fake": p}}, client: client, wait: clock.wait}
			stop := e.start(context.Background())
			defer stop()
			b.Publish(core.Event{Type: core.EventCreated, Message: &m})
			if got := receive(t, seen); got != core.StatusQueued {
				t.Fatal(got)
			}
			_ = receive(t, clock.entered)
			clock.ticks <- struct{}{}
			if got := receive(t, seen); got != core.StatusSent {
				t.Fatal(got)
			}
			_ = receive(t, clock.entered)
			clock.ticks <- struct{}{}
			if got := receive(t, seen); got != core.StatusDelivered {
				t.Fatal(got)
			}
			stop()
		})
	}
}

func TestCancellationAbortsActiveRequest(t *testing.T) {
	for _, kind := range []string{core.EventDeleted, core.EventReset, "stop"} {
		t.Run(kind, func(t *testing.T) {
			m := message("m1")
			s := storeFor(m)
			b := newBus()
			started := make(chan struct{})
			canceled := make(chan struct{})
			p := fakeProvider{status: func(ctx context.Context, m *core.Message, _ core.Status, _ string) (*http.Request, error) {
				return http.NewRequestWithContext(ctx, "POST", m.Options[core.OptStatusCallback], nil)
			}}
			client := &http.Client{Transport: roundTripper(func(r *http.Request) (*http.Response, error) {
				close(started)
				<-r.Context().Done()
				close(canceled)
				return nil, r.Context().Err()
			})}
			e := &engine{deps: core.Deps{Store: s, Bus: b, Providers: map[string]core.Provider{"fake": p}}, client: client, wait: wait}
			stop := e.start(context.Background())
			defer stop()
			b.Publish(core.Event{Type: core.EventCreated, Message: &m})
			_ = receive(t, started)
			if kind == "stop" {
				stop()
			} else {
				if kind == core.EventReset {
					s.remove("")
				} else {
					s.remove(m.ID)
				}
				b.Publish(core.Event{Type: kind, Message: &m})
			}
			_ = receive(t, canceled)
			stop()
			if len(s.statuses()) != 1 {
				t.Fatal(s.statuses())
			}
		})
	}
}

func TestMissingStoreMessageAndFailedUpdateStopDelivery(t *testing.T) {
	for _, failure := range []string{"missing", "update"} {
		t.Run(failure, func(t *testing.T) {
			m := message("m1")
			s := storeFor(m)
			if failure == "missing" {
				s.remove(m.ID)
			} else {
				s.updateErr = errors.New("disk")
			}
			called := false
			p := fakeProvider{status: func(context.Context, *core.Message, core.Status, string) (*http.Request, error) {
				called = true
				return nil, nil
			}}
			e := &engine{deps: core.Deps{Store: s, Providers: map[string]core.Provider{"fake": p}}, client: http.DefaultClient, wait: wait}
			e.deliver(context.Background(), m)
			if called {
				t.Fatal("delivered after store failure")
			}
		})
	}
}

func TestPublicStartAndZeroDelay(t *testing.T) {
	m := message("m1")
	s := storeFor(m)
	b := newBus()
	seen := make(chan core.Status, 3)
	p := fakeProvider{status: func(_ context.Context, _ *core.Message, status core.Status, _ string) (*http.Request, error) {
		seen <- status
		return nil, nil
	}}
	stop := Start(context.Background(), core.Deps{Bus: b, Store: s, Providers: map[string]core.Provider{"fake": p}})
	defer stop()
	b.Publish(core.Event{Type: core.EventCreated, Message: &m})
	for _, want := range []core.Status{core.StatusQueued, core.StatusSent, core.StatusDelivered} {
		if got := receive(t, seen); got != want {
			t.Fatal(got)
		}
	}
	stop()
}
