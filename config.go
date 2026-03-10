package main

import "github.com/taigrr/jety"

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
}

func loadConfig() Config {
	jety.SetDefault("listen_addr", ":9900")
	jety.SetDefault("signal_url", "http://127.0.0.1:8080")

	jety.SetEnvPrefix("GH2SIG")
	jety.SetConfigFile("config.toml")
	jety.SetConfigType("toml")
	_ = jety.ReadInConfig()

	return Config{
		ListenAddr:      jety.GetString("listen_addr"),
		WebhookSecret:   jety.GetString("webhook_secret"),
		SignalURL:       jety.GetString("signal_url"),
		SignalAccount:   jety.GetString("signal_account"),
		SignalRecipient: jety.GetString("signal_recipient"),
		SignalGroupID:   jety.GetString("signal_group_id"),
	}
}
