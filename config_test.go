package main

import "testing"

func TestDefaultSignalURLMatchesDeploymentDocs(t *testing.T) {
	if defaultSignalURL != "http://127.0.0.1:8081" {
		t.Fatalf("defaultSignalURL = %q, want %q", defaultSignalURL, "http://127.0.0.1:8081")
	}
}

func TestParseEndpointsValue(t *testing.T) {
	tests := []struct {
		name string
		raw  any
		want []Endpoint
	}{
		{
			name: "nil config",
			raw:  nil,
			want: nil,
		},
		{
			name: "wrong type",
			raw:  []any{"nope"},
			want: nil,
		},
		{
			name: "normalizes slugs and filters invalid groups",
			raw: []map[string]any{
				{
					"slug":      "prs",
					"group_ids": []any{"group-1", "", 42, "group-2"},
				},
				{
					"slug":      "/releases",
					"group_ids": []string{"group-3"},
				},
			},
			want: []Endpoint{
				{Slug: "/prs", GroupIDs: []string{"group-1", "group-2"}},
				{Slug: "/releases", GroupIDs: []string{"group-3"}},
			},
		},
		{
			name: "skips endpoints missing required fields",
			raw: []map[string]any{
				{"slug": "", "group_ids": []string{"group-1"}},
				{"slug": "/empty-groups", "group_ids": []any{"", nil}},
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseEndpointsValue(tt.raw)
			if len(got) != len(tt.want) {
				t.Fatalf("len(parseEndpointsValue()) = %d, want %d", len(got), len(tt.want))
			}
			for index := range got {
				if got[index].Slug != tt.want[index].Slug {
					t.Fatalf("endpoint %d slug = %q, want %q", index, got[index].Slug, tt.want[index].Slug)
				}
				if len(got[index].GroupIDs) != len(tt.want[index].GroupIDs) {
					t.Fatalf("endpoint %d group count = %d, want %d", index, len(got[index].GroupIDs), len(tt.want[index].GroupIDs))
				}
				for groupIndex := range got[index].GroupIDs {
					if got[index].GroupIDs[groupIndex] != tt.want[index].GroupIDs[groupIndex] {
						t.Fatalf("endpoint %d group %d = %q, want %q", index, groupIndex, got[index].GroupIDs[groupIndex], tt.want[index].GroupIDs[groupIndex])
					}
				}
			}
		})
	}
}
