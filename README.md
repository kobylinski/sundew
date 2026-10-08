# Sundew

**A catch-all SMS service for development and end-to-end tests — Mailpit, for text messages.**

Sundew stands in for the SMS providers your application talks to (Twilio, Vonage, MessageBird,
Plivo, …). Point the application's provider base URL at Sundew and every message it "sends" is
caught, stored and shown in a small web UI, and available to your E2E tests over a plain HTTP API.
Nothing ever reaches a phone.

## Why the name

The sundew (*Drosera*) is a small carnivorous plant whose leaves carry glistening sticky droplets.
Anything small that lands on them is held fast and kept, in plain view, for the plant to take its
time with. Sundew the service does the same with short messages: whatever your application sends
lands on it, sticks, and stays visible until you have looked. Small prey, small messages; a drop of
dew per text.

## Status

Brief stage. See [docs/journal/2026-10-08/draft-sundew-brief.md](docs/journal/2026-10-08/draft-sundew-brief.md).
