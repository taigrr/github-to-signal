package main

import "strings"

// EventFilter determines which event/action combinations to forward.
// An empty filter allows everything (default behavior).
type EventFilter struct {
	rules map[string]map[string]bool // event -> actions (empty map = all actions)
}

// ParseEventFilter parses a list of filter strings into an EventFilter.
// Format: "event" (all actions) or "event:action" (specific action).
//
// Examples:
//
//	["pull_request:opened", "pull_request:closed"] — only PR open/close
//	["push", "issues"] — all push and issue events
//	["pull_request:opened", "workflow_run"] — PR opens + all workflow runs
//	[] — everything allowed (no filtering)
func ParseEventFilter(filters []string) EventFilter {
	ef := EventFilter{rules: make(map[string]map[string]bool)}

	for _, f := range filters {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}

		event, action, hasAction := strings.Cut(f, ":")
		event = strings.ToLower(strings.TrimSpace(event))
		if event == "" {
			continue
		}

		if _, ok := ef.rules[event]; !ok {
			ef.rules[event] = make(map[string]bool)
		}

		if hasAction {
			action = strings.ToLower(strings.TrimSpace(action))
			if action != "" {
				ef.rules[event][action] = true
			}
		}
	}

	return ef
}

// IsEmpty returns true if no filters are configured (allow everything).
func (ef EventFilter) IsEmpty() bool {
	return len(ef.rules) == 0
}

// EventEnabled returns true if the event type is in the filter.
func (ef EventFilter) EventEnabled(event string) bool {
	if ef.IsEmpty() {
		return true
	}
	_, ok := ef.rules[strings.ToLower(event)]
	return ok
}

// Allowed returns true if the event/action combination should be forwarded.
func (ef EventFilter) Allowed(event, action string) bool {
	if ef.IsEmpty() {
		return true
	}

	event = strings.ToLower(event)
	actions, ok := ef.rules[event]
	if !ok {
		return false
	}

	// No specific actions configured = all actions allowed
	if len(actions) == 0 {
		return true
	}

	return actions[strings.ToLower(action)]
}
