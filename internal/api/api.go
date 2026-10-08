// Package api exposes the stable JSON and SSE interface used by tests and the UI.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/kobylinski/sundew/internal/core"
)

type endpoint struct {
	path, method, summary string
	handler               func(http.ResponseWriter, *http.Request)
}

type api struct {
	deps      core.Deps
	client    *http.Client
	heartbeat time.Duration
}

// Register mounts the query API. Dependencies must remain valid until shutdown.
func Register(mux *http.ServeMux, deps core.Deps) {
	a := &api{deps: deps, client: &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, heartbeat: 15 * time.Second}
	a.register(mux)
}

func (a *api) endpoints() []endpoint {
	return []endpoint{
		{"/api/v1/messages", "GET", "List caught messages, newest first", a.list},
		{"/api/v1/messages", "DELETE", "Delete all caught messages", a.reset},
		{"/api/v1/messages/latest", "GET", "Read the newest matching message", a.latest},
		{"/api/v1/messages/stream", "GET", "Subscribe to live store events", a.stream},
		{"/api/v1/messages/{id}", "GET", "Read a caught message", a.get},
		{"/api/v1/messages/{id}", "DELETE", "Delete a caught message", a.delete},
		{"/api/v1/reset", "POST", "Reset the catch store", a.reset},
		{"/api/v1/inbound", "POST", "Deliver and record an inbound SMS", a.inbound},
		{"/api/v1/openapi.json", "GET", "Read this API's OpenAPI document", a.openapi},
	}
}

func (a *api) register(mux *http.ServeMux) {
	routes := map[string][]endpoint{}
	for _, e := range a.endpoints() {
		routes[e.path] = append(routes[e.path], e)
	}
	for path, endpoints := range routes {
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			for _, e := range endpoints {
				if r.Method == e.method {
					e.handler(w, r)
					return
				}
			}
			methods := make([]string, 0, len(endpoints))
			for _, e := range endpoints {
				methods = append(methods, e.method)
			}
			w.Header().Set("Allow", strings.Join(methods, ", "))
			fail(w, http.StatusMethodNotAllowed, "method_not_allowed", "method is not allowed for this route")
		})
	}
	mux.HandleFunc("/api/v1/", func(w http.ResponseWriter, r *http.Request) { fail(w, 404, "not_found", "route not found") })
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func fail(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}

func storeError(w http.ResponseWriter, err error) {
	if errors.Is(err, core.ErrNotFound) {
		fail(w, 404, "not_found", "message not found")
		return
	}
	fail(w, 500, "internal_error", "store operation failed")
}

func parseFilter(r *http.Request) (core.Filter, error) {
	q, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return core.Filter{}, fmt.Errorf("invalid query encoding")
	}
	f := core.Filter{To: q.Get("to"), From: q.Get("from"), BodyContains: q.Get("body"), Account: q.Get("account"), Provider: q.Get("provider"), Cursor: q.Get("cursor")}
	if q.Has("since") {
		f.Since, err = time.Parse(time.RFC3339Nano, q.Get("since"))
		if err != nil {
			return f, fmt.Errorf("since must be an RFC 3339 timestamp")
		}
	}
	if q.Has("limit") {
		f.Limit, err = strconv.Atoi(q.Get("limit"))
		if err != nil || f.Limit < 1 || f.Limit > 500 {
			return f, fmt.Errorf("limit must be an integer between 1 and 500")
		}
	}
	return f, nil
}

func (a *api) list(w http.ResponseWriter, r *http.Request) {
	f, err := parseFilter(r)
	if err != nil {
		fail(w, 400, "invalid_filter", err.Error())
		return
	}
	items, next, err := a.deps.Store.List(r.Context(), f)
	if err != nil {
		storeError(w, err)
		return
	}
	if items == nil {
		items = []*core.Message{}
	}
	writeJSON(w, 200, struct {
		Items      []*core.Message `json:"items"`
		NextCursor string          `json:"next_cursor"`
	}{items, next})
}
func (a *api) latest(w http.ResponseWriter, r *http.Request) {
	f, err := parseFilter(r)
	if err != nil {
		fail(w, 400, "invalid_filter", err.Error())
		return
	}
	m, err := a.deps.Store.Latest(r.Context(), f)
	if err != nil {
		storeError(w, err)
		return
	}
	writeJSON(w, 200, m)
}
func (a *api) get(w http.ResponseWriter, r *http.Request) {
	m, err := a.deps.Store.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		storeError(w, err)
		return
	}
	writeJSON(w, 200, m)
}
func (a *api) delete(w http.ResponseWriter, r *http.Request) {
	if err := a.deps.Store.Delete(r.Context(), r.PathValue("id")); err != nil {
		storeError(w, err)
		return
	}
	w.WriteHeader(204)
}
func (a *api) reset(w http.ResponseWriter, r *http.Request) {
	if err := a.deps.Store.Reset(r.Context()); err != nil {
		storeError(w, err)
		return
	}
	w.WriteHeader(204)
}

func matches(m *core.Message, f core.Filter) bool {
	return m != nil && (f.To == "" || m.To == f.To) && (f.From == "" || m.From == f.From) && (f.Account == "" || m.Account == f.Account) && (f.Provider == "" || m.Provider == f.Provider) && (f.BodyContains == "" || strings.Contains(strings.ToLower(m.Body), strings.ToLower(f.BodyContains))) && (f.Since.IsZero() || !m.CreatedAt.Before(f.Since))
}
func (a *api) stream(w http.ResponseWriter, r *http.Request) {
	f, err := parseFilter(r)
	if err != nil {
		fail(w, 400, "invalid_filter", err.Error())
		return
	}
	rc := http.NewResponseController(w)
	// Subscribe before flushing headers: a client receiving them can safely send.
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	events := a.deps.Bus.Subscribe(ctx)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	if _, err = io.WriteString(w, ": connected\n\n"); err != nil {
		return
	}
	if err = rc.Flush(); err != nil {
		return
	}
	ticker := time.NewTicker(a.heartbeat)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			if _, err = io.WriteString(w, ": heartbeat\n\n"); err != nil {
				return
			}
			if err = rc.Flush(); err != nil {
				return
			}
		case e, ok := <-events:
			if !ok {
				return
			}
			if e.Type != core.EventReset && !matches(e.Message, f) {
				continue
			}
			data, marshalErr := json.Marshal(e)
			if marshalErr != nil {
				return
			}
			if _, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", e.Type, data); err != nil {
				return
			}
			if err = rc.Flush(); err != nil {
				return
			}
		}
	}
}

type inboundInput struct {
	Provider  string   `json:"provider"`
	From      string   `json:"from"`
	To        string   `json:"to"`
	Body      string   `json:"body"`
	Account   string   `json:"account"`
	MediaURLs []string `json:"media_urls"`
	URL       string   `json:"url"`
}

func (a *api) inbound(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	var input *inboundInput
	if err := dec.Decode(&input); err != nil || input == nil {
		fail(w, 400, "invalid_json", "expected an inbound message JSON object")
		return
	}
	in := *input
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		fail(w, 400, "invalid_json", "expected exactly one JSON object")
		return
	}
	if in.Provider == "" {
		in.Provider = "twilio"
	}
	p, ok := a.deps.Providers[in.Provider]
	if !ok {
		fail(w, 400, "unknown_provider", "unknown provider")
		return
	}
	if in.URL == "" {
		in.URL = a.deps.Config.InboundURL
	}
	target, err := url.Parse(in.URL)
	if err != nil || target.Host == "" || (target.Scheme != "http" && target.Scheme != "https") {
		fail(w, 400, "invalid_url", "an http or https webhook url is required")
		return
	}
	req, err := p.InboundRequest(r.Context(), core.InboundMessage{Account: in.Account, From: in.From, To: in.To, Body: in.Body, MediaURLs: in.MediaURLs}, in.URL)
	if err != nil || req == nil {
		fail(w, 500, "provider_error", "provider could not build inbound webhook")
		return
	}
	resp, err := a.client.Do(req)
	if err != nil {
		fail(w, 500, "webhook_error", "inbound webhook delivery failed")
		return
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
	_ = resp.Body.Close()
	m := &core.Message{Provider: in.Provider, Account: in.Account, Direction: core.Inbound, From: in.From, To: in.To, Body: in.Body, MediaURLs: in.MediaURLs, Status: core.StatusReceived}
	if err = a.deps.Store.Insert(r.Context(), m); err != nil {
		storeError(w, err)
		return
	}
	writeJSON(w, 201, struct {
		Message        *core.Message `json:"message"`
		ResponseStatus int           `json:"response_status"`
	}{m, resp.StatusCode})
}
