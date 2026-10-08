// Package twilio imitates the Twilio Messages REST API and webhook forms.
// Public references, read 2026-10-08:
// https://www.twilio.com/docs/messaging/api/message-resource
// https://www.twilio.com/docs/usage/twilios-response
// https://www.twilio.com/docs/messaging/guides/webhook-request
// https://www.twilio.com/docs/iam/test-credentials#test-sending-an-sms
// https://www.twilio.com/docs/usage/security
// https://www.twilio.com/docs/glossary/what-sms-character-limit
// Error references and coverage limitations are in docs/providers/twilio.md.
package twilio

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/kobylinski/sundew/internal/core"
)

const apiVersion = "2010-04-01"
const optMessagingService = "messaging_service_sid"

// Provider implements core.Provider. Its configuration is immutable after New.
type Provider struct{ config core.Config }

var _ core.Provider = (*Provider)(nil)

// New creates a provider using Sundew's response-link configuration.
func New(config core.Config) *Provider { return &Provider{config: config} }

func (*Provider) Name() string { return "twilio" }

func (p *Provider) Register(mux *http.ServeMux, d core.Deps) {
	mux.HandleFunc("POST /2010-04-01/Accounts/{account}/Messages.json", func(w http.ResponseWriter, r *http.Request) { p.send(w, r, d) })
	mux.HandleFunc("GET /2010-04-01/Accounts/{account}/Messages/{message}", func(w http.ResponseWriter, r *http.Request) { p.fetch(w, r, d) })
}

func (p *Provider) send(w http.ResponseWriter, r *http.Request, d core.Deps) {
	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || contentType != "application/x-www-form-urlencoded" {
		writeError(w, 400, 20001, "Expected application/x-www-form-urlencoded")
		return
	}
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		status := http.StatusBadRequest
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			status = http.StatusRequestEntityTooLarge
		}
		writeError(w, status, 20001, "Unable to read request body")
		return
	}
	form, err := url.ParseQuery(string(raw))
	if err != nil {
		writeError(w, 400, 20001, "Invalid form encoding")
		return
	}
	if form.Get("To") == "" {
		writeError(w, 400, 21604, "The destination 'To' phone number is required to send an SMS")
		return
	}
	if form.Get("From") == "" && form.Get("MessagingServiceSid") == "" {
		writeError(w, 400, 21603, "A 'From' or 'MessagingServiceSid' parameter is required to send a message")
		return
	}
	media := append([]string{}, form["MediaUrl"]...)
	if form.Get("Body") == "" && len(media) == 0 {
		writeError(w, 400, 21602, "Message body is required")
		return
	}
	if code, message := magicError(form.Get("From"), form.Get("To")); code != 0 {
		writeError(w, http.StatusBadRequest, code, message)
		return
	}
	account, _, hasBasic := r.BasicAuth()
	if !hasBasic {
		account = r.PathValue("account")
	}
	sid, err := newSID()
	if err != nil {
		writeError(w, 500, 20500, "Internal Server Error")
		return
	}
	now := time.Now().UTC()
	if d.Now != nil {
		now = d.Now().UTC()
	}
	options := make(map[string]string)
	for k, values := range form {
		switch k {
		case "To", "From", "Body", "MediaUrl":
		default:
			options[k] = values[0]
		}
	}
	options[core.OptStatusCallback] = form.Get("StatusCallback")
	options[core.OptValidity] = form.Get("ValidityPeriod")
	options[optMessagingService] = form.Get("MessagingServiceSid")
	m := &core.Message{Provider: p.Name(), ProviderID: sid, Account: account, Direction: core.Outbound,
		From: form.Get("From"), To: form.Get("To"), Body: form.Get("Body"), Segments: segments(form.Get("Body")),
		MediaURLs: media, Status: core.StatusQueued, Options: options, CreatedAt: now, UpdatedAt: now}
	response, err := json.Marshal(p.resource(m))
	if err != nil {
		writeError(w, 500, 20500, "Internal Server Error")
		return
	}
	response = append(response, '\n')
	header := r.Header.Clone()
	m.Exchange = core.Exchange{Method: r.Method, Path: r.URL.EscapedPath(), Query: r.URL.RawQuery, Header: header, Body: string(raw),
		ResponseStatus: http.StatusCreated, ResponseHeader: http.Header{"Content-Type": {"application/json"}}, ResponseBody: string(response)}
	if err := d.Store.Insert(r.Context(), m); err != nil {
		writeError(w, 500, 20500, "Internal Server Error")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write(response)
}

func (p *Provider) fetch(w http.ResponseWriter, r *http.Request, d core.Deps) {
	id, jsonSuffix := strings.CutSuffix(r.PathValue("message"), ".json")
	if !jsonSuffix {
		writeError(w, 404, 20404, "The requested resource was not found")
		return
	}
	m, err := d.Store.GetByProviderID(r.Context(), p.Name(), id)
	if errors.Is(err, core.ErrNotFound) || (err == nil && (m == nil || m.Account != r.PathValue("account"))) {
		writeError(w, 404, 20404, "The requested resource was not found")
		return
	}
	if err != nil {
		writeError(w, 500, 20500, "Internal Server Error")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(p.resource(m))
}

type messageResource struct {
	SID                 string            `json:"sid"`
	AccountSID          string            `json:"account_sid"`
	To                  string            `json:"to"`
	From                *string           `json:"from"`
	Body                string            `json:"body"`
	Status              core.Status       `json:"status"`
	NumSegments         string            `json:"num_segments"`
	NumMedia            string            `json:"num_media"`
	Direction           string            `json:"direction"`
	DateCreated         string            `json:"date_created"`
	DateUpdated         string            `json:"date_updated"`
	DateSent            *string           `json:"date_sent"`
	URI                 string            `json:"uri"`
	SubresourceURIs     map[string]string `json:"subresource_uris"`
	ErrorCode           *int              `json:"error_code"`
	ErrorMessage        *string           `json:"error_message"`
	Price               *string           `json:"price"`
	PriceUnit           *string           `json:"price_unit"`
	APIVersion          string            `json:"api_version"`
	MessagingServiceSID *string           `json:"messaging_service_sid"`
}

func (p *Provider) resource(m *core.Message) messageResource {
	path := "/" + apiVersion + "/Accounts/" + url.PathEscape(m.Account) + "/Messages/" + url.PathEscape(m.ProviderID)
	base := strings.TrimRight(p.config.BaseURL, "/")
	direction := "outbound-api"
	if m.Direction == core.Inbound {
		direction = "inbound"
	}
	result := messageResource{SID: m.ProviderID, AccountSID: m.Account, To: m.To, From: optional(m.From), Body: m.Body, Status: m.Status,
		NumSegments: strconv.Itoa(m.Segments), NumMedia: strconv.Itoa(len(m.MediaURLs)), Direction: direction,
		DateCreated: m.CreatedAt.UTC().Format(time.RFC1123Z), DateUpdated: m.UpdatedAt.UTC().Format(time.RFC1123Z),
		URI: base + path + ".json", SubresourceURIs: map[string]string{"media": base + path + "/Media.json"},
		APIVersion: apiVersion, MessagingServiceSID: optional(m.Options[optMessagingService])}
	// The core seam has no independent sent timestamp. Never invent one from UpdatedAt.
	code := m.ErrorCode
	if code == "" && m.Status == core.StatusFailed {
		code = "30008"
	}
	if code != "" {
		if n, err := strconv.Atoi(code); err == nil {
			result.ErrorCode = &n
		}
		if code == "30008" {
			result.ErrorMessage = optional("Unknown error")
		}
	}
	return result
}

func optional(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func newSID() (string, error) {
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return "", err
	}
	return "SM" + hex.EncodeToString(id[:]), nil
}

func writeError(w http.ResponseWriter, status, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(struct {
		Code     int    `json:"code"`
		Message  string `json:"message"`
		MoreInfo string `json:"more_info"`
		Status   int    `json:"status"`
	}{code, message, "https://www.twilio.com/docs/errors/" + strconv.Itoa(code), status})
}
