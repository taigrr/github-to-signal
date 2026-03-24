package main

import (
	"log"
	"strings"

	"github.com/taigrr/jety"
)

// Config holds the application configuration.
type Config struct {
	ListenAddr      string
	WebhookSecret   string
	CISecret        string
	SignalURL       string
	SignalAccount   string
	SignalRecipient string
	SignalGroupID   string
	Events          EventFilter
	Endpoints       []Endpoint
}

// Endpoint defines a custom HTTP endpoint that forwards messages to one or more Signal groups.
type Endpoint struct {
	Slug     string
	GroupIDs []string
}

func loadConfig() Config {
	jety.SetDefault("listen_addr", ":9900")
	jety.SetDefault("signal_url", "http://127.0.0.1:8080")

	jety.SetEnvPrefix("GH2SIG")
	jety.SetConfigFile("config.toml")
	jety.SetConfigType("toml")
	_ = jety.ReadInConfig()

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
		CISecret:        jety.GetString("ci_secret"),
		SignalURL:       jety.GetString("signal_url"),
		SignalAccount:   jety.GetString("signal_account"),
		SignalRecipient: jety.GetString("signal_recipient"),
		SignalGroupID:   jety.GetString("signal_group_id"),
		Events:          ParseEventFilter(filters),
		Endpoints:       parseEndpoints(),
	}
}

func parseEndpoints() []Endpoint {
	raw := jety.Get("endpoints")
	if raw == nil {
		return nil
	}

	tables, ok := raw.([]map[string]any)
	if !ok {
		log.Printf("warning: endpoints config is not a valid TOML array of tables")
		return nil
	}

	var endpoints []Endpoint
	for _, t := range tables {
		slug, _ := t["slug"].(string)
		if slug == "" {
			log.Printf("warning: endpoint missing slug, skipping")
			continue
		}
		if !strings.HasPrefix(slug, "/") {
			slug = "/" + slug
		}

		var groupIDs []string
		switch v := t["group_ids"].(type) {
		case []any:
			for _, item := range v {
				if s, ok := item.(string); ok && s != "" {
					groupIDs = append(groupIDs, s)
				}
			}
		case []string:
			groupIDs = v
		}

		if len(groupIDs) == 0 {
			log.Printf("warning: endpoint %q has no group_ids, skipping", slug)
			continue
		}

		endpoints = append(endpoints, Endpoint{Slug: slug, GroupIDs: groupIDs})
	}

	return endpoints
}
