package main

import (
	"strings"

	"github.com/taigrr/jety"
)

// Config holds the application configuration.
type Config struct {
	// ListenAddr is the address to bind the webhook server to.
	ListenAddr string
	// WebhookSecret is the GitHub webhook secret for signature validation.
	WebhookSecret string
	// SignalURL is the signal-cli JSON-RPC base URL.
	SignalURL string
	// SignalAccount is the signal-cli account (phone number or UUID).
	SignalAccount string
	// SignalRecipient is the default Signal recipient UUID for DM notifications.
	SignalRecipient string
	// SignalGroupID is the Signal group ID for group notifications (overrides SignalRecipient).
	SignalGroupID string
	// Events is the event filter. Empty means all events are forwarded.
	Events EventFilter
}

func loadConfig() Config {
	jety.SetDefault("listen_addr", ":9900")
	jety.SetDefault("signal_url", "http://127.0.0.1:8080")

	jety.SetEnvPrefix("GH2SIG")
	jety.SetConfigFile("config.toml")
	jety.SetConfigType("toml")
	_ = jety.ReadInConfig()

	// Parse events filter from comma-separated string or TOML array.
	var filters []string
	raw := jety.GetString("events")
	if raw != "" {
		for _, s := range strings.Split(raw, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				filters = append(filters, s)
			}
		}
	}

	return Config{
		ListenAddr:      jety.GetString("listen_addr"),
		WebhookSecret:   jety.GetString("webhook_secret"),
		SignalURL:       jety.GetString("signal_url"),
		SignalAccount:   jety.GetString("signal_account"),
		SignalRecipient: jety.GetString("signal_recipient"),
		SignalGroupID:   jety.GetString("signal_group_id"),
		Events:          ParseEventFilter(filters),
	}
}
