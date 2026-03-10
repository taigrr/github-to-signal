// github-to-signal is an HTTP server that receives GitHub webhook events
// and forwards formatted notifications to Signal via signal-cli's JSON-RPC API.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/cbrgm/githubevents/v2/githubevents"
	"github.com/google/go-github/v70/github"
	"github.com/taigrr/signalcli"
)

func main() {
	cfg := loadConfig()

	if cfg.SignalAccount == "" {
		log.Fatal("signal_account is required (set GH2SIG_SIGNAL_ACCOUNT or config.toml)")
	}
	if cfg.SignalRecipient == "" {
		log.Fatal("signal_recipient is required (set GH2SIG_SIGNAL_RECIPIENT or config.toml)")
	}

	signal := signalcli.NewClient(cfg.SignalURL, cfg.SignalAccount)
	handle := githubevents.New(cfg.WebhookSecret)

	notifier := &notifier{
		signal:    signal,
		recipient: cfg.SignalRecipient,
	}

	// Register event handlers.
	handle.OnPushEventAny(notifier.onPush)
	handle.OnIssuesEventAny(notifier.onIssue)
	handle.OnIssueCommentEventAny(notifier.onIssueComment)
	handle.OnPullRequestEventAny(notifier.onPR)
	handle.OnPullRequestReviewEventAny(notifier.onPRReview)
	handle.OnPullRequestReviewCommentEventAny(notifier.onPRReviewComment)
	handle.OnReleaseEventAny(notifier.onRelease)
	handle.OnStarEventAny(notifier.onStar)
	handle.OnForkEventAny(notifier.onFork)
	handle.OnWorkflowRunEventAny(notifier.onWorkflowRun)
	handle.OnCreateEventAny(notifier.onCreate)
	handle.OnDeleteEventAny(notifier.onDelete)

	handle.OnError(func(_ context.Context, _ string, _ string, _ interface{}, err error) error {
		log.Printf("webhook error: %v", err)
		return nil
	})

	mux := http.NewServeMux()
	mux.HandleFunc("POST /webhook", func(w http.ResponseWriter, r *http.Request) {
		if err := handle.HandleEventRequest(r); err != nil {
			log.Printf("handle event: %v", err)
			http.Error(w, "webhook processing failed", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "ok")
	})

	log.Printf("listening on %s", cfg.ListenAddr)
	if err := http.ListenAndServe(cfg.ListenAddr, mux); err != nil {
		log.Fatal(err)
	}
}

type notifier struct {
	signal    *signalcli.Client
	recipient string
}

func (n *notifier) send(ctx context.Context, msg string) {
	if msg == "" {
		return
	}
	_, err := n.signal.Send(ctx, signalcli.SendParams{
		Recipient: n.recipient,
		Message:   msg,
	})
	if err != nil {
		log.Printf("signal send error: %v", err)
	}
}

func (n *notifier) onPush(ctx context.Context, _ string, _ string, event *github.PushEvent) error {
	n.send(ctx, formatPush(event))
	return nil
}

func (n *notifier) onIssue(ctx context.Context, _ string, _ string, event *github.IssuesEvent) error {
	n.send(ctx, formatIssue(event))
	return nil
}

func (n *notifier) onIssueComment(ctx context.Context, _ string, _ string, event *github.IssueCommentEvent) error {
	n.send(ctx, formatIssueComment(event))
	return nil
}

func (n *notifier) onPR(ctx context.Context, _ string, _ string, event *github.PullRequestEvent) error {
	n.send(ctx, formatPR(event))
	return nil
}

func (n *notifier) onPRReview(ctx context.Context, _ string, _ string, event *github.PullRequestReviewEvent) error {
	n.send(ctx, formatPRReview(event))
	return nil
}

func (n *notifier) onPRReviewComment(ctx context.Context, _ string, _ string, event *github.PullRequestReviewCommentEvent) error {
	n.send(ctx, formatPRReviewComment(event))
	return nil
}

func (n *notifier) onRelease(ctx context.Context, _ string, _ string, event *github.ReleaseEvent) error {
	n.send(ctx, formatRelease(event))
	return nil
}

func (n *notifier) onStar(ctx context.Context, _ string, _ string, event *github.StarEvent) error {
	n.send(ctx, formatStar(event))
	return nil
}

func (n *notifier) onFork(ctx context.Context, _ string, _ string, event *github.ForkEvent) error {
	n.send(ctx, formatFork(event))
	return nil
}

func (n *notifier) onWorkflowRun(ctx context.Context, _ string, _ string, event *github.WorkflowRunEvent) error {
	n.send(ctx, formatWorkflowRun(event))
	return nil
}

func (n *notifier) onCreate(ctx context.Context, _ string, _ string, event *github.CreateEvent) error {
	n.send(ctx, formatCreate(event))
	return nil
}

func (n *notifier) onDelete(ctx context.Context, _ string, _ string, event *github.DeleteEvent) error {
	n.send(ctx, formatDelete(event))
	return nil
}
