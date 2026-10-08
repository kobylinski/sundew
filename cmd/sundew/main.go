package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kobylinski/sundew/internal/api"
	"github.com/kobylinski/sundew/internal/bus"
	"github.com/kobylinski/sundew/internal/callback"
	"github.com/kobylinski/sundew/internal/config"
	"github.com/kobylinski/sundew/internal/core"
	"github.com/kobylinski/sundew/internal/provider/twilio"
	"github.com/kobylinski/sundew/internal/server"
	"github.com/kobylinski/sundew/internal/store"
)

var version = "dev"

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--version" {
		fmt.Println(version)
		return
	}
	if err := run(); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	events := bus.New()
	db := store.New(events)
	deps := core.Deps{Config: cfg, Store: db, Bus: events, Providers: make(map[string]core.Provider), Now: time.Now}
	deps.Providers["twilio"] = twilio.New(cfg)
	registrations := []func(*http.ServeMux, core.Deps){
		// registrations:
		deps.Providers["twilio"].Register,
		api.Register,
		// end registrations
	}
	stopCallbacks := callback.Start(ctx, deps)
	defer stopCallbacks()
	handler := server.New(deps, registrations)
	listener, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		return fmt.Errorf("SUNDEW_ADDR: %w", err)
	}
	srv := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second, BaseContext: func(net.Listener) context.Context { return ctx }}
	result := make(chan error, 1)
	go func() { result <- srv.Serve(listener) }()
	log.Printf("Sundew listening on %s (version %s)", listener.Addr(), version)
	select {
	case err := <-result:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdown); err != nil {
			_ = srv.Close()
			return fmt.Errorf("shutdown: %w", err)
		}
		if err := <-result; !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	}
	return nil
}
