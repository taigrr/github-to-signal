package main

import (
	"log"
	"net/url"
	"strings"
	"unicode"

	"github.com/taigrr/jety"
)

const defaultSignalURL = "http://127.0.0.1:8081"

// Built-in HTTP route paths. Custom endpoints may not reuse these, otherwise
// registering them on the ServeMux would panic at startup.
const (
	webhookPath = "/webhook"
	healthPath  = "/health"
)

// reservedEndpointSlugs are paths owned by built-in routes; custom endpoints
// using them are rejected during config parsing.
var reservedEndpointSlugs = map[string]bool{
	webhookPath: true,
	healthPath:  true,
}

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

	// SignalCLIPath, when set, makes this process launch and supervise the
	// signal-cli daemon itself (instead of talking to an externally managed
	// one). This enables the JVM memory cap and RSS watchdog below.
	SignalCLIPath string
	// SignalMemoryLimitMB is the RSS threshold (in MiB) at which the managed
	// signal-cli daemon is restarted. signal-cli runs on the JVM and leaks
	// memory over time; the watchdog bounds its footprint. 0 disables the
	// watchdog.
	SignalMemoryLimitMB int
	// SignalJavaMaxHeapMB caps the managed daemon's JVM max heap via -Xmx.
	// 0 lets the watchdog derive it from SignalMemoryLimitMB.
	SignalJavaMaxHeapMB int
}

// Endpoint defines a custom HTTP endpoint that forwards messages to one or more Signal groups.
type Endpoint struct {
	Slug     string
	GroupIDs []string
}

func loadConfig() Config {
	jety.SetDefault("listen_addr", ":9900")
	jety.SetDefault("signal_url", defaultSignalURL)

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

		SignalCLIPath:       jety.GetString("signal_cli_path"),
		SignalMemoryLimitMB: jety.GetInt("signal_memory_limit_mb"),
		SignalJavaMaxHeapMB: jety.GetInt("signal_java_max_heap_mb"),
	}
}

func parseEndpoints() []Endpoint {
	return parseEndpointsValue(jety.Get("endpoints"))
}

func parseEndpointsValue(raw any) []Endpoint {
	if raw == nil {
		return nil
	}

	tables, ok := raw.([]map[string]any)
	if !ok {
		log.Printf("warning: endpoints config is not a valid TOML array of tables")
		return nil
	}

	var endpoints []Endpoint
	seen := make(map[string]bool)
	for _, t := range tables {
		slug, _ := t["slug"].(string)
		var ok bool
		slug, ok = normalizeEndpointSlug(slug)
		if !ok {
			log.Printf("warning: endpoint has invalid slug, skipping")
			continue
		}
		if reservedEndpointSlugs[slug] {
			log.Printf("warning: endpoint slug %q is reserved, skipping", slug)
			continue
		}
		if seen[slug] {
			log.Printf("warning: duplicate endpoint slug %q, skipping", slug)
			continue
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
			for _, groupID := range v {
				if groupID != "" {
					groupIDs = append(groupIDs, groupID)
				}
			}
		}

		if len(groupIDs) == 0 {
			log.Printf("warning: endpoint %q has no group_ids, skipping", slug)
			continue
		}

		endpoints = append(endpoints, Endpoint{Slug: slug, GroupIDs: groupIDs})
		seen[slug] = true
	}

	return endpoints
}

func normalizeEndpointSlug(slug string) (string, bool) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return "", false
	}
	if !strings.HasPrefix(slug, "/") {
		slug = "/" + slug
	}
	if strings.ContainsAny(slug, "{}") {
		return "", false
	}
	for _, r := range slug {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return "", false
		}
	}

	parsed, err := url.ParseRequestURI(slug)
	if err != nil || parsed.Path != slug || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", false
	}
	return slug, true
}
