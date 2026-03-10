package main

import (
	"fmt"
	"strings"

	"github.com/google/go-github/v70/github"
)

// formatPush formats a push event into a Signal message.
func formatPush(event *github.PushEvent) string {
	repo := event.GetRepo().GetFullName()
	ref := strings.TrimPrefix(event.GetRef(), "refs/heads/")
	pusher := event.GetPusher().GetName()
	count := len(event.Commits)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("[%s] %s pushed %d commit", repo, pusher, count))
	if count != 1 {
		sb.WriteString("s")
	}
	sb.WriteString(fmt.Sprintf(" to %s\n", ref))

	for _, commit := range event.Commits {
		short := commit.GetID()
		if len(short) > 7 {
			short = short[:7]
		}
		msg := firstLine(commit.GetMessage())
		sb.WriteString(fmt.Sprintf("  %s %s\n", short, msg))
	}

	return strings.TrimSpace(sb.String())
}

// formatIssue formats an issue event into a Signal message.
func formatIssue(event *github.IssuesEvent) string {
	repo := event.GetRepo().GetFullName()
	action := event.GetAction()
	issue := event.GetIssue()
	sender := event.GetSender().GetLogin()

	msg := fmt.Sprintf("[%s] %s %s issue #%d: %s\n%s",
		repo, sender, action, issue.GetNumber(), issue.GetTitle(), issue.GetHTMLURL())

	if action == "opened" && issue.GetBody() != "" {
		body := truncate(issue.GetBody(), 200)
		msg += "\n\n" + body
	}

	return msg
}

// formatIssueComment formats an issue comment event into a Signal message.
func formatIssueComment(event *github.IssueCommentEvent) string {
	repo := event.GetRepo().GetFullName()
	sender := event.GetSender().GetLogin()
	issue := event.GetIssue()
	comment := event.GetComment()

	body := truncate(comment.GetBody(), 300)

	return fmt.Sprintf("[%s] %s commented on #%d (%s):\n%s\n%s",
		repo, sender, issue.GetNumber(), issue.GetTitle(), body, comment.GetHTMLURL())
}

// formatPR formats a pull request event into a Signal message.
func formatPR(event *github.PullRequestEvent) string {
	repo := event.GetRepo().GetFullName()
	action := event.GetAction()
	pr := event.GetPullRequest()
	sender := event.GetSender().GetLogin()

	msg := fmt.Sprintf("[%s] %s %s PR #%d: %s\n%s",
		repo, sender, action, pr.GetNumber(), pr.GetTitle(), pr.GetHTMLURL())

	if action == "opened" && pr.GetBody() != "" {
		body := truncate(pr.GetBody(), 200)
		msg += "\n\n" + body
	}

	return msg
}

// formatPRReview formats a pull request review event into a Signal message.
func formatPRReview(event *github.PullRequestReviewEvent) string {
	repo := event.GetRepo().GetFullName()
	sender := event.GetSender().GetLogin()
	pr := event.GetPullRequest()
	review := event.GetReview()

	state := review.GetState()
	body := truncate(review.GetBody(), 200)

	msg := fmt.Sprintf("[%s] %s %s PR #%d: %s\n%s",
		repo, sender, state, pr.GetNumber(), pr.GetTitle(), review.GetHTMLURL())

	if body != "" {
		msg += "\n\n" + body
	}

	return msg
}

// formatPRReviewComment formats a pull request review comment event.
func formatPRReviewComment(event *github.PullRequestReviewCommentEvent) string {
	repo := event.GetRepo().GetFullName()
	sender := event.GetSender().GetLogin()
	pr := event.GetPullRequest()
	comment := event.GetComment()

	body := truncate(comment.GetBody(), 300)

	return fmt.Sprintf("[%s] %s commented on PR #%d (%s):\n%s\n%s",
		repo, sender, pr.GetNumber(), pr.GetTitle(), body, comment.GetHTMLURL())
}

// formatRelease formats a release event into a Signal message.
func formatRelease(event *github.ReleaseEvent) string {
	repo := event.GetRepo().GetFullName()
	release := event.GetRelease()
	sender := event.GetSender().GetLogin()

	return fmt.Sprintf("[%s] %s %s release %s\n%s",
		repo, sender, event.GetAction(), release.GetTagName(), release.GetHTMLURL())
}

// formatStar formats a star event into a Signal message.
func formatStar(event *github.StarEvent) string {
	repo := event.GetRepo().GetFullName()
	sender := event.GetSender().GetLogin()
	action := event.GetAction()
	count := event.GetRepo().GetStargazersCount()

	if action == "deleted" {
		return fmt.Sprintf("[%s] %s unstarred (now %d)", repo, sender, count)
	}
	return fmt.Sprintf("[%s] %s starred (now %d)", repo, sender, count)
}

// formatFork formats a fork event into a Signal message.
func formatFork(event *github.ForkEvent) string {
	repo := event.GetRepo().GetFullName()
	forkee := event.GetForkee().GetFullName()
	sender := event.GetSender().GetLogin()

	return fmt.Sprintf("[%s] %s forked to %s", repo, sender, forkee)
}

// formatWorkflowRun formats a workflow run event into a Signal message.
func formatWorkflowRun(event *github.WorkflowRunEvent) string {
	repo := event.GetRepo().GetFullName()
	run := event.GetWorkflowRun()
	conclusion := run.GetConclusion()
	name := run.GetName()
	branch := run.GetHeadBranch()

	// Only notify on completion
	if event.GetAction() != "completed" {
		return ""
	}

	emoji := "✅"
	if conclusion == "failure" {
		emoji = "❌"
	} else if conclusion == "cancelled" {
		emoji = "⚠️"
	}

	return fmt.Sprintf("%s [%s] workflow %q %s on %s\n%s",
		emoji, repo, name, conclusion, branch, run.GetHTMLURL())
}

// formatCreate formats a create event (branch/tag) into a Signal message.
func formatCreate(event *github.CreateEvent) string {
	repo := event.GetRepo().GetFullName()
	sender := event.GetSender().GetLogin()
	refType := event.GetRefType()
	ref := event.GetRef()

	return fmt.Sprintf("[%s] %s created %s %s", repo, sender, refType, ref)
}

// formatDelete formats a delete event (branch/tag) into a Signal message.
func formatDelete(event *github.DeleteEvent) string {
	repo := event.GetRepo().GetFullName()
	sender := event.GetSender().GetLogin()
	refType := event.GetRefType()
	ref := event.GetRef()

	return fmt.Sprintf("[%s] %s deleted %s %s", repo, sender, refType, ref)
}

func firstLine(s string) string {
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		return s[:idx]
	}
	return s
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
