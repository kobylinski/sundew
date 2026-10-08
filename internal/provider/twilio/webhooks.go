package twilio

import (
	"context"
	"crypto/hmac"
	"crypto/sha1" // Required by Twilio's public webhook signing protocol.
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/kobylinski/sundew/internal/core"
)

func (p *Provider) StatusRequest(ctx context.Context, m *core.Message, status core.Status, errorCode string) (*http.Request, error) {
	target := m.Options[core.OptStatusCallback]
	if target == "" {
		return nil, nil
	}
	form := url.Values{"MessageSid": {m.ProviderID}, "MessageStatus": {string(status)}, "AccountSid": {m.Account},
		"To": {m.To}, "From": {m.From}, "ApiVersion": {apiVersion}}
	if status == core.StatusFailed {
		if errorCode == "" {
			errorCode = "30008"
		}
		form.Set("ErrorCode", errorCode)
	}
	original := &http.Request{Header: m.Exchange.Header}
	_, token, hasBasic := original.BasicAuth()
	return p.webhook(ctx, target, token, hasBasic, form)
}

func (p *Provider) InboundRequest(ctx context.Context, in core.InboundMessage, webhookURL string) (*http.Request, error) {
	sid, err := newSID()
	if err != nil {
		return nil, err
	}
	form := url.Values{"MessageSid": {sid}, "SmsSid": {sid}, "SmsMessageSid": {sid}, "AccountSid": {in.Account},
		"From": {in.From}, "To": {in.To}, "Body": {in.Body}, "NumMedia": {strconv.Itoa(len(in.MediaURLs))},
		"NumSegments": {strconv.Itoa(segments(in.Body))}, "ApiVersion": {apiVersion}}
	for i, media := range in.MediaURLs {
		form.Set("MediaUrl"+strconv.Itoa(i), media)
	}
	return p.webhook(ctx, webhookURL, "", false, form)
}

func (p *Provider) webhook(ctx context.Context, target, token string, sign bool, form url.Values) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	if (req.URL.Scheme != "http" && req.URL.Scheme != "https") || req.URL.Host == "" {
		return nil, fmt.Errorf("twilio webhook requires an absolute http(s) URL")
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if sign {
		signedURL := *req.URL
		signedURL.User = nil
		signedURL.Fragment = ""
		signedURL.RawFragment = ""
		req.Header.Set("X-Twilio-Signature", signature(signedURL.String(), form, token))
	}
	return req, nil
}

func signature(target string, form url.Values, token string) string {
	mac := hmac.New(sha1.New, []byte(token))
	_, _ = mac.Write([]byte(target))
	keys := make([]string, 0, len(form))
	for key := range form {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		// All fields built above are singular. Sorting also supports repeated values
		// according to Twilio's SDK algorithm without depending on insertion order.
		values := append([]string(nil), form[key]...)
		sort.Strings(values)
		last := ""
		for i, value := range values {
			if i > 0 && value == last {
				continue
			}
			_, _ = mac.Write([]byte(key + value))
			last = value
		}
	}
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}
