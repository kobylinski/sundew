// Package config reads and validates Sundew's environment.
package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/kobylinski/sundew/internal/core"
)

// Load reads SUNDEW_* variables. Empty values use the defaults.
func Load() (core.Config, error) {
	c := core.Config{Addr: ":8025", CallbackOutcome: "delivered"}
	if v := os.Getenv("SUNDEW_ADDR"); v != "" {
		_, port, err := net.SplitHostPort(v)
		p, perr := strconv.Atoi(port)
		if err != nil || perr != nil || p < 0 || p > 65535 {
			return c, fmt.Errorf("SUNDEW_ADDR: expected host:port with a numeric port")
		}
		c.Addr = v
	}
	if v := os.Getenv("SUNDEW_CALLBACK_DELAY"); v != "" {
		var err error
		c.CallbackDelay, err = time.ParseDuration(v)
		if err != nil || c.CallbackDelay < 0 {
			return c, fmt.Errorf("SUNDEW_CALLBACK_DELAY: expected a nonnegative duration")
		}
	}
	if v := os.Getenv("SUNDEW_CALLBACK_OUTCOME"); v != "" {
		if v != "delivered" && v != "failed" {
			return c, fmt.Errorf("SUNDEW_CALLBACK_OUTCOME: expected delivered or failed")
		}
		c.CallbackOutcome = v
	}
	for _, field := range []struct {
		name string
		dest *string
	}{{"SUNDEW_BASE_URL", &c.BaseURL}, {"SUNDEW_INBOUND_URL", &c.InboundURL}} {
		v := os.Getenv(field.name)
		if v == "" {
			continue
		}
		u, err := url.Parse(v)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil || u.Fragment != "" {
			return c, fmt.Errorf("%s: expected an absolute HTTP(S) URL without user info or fragment", field.name)
		}
		*field.dest = v
	}
	return c, nil
}
