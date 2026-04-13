package main

import (
	"testing"

	"github.com/google/go-github/v84/github"
)

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

func TestFormatPush(t *testing.T) {
	event := &github.PushEvent{
		Ref: strPtr("refs/heads/main"),
		Repo: &github.PushEventRepository{
			FullName: strPtr("taigrr/example"),
		},
		Pusher: &github.CommitAuthor{
			Name: strPtr("tai"),
		},
		Commits: []*github.HeadCommit{
			{
				ID:      strPtr("abc1234567890"),
				Message: strPtr("feat: initial commit"),
			},
			{
				ID:      strPtr("def4567890123"),
				Message: strPtr("fix: typo\n\nLonger description here"),
			},
		},
	}

	got := formatPush(event)
	if got == "" {
		t.Fatal("formatPush returned empty string")
	}

	// Check key parts are present
	checks := []string{
		"[taigrr/example]",
		"tai pushed 2 commits to main",
		"abc1234",
		"feat: initial commit",
		"def4567",
		"fix: typo",
	}
	for _, c := range checks {
		if !contains(got, c) {
			t.Errorf("formatPush missing %q in output:\n%s", c, got)
		}
	}
}

func TestFormatPushSingular(t *testing.T) {
	event := &github.PushEvent{
		Ref: strPtr("refs/heads/dev"),
		Repo: &github.PushEventRepository{
			FullName: strPtr("taigrr/repo"),
		},
		Pusher: &github.CommitAuthor{
			Name: strPtr("tai"),
		},
		Commits: []*github.HeadCommit{
			{
				ID:      strPtr("aaa1111222233"),
				Message: strPtr("docs: update readme"),
			},
		},
	}

	got := formatPush(event)
	if !contains(got, "1 commit to dev") {
		t.Errorf("expected singular 'commit', got:\n%s", got)
	}
}

func TestFormatIssue(t *testing.T) {
	event := &github.IssuesEvent{
		Action: strPtr("opened"),
		Repo: &github.Repository{
			FullName: strPtr("taigrr/example"),
		},
		Sender: &github.User{
			Login: strPtr("contributor"),
		},
		Issue: &github.Issue{
			Number:  intPtr(42),
			Title:   strPtr("Bug: something broken"),
			HTMLURL: strPtr("https://github.com/taigrr/example/issues/42"),
			Body:    strPtr("Steps to reproduce..."),
		},
	}

	got := formatIssue(event)
	checks := []string{
		"[taigrr/example]",
		"contributor",
		"opened",
		"#42",
		"Bug: something broken",
		"https://github.com/taigrr/example/issues/42",
		"Steps to reproduce...",
	}
	for _, c := range checks {
		if !contains(got, c) {
			t.Errorf("formatIssue missing %q in output:\n%s", c, got)
		}
	}
}

func TestFormatIssueComment(t *testing.T) {
	event := &github.IssueCommentEvent{
		Repo: &github.Repository{
			FullName: strPtr("taigrr/example"),
		},
		Sender: &github.User{
			Login: strPtr("reviewer"),
		},
		Issue: &github.Issue{
			Number: intPtr(10),
			Title:  strPtr("Feature request"),
		},
		Comment: &github.IssueComment{
			Body:    strPtr("Looks good to me!"),
			HTMLURL: strPtr("https://github.com/taigrr/example/issues/10#comment-1"),
		},
	}

	got := formatIssueComment(event)
	checks := []string{
		"[taigrr/example]",
		"reviewer",
		"#10",
		"Feature request",
		"Looks good to me!",
	}
	for _, c := range checks {
		if !contains(got, c) {
			t.Errorf("formatIssueComment missing %q in output:\n%s", c, got)
		}
	}
}

func TestFormatPR(t *testing.T) {
	event := &github.PullRequestEvent{
		Action: strPtr("opened"),
		Repo: &github.Repository{
			FullName: strPtr("taigrr/example"),
		},
		Sender: &github.User{
			Login: strPtr("author"),
		},
		PullRequest: &github.PullRequest{
			Number:  intPtr(5),
			Title:   strPtr("Add feature X"),
			HTMLURL: strPtr("https://github.com/taigrr/example/pull/5"),
			Body:    strPtr("This PR adds feature X"),
		},
	}

	got := formatPR(event)
	checks := []string{
		"[taigrr/example]",
		"author",
		"opened",
		"PR #5",
		"Add feature X",
		"This PR adds feature X",
	}
	for _, c := range checks {
		if !contains(got, c) {
			t.Errorf("formatPR missing %q in output:\n%s", c, got)
		}
	}
}

func TestFormatPRReview(t *testing.T) {
	event := &github.PullRequestReviewEvent{
		Repo: &github.Repository{
			FullName: strPtr("taigrr/example"),
		},
		Sender: &github.User{
			Login: strPtr("reviewer"),
		},
		PullRequest: &github.PullRequest{
			Number: intPtr(5),
			Title:  strPtr("Add feature X"),
		},
		Review: &github.PullRequestReview{
			State:   strPtr("approved"),
			Body:    strPtr("Ship it!"),
			HTMLURL: strPtr("https://github.com/taigrr/example/pull/5#pullrequestreview-1"),
		},
	}

	got := formatPRReview(event)
	checks := []string{
		"[taigrr/example]",
		"reviewer",
		"approved",
		"PR #5",
		"Ship it!",
	}
	for _, c := range checks {
		if !contains(got, c) {
			t.Errorf("formatPRReview missing %q in output:\n%s", c, got)
		}
	}
}

func TestFormatPRReviewComment(t *testing.T) {
	event := &github.PullRequestReviewCommentEvent{
		Repo: &github.Repository{
			FullName: strPtr("taigrr/example"),
		},
		Sender: &github.User{
			Login: strPtr("reviewer"),
		},
		PullRequest: &github.PullRequest{
			Number: intPtr(5),
			Title:  strPtr("Add feature X"),
		},
		Comment: &github.PullRequestComment{
			Body:    strPtr("Nit: rename this variable"),
			HTMLURL: strPtr("https://github.com/taigrr/example/pull/5#discussion_r1"),
		},
	}

	got := formatPRReviewComment(event)
	checks := []string{
		"[taigrr/example]",
		"reviewer",
		"PR #5",
		"Nit: rename this variable",
	}
	for _, c := range checks {
		if !contains(got, c) {
			t.Errorf("formatPRReviewComment missing %q in output:\n%s", c, got)
		}
	}
}

func TestFormatRelease(t *testing.T) {
	event := &github.ReleaseEvent{
		Action: strPtr("published"),
		Repo: &github.Repository{
			FullName: strPtr("taigrr/example"),
		},
		Sender: &github.User{
			Login: strPtr("tai"),
		},
		Release: &github.RepositoryRelease{
			TagName: strPtr("v1.0.0"),
			HTMLURL: strPtr("https://github.com/taigrr/example/releases/tag/v1.0.0"),
		},
	}

	got := formatRelease(event)
	checks := []string{
		"[taigrr/example]",
		"tai",
		"published",
		"v1.0.0",
	}
	for _, c := range checks {
		if !contains(got, c) {
			t.Errorf("formatRelease missing %q in output:\n%s", c, got)
		}
	}
}

func TestFormatStar(t *testing.T) {
	event := &github.StarEvent{
		Action: strPtr("created"),
		Repo: &github.Repository{
			FullName:        strPtr("taigrr/example"),
			StargazersCount: intPtr(100),
		},
		Sender: &github.User{
			Login: strPtr("fan"),
		},
	}

	got := formatStar(event)
	if !contains(got, "fan") || !contains(got, "starred") || !contains(got, "100") {
		t.Errorf("formatStar unexpected output:\n%s", got)
	}

	// Test unstar
	event.Action = strPtr("deleted")
	got = formatStar(event)
	if !contains(got, "unstarred") {
		t.Errorf("formatStar deleted should say 'unstarred', got:\n%s", got)
	}
}

func TestFormatFork(t *testing.T) {
	event := &github.ForkEvent{
		Repo: &github.Repository{
			FullName: strPtr("taigrr/example"),
		},
		Forkee: &github.Repository{
			FullName: strPtr("user/example"),
		},
		Sender: &github.User{
			Login: strPtr("user"),
		},
	}

	got := formatFork(event)
	checks := []string{"taigrr/example", "user", "forked", "user/example"}
	for _, c := range checks {
		if !contains(got, c) {
			t.Errorf("formatFork missing %q in output:\n%s", c, got)
		}
	}
}

func TestFormatWorkflowRun(t *testing.T) {
	event := &github.WorkflowRunEvent{
		Action: strPtr("completed"),
		Repo: &github.Repository{
			FullName: strPtr("taigrr/example"),
		},
		WorkflowRun: &github.WorkflowRun{
			Name:       strPtr("CI"),
			Conclusion: strPtr("success"),
			HeadBranch: strPtr("main"),
			HTMLURL:    strPtr("https://github.com/taigrr/example/actions/runs/1"),
		},
	}

	got := formatWorkflowRun(event)
	if !contains(got, "✅") || !contains(got, "CI") || !contains(got, "success") {
		t.Errorf("formatWorkflowRun success unexpected output:\n%s", got)
	}

	// Test failure
	event.WorkflowRun.Conclusion = strPtr("failure")
	got = formatWorkflowRun(event)
	if !contains(got, "❌") {
		t.Errorf("formatWorkflowRun failure should have ❌, got:\n%s", got)
	}

	// Test cancelled
	event.WorkflowRun.Conclusion = strPtr("cancelled")
	got = formatWorkflowRun(event)
	if !contains(got, "⚠️") {
		t.Errorf("formatWorkflowRun cancelled should have ⚠️, got:\n%s", got)
	}

	// Test non-completed action returns empty
	event.Action = strPtr("requested")
	got = formatWorkflowRun(event)
	if got != "" {
		t.Errorf("formatWorkflowRun non-completed should return empty, got:\n%s", got)
	}
}

func TestFormatCreate(t *testing.T) {
	event := &github.CreateEvent{
		Repo: &github.Repository{
			FullName: strPtr("taigrr/example"),
		},
		Sender: &github.User{
			Login: strPtr("tai"),
		},
		RefType: strPtr("branch"),
		Ref:     strPtr("feature-x"),
	}

	got := formatCreate(event)
	if !contains(got, "created") || !contains(got, "branch") || !contains(got, "feature-x") {
		t.Errorf("formatCreate unexpected output:\n%s", got)
	}
}

func TestFormatDelete(t *testing.T) {
	event := &github.DeleteEvent{
		Repo: &github.Repository{
			FullName: strPtr("taigrr/example"),
		},
		Sender: &github.User{
			Login: strPtr("tai"),
		},
		RefType: strPtr("tag"),
		Ref:     strPtr("v0.1.0"),
	}

	got := formatDelete(event)
	if !contains(got, "deleted") || !contains(got, "tag") || !contains(got, "v0.1.0") {
		t.Errorf("formatDelete unexpected output:\n%s", got)
	}
}

func TestFirstLine(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"single line", "single line"},
		{"first\nsecond\nthird", "first"},
		{"", ""},
		{"trailing\n", "trailing"},
	}
	for _, tt := range tests {
		if got := firstLine(tt.input); got != tt.want {
			t.Errorf("firstLine(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is too long", 10, "this is to..."},
		{"", 5, ""},
	}
	for _, tt := range tests {
		if got := truncate(tt.input, tt.maxLen); got != tt.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
	}
}

func TestSplitMessage(t *testing.T) {
	// Short message — no split
	short := "hello"
	chunks := splitMessage(short)
	if len(chunks) != 1 || chunks[0] != short {
		t.Errorf("splitMessage short: got %v", chunks)
	}

	// Exactly at limit
	exact := string(make([]byte, maxMessageLen))
	for i := range exact {
		exact = exact[:i] + "a" + exact[i+1:]
	}
	chunks = splitMessage(exact)
	if len(chunks) != 1 {
		t.Errorf("splitMessage exact: got %d chunks, want 1", len(chunks))
	}

	// Over limit — should split
	long := make([]byte, maxMessageLen+500)
	for i := range long {
		long[i] = 'x'
	}
	// Insert a newline near the boundary for clean split
	long[maxMessageLen-10] = '\n'
	chunks = splitMessage(string(long))
	if len(chunks) < 2 {
		t.Errorf("splitMessage long: expected 2+ chunks, got %d", len(chunks))
	}
	// Verify all content is preserved
	total := 0
	for _, c := range chunks {
		total += len(c)
	}
	// Account for whitespace trimming
	if total < maxMessageLen {
		t.Errorf("splitMessage long: total content %d seems too small", total)
	}
}

func TestFormatIssueClosedNoBody(t *testing.T) {
	event := &github.IssuesEvent{
		Action: strPtr("closed"),
		Repo: &github.Repository{
			FullName: strPtr("taigrr/example"),
		},
		Sender: &github.User{
			Login: strPtr("tai"),
		},
		Issue: &github.Issue{
			Number:  intPtr(1),
			Title:   strPtr("Old bug"),
			HTMLURL: strPtr("https://github.com/taigrr/example/issues/1"),
			Body:    strPtr("Some body that should not appear"),
		},
	}

	got := formatIssue(event)
	// Body should NOT be included for non-opened actions
	if contains(got, "Some body that should not appear") {
		t.Errorf("formatIssue closed should not include body, got:\n%s", got)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(substr) > 0 && containsStr(s, substr)))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
