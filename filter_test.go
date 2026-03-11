package main

import "testing"

func TestEventFilter(t *testing.T) {
	tests := []struct {
		name    string
		filters []string
		event   string
		action  string
		want    bool
	}{
		{"empty allows all", nil, "push", "", true},
		{"event only", []string{"push"}, "push", "", true},
		{"event only blocks other", []string{"push"}, "issues", "opened", false},
		{"event:action match", []string{"pull_request:opened"}, "pull_request", "opened", true},
		{"event:action no match", []string{"pull_request:opened"}, "pull_request", "closed", false},
		{"multiple actions", []string{"pull_request:opened", "pull_request:closed"}, "pull_request", "closed", true},
		{"mixed event and action", []string{"push", "pull_request:opened"}, "push", "", true},
		{"mixed blocks filtered", []string{"push", "pull_request:opened"}, "pull_request", "closed", false},
		{"case insensitive", []string{"Pull_Request:Opened"}, "pull_request", "opened", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ef := ParseEventFilter(tt.filters)
			if got := ef.Allowed(tt.event, tt.action); got != tt.want {
				t.Errorf("Allowed(%q, %q) = %v, want %v", tt.event, tt.action, got, tt.want)
			}
		})
	}
}
