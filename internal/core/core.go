// Package core holds the types and interfaces every other package depends on.
// It is the seam between the store, the provider façades, the query API and the
// callback engine: it imports nothing from this module and contains no behaviour.
// A change here goes through a decide gate, because every lane builds against it.
package core

import (
	"context"
	"errors"
	"net/http"
	"time"
)

// Status is the canonical delivery status of a message.
type Status string

const (
	StatusQueued    Status = "queued"
	StatusSent      Status = "sent"
	StatusDelivered Status = "delivered"
	StatusFailed    Status = "failed"
	StatusReceived  Status = "received" // simulated inbound messages
)

// Direction says which way a message travelled.
type Direction string

const (
	Outbound Direction = "outbound" // the application sent it through a façade
	Inbound  Direction = "inbound"  // Sundew simulated it towards the application
)

// Well-known keys of Message.Options. Provider modules may add their own.
const (
	OptStatusCallback = "status_callback" // URL the status webhooks are delivered to
	OptValidity       = "validity"
	OptSenderID       = "sender_id"
)

// Exchange is the raw HTTP exchange that produced a message: the request as it
// arrived at the façade and the response Sundew returned.
type Exchange struct {
	Method         string      `json:"method"`
	Path           string      `json:"path"`
	Query          string      `json:"query"`
	Header         http.Header `json:"header"`
	Body           string      `json:"body"`
	ResponseStatus int         `json:"response_status"`
	ResponseHeader http.Header `json:"response_header"`
	ResponseBody   string      `json:"response_body"`
}

// Message is the canonical caught message. Its JSON form is the query API's
// message object.
type Message struct {
	ID         string            `json:"id"`          // Sundew's own id; assigned by Store.Insert when empty
	Provider   string            `json:"provider"`    // Provider.Name(), e.g. "twilio"
	ProviderID string            `json:"provider_id"` // provider-shaped id, e.g. a Twilio SID
	Account    string            `json:"account"`     // account or credential identifier used
	Direction  Direction         `json:"direction"`
	From       string            `json:"from"`
	To         string            `json:"to"`
	Body       string            `json:"body"`
	Segments   int               `json:"segments"`
	MediaURLs  []string          `json:"media_urls"`
	Status     Status            `json:"status"`
	ErrorCode  string            `json:"error_code,omitempty"`
	Options    map[string]string `json:"options"`
	Exchange   Exchange          `json:"exchange"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
}

// Filter selects messages. Zero values do not filter and set fields combine
// with AND. To, From, Account and Provider match exactly; BodyContains is a
// case-insensitive substring; Query is a case-insensitive substring matched
// against To, From, Body and Account, any one of which may contain it.
type Filter struct {
	Query        string // the UI's single search field; `q` in the query API
	To           string
	From         string
	BodyContains string
	Account      string
	Provider     string
	Since        time.Time // CreatedAt >= Since
	Limit        int       // 0 means the store's default page size
	Cursor       string    // opaque; from a previous List
}

// ErrNotFound is returned by Store when no message matches.
var ErrNotFound = errors.New("message not found")

// ErrInvalidFilter is returned (wrapped) by Store.List and Store.Latest for a
// malformed cursor or a limit outside the store's range.
var ErrInvalidFilter = errors.New("invalid message filter")

// Store keeps caught messages in memory, newest first, for the life of the
// process; nothing is written to disk. Insert assigns ID when it is empty and
// sets CreatedAt and UpdatedAt when they are zero. Every successful write
// publishes the matching Event on the Bus the store was built with.
type Store interface {
	Insert(ctx context.Context, m *Message) error
	Get(ctx context.Context, id string) (*Message, error)
	GetByProviderID(ctx context.Context, provider, providerID string) (*Message, error)
	// List returns one page, newest first, and the cursor of the next page ("" at the end).
	List(ctx context.Context, f Filter) (items []*Message, next string, err error)
	// Latest returns the newest message matching f, or ErrNotFound.
	Latest(ctx context.Context, f Filter) (*Message, error)
	UpdateStatus(ctx context.Context, id string, s Status, errorCode string) error
	Delete(ctx context.Context, id string) error
	Reset(ctx context.Context) error
}

// Event types published on the Bus.
const (
	EventCreated = "message.created"
	EventUpdated = "message.updated"
	EventDeleted = "message.deleted"
	EventReset   = "store.reset"
)

// Event is one change in the store. Message is nil for EventReset; for
// EventDeleted it carries the deleted message.
type Event struct {
	Type    string   `json:"type"`
	Message *Message `json:"message,omitempty"`
}

// Bus fans store events out to subscribers (the stream endpoint, the callback
// engine). Publish never blocks and no event is dropped: each subscriber has
// its own unbounded queue and receives every event published after it
// subscribed, in publication order, until its context is done.
type Bus interface {
	Publish(e Event)
	// Subscribe delivers events published after the call until ctx is done,
	// then closes the channel.
	Subscribe(ctx context.Context) <-chan Event
}

// InboundMessage describes a message Sundew is asked to simulate towards the
// application.
type InboundMessage struct {
	Account   string
	From      string
	To        string
	Body      string
	MediaURLs []string
}

// Provider is one imitated SMS provider. Adding a provider adds one package
// implementing this interface and one registration line; it never touches the
// store, the query API or the callback engine.
type Provider interface {
	// Name is the stable lower-case identifier stored in Message.Provider.
	Name() string
	// Register mounts the façade's routes. A handler parses the provider's
	// request into a Message, inserts it into d.Store and writes the
	// provider-shaped response. Any credential is accepted and stored as sent;
	// provider errors are triggered by the provider's documented magic numbers.
	Register(mux *http.ServeMux, d Deps)
	// StatusRequest builds the provider-shaped status webhook for m moving to
	// s. It returns nil, nil when m asked for no status callback.
	StatusRequest(ctx context.Context, m *Message, s Status, errorCode string) (*http.Request, error)
	// InboundRequest builds the provider-shaped inbound-message webhook.
	InboundRequest(ctx context.Context, in InboundMessage, webhookURL string) (*http.Request, error)
}

// Config is Sundew's configuration, read from the environment by
// internal/config.
type Config struct {
	Addr            string        // SUNDEW_ADDR, default ":8025"
	CallbackDelay   time.Duration // SUNDEW_CALLBACK_DELAY, between two status webhooks
	CallbackOutcome string        // SUNDEW_CALLBACK_OUTCOME, "delivered" (default) or "failed"
	BaseURL         string        // SUNDEW_BASE_URL, used in links the façades return
	InboundURL      string        // SUNDEW_INBOUND_URL, default target of POST /api/v1/inbound
}

// Deps is what a package needs to register its routes or run.
type Deps struct {
	Config    Config
	Store     Store
	Bus       Bus
	Providers map[string]Provider // by Name()
	Now       func() time.Time
}
