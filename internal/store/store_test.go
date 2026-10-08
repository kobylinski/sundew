package store_test

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/kobylinski/sundew/internal/core"
	"github.com/kobylinski/sundew/internal/store"
)

type recordingBus struct {
	mu     sync.Mutex
	events []core.Event
}

func (b *recordingBus) Publish(e core.Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = append(b.events, e)
}
func (b *recordingBus) Subscribe(context.Context) <-chan core.Event { panic("not used") }
func (b *recordingBus) snapshot() []core.Event {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]core.Event(nil), b.events...)
}

func open(t *testing.T) (*store.Store, *recordingBus) {
	t.Helper()
	b := new(recordingBus)
	return store.New(b), b
}

func insert(t *testing.T, s *store.Store, m core.Message) *core.Message {
	t.Helper()
	if err := s.Insert(context.Background(), &m); err != nil {
		t.Fatal(err)
	}
	return &m
}
func fixture() core.Message {
	return core.Message{
		Provider: "twilio", ProviderID: "SM-invented", Account: "invented-account", Direction: core.Outbound,
		From: "+15550000001", To: "+15550000002", Body: "Zażółć, HELLO 100%_", Segments: 2,
		MediaURLs: []string{"https://example.invalid/photo"}, Status: core.StatusQueued,
		Options:  map[string]string{core.OptStatusCallback: "https://example.invalid/status", "custom": "value"},
		Exchange: core.Exchange{Method: "POST", Path: "/send", Query: "key=value", Header: http.Header{"X-Test": []string{"one", "two"}}, Body: "To=example", ResponseStatus: 201, ResponseHeader: http.Header{"Content-Type": []string{"application/json"}}, ResponseBody: `{"sid":"SM-invented"}`},
	}
}

func TestInsertGetAndProviderLookup(t *testing.T) {
	s, b := open(t)
	ctx := context.Background()
	before := time.Now()
	m := fixture()
	if err := s.Insert(ctx, &m); err != nil {
		t.Fatal(err)
	}
	if m.ID == "" || m.CreatedAt.Before(before) || !m.CreatedAt.Equal(m.UpdatedAt) || m.CreatedAt.Location() != time.UTC {
		t.Fatalf("timestamps/ID not assigned: %#v", m)
	}
	for _, get := range []func() (*core.Message, error){func() (*core.Message, error) { return s.Get(ctx, m.ID) }, func() (*core.Message, error) { return s.GetByProviderID(ctx, m.Provider, m.ProviderID) }} {
		got, err := get()
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, &m) {
			t.Fatalf("message changed: got %#v want %#v", got, &m)
		}
	}
	events := b.snapshot()
	if len(events) != 1 || events[0].Type != core.EventCreated || !reflect.DeepEqual(events[0].Message, &m) {
		t.Fatalf("created event: %#v", events)
	}
	m.Body = "changed"
	m.Options["custom"] = "changed"
	m.Exchange.Header.Set("X-Test", "changed")
	m.MediaURLs[0] = "changed"
	got, err := s.Get(ctx, m.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Body == "changed" || events[0].Message.Options["custom"] == "changed" || events[0].Message.Exchange.Header.Get("X-Test") == "changed" || events[0].Message.MediaURLs[0] == "changed" {
		t.Fatal("store/event aliases caller memory")
	}
	got.Options["custom"] = "also changed"
	again, _ := s.Get(ctx, m.ID)
	if again.Options["custom"] != "value" {
		t.Fatal("read aliases store")
	}
	explicit := insert(t, s, core.Message{ID: "explicit", Provider: "other", ProviderID: m.ProviderID})
	if explicit.ID != "explicit" {
		t.Fatal("explicit ID replaced")
	}
	if got, err := s.GetByProviderID(ctx, "other", m.ProviderID); err != nil || got.ID != "explicit" {
		t.Fatalf("provider lookup crossed provider: %v %v", got, err)
	}
}

func TestFiltersAndSince(t *testing.T) {
	s, _ := open(t)
	ctx := context.Background()
	a := insert(t, s, fixture())
	other := fixture()
	other.Provider = "other"
	other.ProviderID = "other"
	other.Account = "other"
	other.From = "other"
	other.To = "other"
	other.Body = "unrelated"
	insert(t, s, other)
	for _, tc := range []struct {
		name string
		f    core.Filter
		want int
	}{
		{"to", core.Filter{To: a.To}, 1}, {"from", core.Filter{From: a.From}, 1}, {"account", core.Filter{Account: a.Account}, 1}, {"provider", core.Filter{Provider: a.Provider}, 1},
		{"body case", core.Filter{BodyContains: "hello"}, 1}, {"unicode case", core.Filter{BodyContains: "ZAŻÓŁĆ"}, 1}, {"literal wildcards", core.Filter{BodyContains: "100%_"}, 1},
		{"inclusive since", core.Filter{Since: a.CreatedAt}, 2}, {"exclusive old", core.Filter{Since: a.CreatedAt.Add(time.Nanosecond)}, 1}, {"future", core.Filter{Since: time.Now().Add(time.Hour)}, 0},
		{"exact to", core.Filter{To: "+1555"}, 0}, {"exact account case", core.Filter{Account: "INVENTED-ACCOUNT"}, 0}, {"combined", core.Filter{To: a.To, From: a.From, Provider: a.Provider, Account: a.Account, BodyContains: "HELLO"}, 1},
		{"incompatible", core.Filter{To: a.To, Provider: "other"}, 0}, {"SQL literal", core.Filter{To: "' OR 1=1 --"}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, next, err := s.List(ctx, tc.f)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != tc.want || next != "" {
				t.Fatalf("got %d, cursor %q; want %d", len(got), next, tc.want)
			}
			if got == nil {
				t.Fatal("nil page instead of empty array")
			}
		})
	}
}

func TestPagingLatestAndLimits(t *testing.T) {
	s, _ := open(t)
	ctx := context.Background()
	var ids []string
	for i := range 123 {
		m := insert(t, s, core.Message{To: "same", Body: fmt.Sprintf("message %d", i)})
		ids = append(ids, m.ID)
	}
	first, next, err := s.List(ctx, core.Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 50 || next == "" {
		t.Fatalf("default page: %d %q", len(first), next)
	}
	var got []string
	cursor := ""
	seen := map[string]bool{}
	for {
		page, next, err := s.List(ctx, core.Filter{To: "same", Limit: 7, Cursor: cursor})
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range page {
			if seen[m.ID] {
				t.Fatal("duplicate across pages")
			}
			seen[m.ID] = true
			got = append(got, m.ID)
		}
		if next == "" {
			break
		}
		cursor = next
	}
	if len(got) != len(ids) {
		t.Fatalf("gap: got %d want %d", len(got), len(ids))
	}
	for i, id := range got {
		if id != ids[len(ids)-1-i] {
			t.Fatal("not newest first")
		}
	}
	latest, err := s.Latest(ctx, core.Filter{To: "same"})
	if err != nil || latest.ID != ids[len(ids)-1] {
		t.Fatalf("latest: %v %v", latest, err)
	}
	for _, f := range []core.Filter{{Limit: -1}, {Limit: 501}, {Cursor: "not a cursor"}, {Cursor: base64.RawURLEncoding.EncodeToString([]byte(`{"v":2,"t":1,"s":1}`))}, {Cursor: base64.RawURLEncoding.EncodeToString([]byte(`{}`))}} {
		if _, _, err := s.List(ctx, f); !errors.Is(err, core.ErrInvalidFilter) {
			t.Fatalf("invalid filter accepted: %#v", f)
		}
	}
	if page, _, err := s.List(ctx, core.Filter{Limit: 500}); err != nil || len(page) != 123 {
		t.Fatalf("maximum page: %d %v", len(page), err)
	}
}

func TestWritesAndMissingMessages(t *testing.T) {
	s, b := open(t)
	ctx := context.Background()
	m := insert(t, s, fixture())
	created := m.CreatedAt
	if err := s.UpdateStatus(ctx, m.ID, core.StatusFailed, "invented-error"); err != nil {
		t.Fatal(err)
	}
	got, err := s.Get(ctx, m.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != core.StatusFailed || got.ErrorCode != "invented-error" || !got.CreatedAt.Equal(created) || got.UpdatedAt.Before(created) {
		t.Fatalf("incorrect update: %#v", got)
	}
	if err := s.Delete(ctx, m.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.Reset(ctx); err != nil {
		t.Fatal(err)
	}
	events := b.snapshot()
	want := []string{core.EventCreated, core.EventUpdated, core.EventDeleted, core.EventReset}
	if len(events) != len(want) {
		t.Fatalf("events: %#v", events)
	}
	for i, kind := range want {
		if events[i].Type != kind {
			t.Fatalf("event %d: %s", i, events[i].Type)
		}
	}
	if events[1].Message.Status != core.StatusFailed || events[2].Message.ID != m.ID || events[3].Message != nil {
		t.Fatal("wrong write payload")
	}
	for _, check := range []func() error{
		func() error { _, err := s.Get(ctx, m.ID); return err }, func() error { _, err := s.GetByProviderID(ctx, "twilio", "missing"); return err }, func() error { _, err := s.Latest(ctx, core.Filter{}); return err },
		func() error { return s.Delete(ctx, m.ID) }, func() error { return s.UpdateStatus(ctx, m.ID, core.StatusSent, "") },
	} {
		if err := check(); !errors.Is(err, core.ErrNotFound) {
			t.Fatalf("want ErrNotFound, got %v", err)
		}
	}
	if len(b.snapshot()) != 4 {
		t.Fatal("failed write published event")
	}
	insert(t, s, fixture())
	insert(t, s, fixture())
	if err := s.Reset(ctx); err != nil {
		t.Fatal(err)
	}
	if items, next, err := s.List(ctx, core.Filter{}); err != nil || len(items) != 0 || next != "" {
		t.Fatalf("reset: %v %s %v", items, next, err)
	}
}

func TestFailedWritesDoNotPublishOrMutate(t *testing.T) {
	s, b := open(t)
	ctx := context.Background()
	m := insert(t, s, core.Message{ID: "duplicate"})
	count := len(b.snapshot())
	copy := *m
	copy.Body = "different"
	copy.CreatedAt = time.Time{}
	if err := s.Insert(ctx, &copy); err == nil {
		t.Fatal("duplicate ID accepted")
	}
	if !copy.CreatedAt.IsZero() {
		t.Fatal("failed insert mutated caller")
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if err := s.Insert(canceled, &core.Message{}); err == nil {
		t.Fatal("canceled insert succeeded")
	}
	if err := s.UpdateStatus(canceled, m.ID, core.StatusFailed, ""); err == nil {
		t.Fatal("canceled update succeeded")
	}
	if err := s.Delete(canceled, m.ID); err == nil {
		t.Fatal("canceled delete succeeded")
	}
	if err := s.Reset(canceled); err == nil {
		t.Fatal("canceled reset succeeded")
	}
	if err := s.Insert(ctx, nil); err == nil {
		t.Fatal("nil insert succeeded")
	}
	if len(b.snapshot()) != count {
		t.Fatal("failed writes published")
	}
}

func TestMemoryIsolation(t *testing.T) {
	a, _ := open(t)
	b, _ := open(t)
	isolated := insert(t, a, fixture())
	if _, err := b.Get(context.Background(), isolated.ID); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("memory stores shared: %v", err)
	}
}

func TestConcurrentInsertsReadsAndStatusWrites(t *testing.T) {
	s, b := open(t)
	ctx := context.Background()
	var wg sync.WaitGroup
	for i := range 100 {
		wg.Go(func() {
			m := core.Message{Provider: "parallel", ProviderID: fmt.Sprint(i)}
			if err := s.Insert(ctx, &m); err != nil {
				t.Error(err)
				return
			}
			if _, err := s.Get(ctx, m.ID); err != nil {
				t.Error(err)
			}
			if err := s.UpdateStatus(ctx, m.ID, core.StatusSent, ""); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	items, _, err := s.List(ctx, core.Filter{Limit: 500})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 100 || len(b.snapshot()) != 200 {
		t.Fatalf("messages=%d events=%d", len(items), len(b.snapshot()))
	}
	ids := map[string]bool{}
	for _, m := range items {
		if ids[m.ID] || m.Status != core.StatusSent {
			t.Fatal("duplicate ID or lost update")
		}
		ids[m.ID] = true
	}
}
