// Package store keeps canonical messages in memory and publishes committed writes.
package store

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/kobylinski/sundew/internal/core"
)

type entry struct {
	message *core.Message
	seq     uint64
}

// Store owns immutable snapshots; readers and event consumers never receive its
// internal maps or slices. Writes and notifications retain the same order.
type Store struct {
	writes   sync.Mutex
	mu       sync.RWMutex
	messages map[string]entry
	ordered  []entry
	seq      uint64
	events   core.Bus
}

var _ core.Store = (*Store)(nil)

func New(events core.Bus) *Store { return &Store{messages: make(map[string]entry), events: events} }

func clone(m *core.Message) *core.Message {
	c := *m
	c.MediaURLs = slices.Clone(m.MediaURLs)
	c.Options = maps.Clone(m.Options)
	c.Exchange.Header = m.Exchange.Header.Clone()
	c.Exchange.ResponseHeader = m.Exchange.ResponseHeader.Clone()
	return &c
}
func (s *Store) publish(kind string, m *core.Message) {
	if s.events != nil {
		if m != nil {
			m = clone(m)
		}
		s.events.Publish(core.Event{Type: kind, Message: m})
	}
}
func newer(a, b entry) bool {
	if a.message.CreatedAt.Equal(b.message.CreatedAt) {
		return a.seq > b.seq
	}
	return a.message.CreatedAt.After(b.message.CreatedAt)
}

func (s *Store) Insert(ctx context.Context, m *core.Message) error {
	if m == nil {
		return fmt.Errorf("insert: nil message")
	}
	s.writes.Lock()
	defer s.writes.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	saved := clone(m)
	if saved.ID == "" {
		saved.ID = rand.Text()
	}
	now := time.Now().UTC()
	if saved.CreatedAt.IsZero() {
		saved.CreatedAt = now
	}
	if saved.UpdatedAt.IsZero() {
		saved.UpdatedAt = saved.CreatedAt
	}
	s.mu.Lock()
	if _, exists := s.messages[saved.ID]; exists {
		s.mu.Unlock()
		return fmt.Errorf("insert: duplicate message ID")
	}
	s.seq++
	e := entry{message: saved, seq: s.seq}
	index := sort.Search(len(s.ordered), func(i int) bool { return newer(e, s.ordered[i]) })
	s.ordered = slices.Insert(s.ordered, index, e)
	s.messages[saved.ID] = e
	s.mu.Unlock()
	*m = *clone(saved)
	s.publish(core.EventCreated, saved)
	return nil
}
func (s *Store) Get(ctx context.Context, id string) (*core.Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	e, ok := s.messages[id]
	if !ok {
		return nil, core.ErrNotFound
	}
	return clone(e.message), nil
}
func (s *Store) GetByProviderID(ctx context.Context, provider, providerID string) (*core.Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	for _, e := range s.ordered {
		if e.message.Provider == provider && e.message.ProviderID == providerID {
			return clone(e.message), nil
		}
	}
	return nil, core.ErrNotFound
}

type cursor struct {
	Version int       `json:"v"`
	Created time.Time `json:"t"`
	Seq     uint64    `json:"s"`
}

func parseFilter(f core.Filter) (int, *cursor, error) {
	limit := f.Limit
	if limit == 0 {
		limit = 50
	}
	if limit < 1 || limit > 500 {
		return 0, nil, fmt.Errorf("%w: limit must be between 1 and 500", core.ErrInvalidFilter)
	}
	if f.Cursor == "" {
		return limit, nil, nil
	}
	data, err := base64.RawURLEncoding.DecodeString(f.Cursor)
	var c cursor
	if err != nil || json.Unmarshal(data, &c) != nil || c.Version != 1 || c.Seq < 1 || c.Created.IsZero() {
		return 0, nil, fmt.Errorf("%w: invalid cursor", core.ErrInvalidFilter)
	}
	return limit, &c, nil
}
func matches(m *core.Message, f core.Filter, body, query string) bool {
	return (f.To == "" || m.To == f.To) && (f.From == "" || m.From == f.From) &&
		(f.Account == "" || m.Account == f.Account) && (f.Provider == "" || m.Provider == f.Provider) &&
		(f.Since.IsZero() || !m.CreatedAt.Before(f.Since)) &&
		(body == "" || strings.Contains(strings.ToLower(m.Body), body)) &&
		(query == "" || strings.Contains(strings.ToLower(m.To), query) ||
			strings.Contains(strings.ToLower(m.From), query) ||
			strings.Contains(strings.ToLower(m.Body), query) ||
			strings.Contains(strings.ToLower(m.Account), query))
}
func (s *Store) List(ctx context.Context, f core.Filter) ([]*core.Message, string, error) {
	limit, after, err := parseFilter(f)
	if err != nil {
		return nil, "", err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]*core.Message, 0, limit)
	body, query := strings.ToLower(f.BodyContains), strings.ToLower(f.Query)
	var last entry
	for _, e := range s.ordered {
		if err := ctx.Err(); err != nil {
			return nil, "", err
		}
		if after != nil && (e.message.CreatedAt.After(after.Created) || (e.message.CreatedAt.Equal(after.Created) && e.seq >= after.Seq)) {
			continue
		}
		if !matches(e.message, f, body, query) {
			continue
		}
		if len(items) == limit {
			data, _ := json.Marshal(cursor{Version: 1, Created: last.message.CreatedAt, Seq: last.seq})
			return items, base64.RawURLEncoding.EncodeToString(data), nil
		}
		items = append(items, clone(e.message))
		last = e
	}
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	return items, "", nil
}
func (s *Store) Latest(ctx context.Context, f core.Filter) (*core.Message, error) {
	if _, _, err := parseFilter(f); err != nil {
		return nil, err
	}
	f.Limit = 1
	items, _, err := s.List(ctx, f)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, core.ErrNotFound
	}
	return items[0], nil
}
func (s *Store) UpdateStatus(ctx context.Context, id string, status core.Status, errorCode string) error {
	s.writes.Lock()
	defer s.writes.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	e, ok := s.messages[id]
	if !ok {
		s.mu.Unlock()
		return core.ErrNotFound
	}
	saved := clone(e.message)
	saved.Status = status
	saved.ErrorCode = errorCode
	saved.UpdatedAt = time.Now().UTC()
	e.message = saved
	s.messages[id] = e
	for i := range s.ordered {
		if s.ordered[i].message.ID == id {
			s.ordered[i] = e
			break
		}
	}
	s.mu.Unlock()
	s.publish(core.EventUpdated, saved)
	return nil
}
func (s *Store) Delete(ctx context.Context, id string) error {
	s.writes.Lock()
	defer s.writes.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	e, ok := s.messages[id]
	if !ok {
		s.mu.Unlock()
		return core.ErrNotFound
	}
	delete(s.messages, id)
	for i := range s.ordered {
		if s.ordered[i].message.ID == id {
			s.ordered = slices.Delete(s.ordered, i, i+1)
			break
		}
	}
	s.mu.Unlock()
	s.publish(core.EventDeleted, e.message)
	return nil
}
func (s *Store) Reset(ctx context.Context) error {
	s.writes.Lock()
	defer s.writes.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	s.messages = make(map[string]entry)
	s.ordered = nil
	s.mu.Unlock()
	s.publish(core.EventReset, nil)
	return nil
}
