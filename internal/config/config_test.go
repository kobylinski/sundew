package config_test

import (
	"strings"
	"testing"
	"time"

	"github.com/kobylinski/sundew/internal/config"
)

var names = []string{"ADDR", "CALLBACK_DELAY", "CALLBACK_OUTCOME", "BASE_URL", "INBOUND_URL"}

func clearEnv(t *testing.T) {
	t.Helper()
	for _, name := range names {
		t.Setenv("SUNDEW_"+name, "")
	}
}

func TestDefaults(t *testing.T) {
	clearEnv(t)
	c, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.Addr != ":8025" || c.CallbackOutcome != "delivered" || c.CallbackDelay != 0 || c.BaseURL != "" || c.InboundURL != "" {
		t.Fatalf("unexpected defaults: %#v", c)
	}
}

func TestEnvironment(t *testing.T) {
	clearEnv(t)
	values := map[string]string{"ADDR": "127.0.0.1:9000", "CALLBACK_DELAY": "250ms", "CALLBACK_OUTCOME": "failed", "BASE_URL": "http://sundew:9000", "INBOUND_URL": "https://example.invalid/inbound"}
	for name, value := range values {
		t.Setenv("SUNDEW_"+name, value)
	}
	c, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.Addr != values["ADDR"] || c.CallbackDelay != 250*time.Millisecond || c.CallbackOutcome != "failed" || c.BaseURL != values["BASE_URL"] || c.InboundURL != values["INBOUND_URL"] {
		t.Fatal("environment fields did not round-trip")
	}
}

func TestInvalidValuesNameVariableWithoutLeakingValue(t *testing.T) {
	for _, tc := range []struct{ name, value string }{
		{"ADDR", "missing-port"}, {"ADDR", ":-1"}, {"ADDR", ":65536"},
		{"CALLBACK_DELAY", "soon"}, {"CALLBACK_DELAY", "-1s"}, {"CALLBACK_OUTCOME", "queued"},
		{"BASE_URL", "/relative"}, {"BASE_URL", "ftp://example.invalid"}, {"BASE_URL", "http://user:secret@example.invalid"}, {"INBOUND_URL", "https://example.invalid/#fragment"},
	} {
		t.Run(tc.name+"/"+tc.value, func(t *testing.T) {
			clearEnv(t)
			t.Setenv("SUNDEW_"+tc.name, tc.value)
			_, err := config.Load()
			if err == nil || !strings.Contains(err.Error(), "SUNDEW_"+tc.name) || strings.Contains(err.Error(), "secret") {
				t.Fatalf("wrong error: %v", err)
			}
		})
	}
}
