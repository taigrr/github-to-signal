// github-to-signal is an HTTP server that receives GitHub webhook events
// and forwards formatted notifications to Signal via signal-cli's JSON-RPC API.
package main

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cbrgm/githubevents/v2/githubevents"
	"github.com/google/go-github/v84/github"
	"github.com/taigrr/signalcli"
)

// shutdownTimeout bounds graceful HTTP shutdown on SIGINT/SIGTERM.
const shutdownTimeout = 10 * time.Second

func main() {
	cfg := loadConfig()

	if cfg.SignalAccount == "" {
		log.Fatal("signal_account is required (set GH2SIG_SIGNAL_ACCOUNT or config.toml)")
	}
	if cfg.SignalRecipient == "" && cfg.SignalGroupID == "" {
		log.Fatal("signal_recipient or signal_group_id is required")
	}

	// Root context cancelled on SIGINT/SIGTERM so the managed daemon and HTTP
	// server shut down gracefully (and the leaky signal-cli JVM child is
	// stopped) instead of being orphaned on exit.
	ctx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	signalURL := cfg.SignalURL
	stopDaemon := func() {}
	if cfg.SignalCLIPath != "" {
		baseURL, stop, err := startManagedDaemon(ctx, cfg)
		if err != nil {
			log.Fatalf("managed signal-cli daemon: %v", err)
		}
		stopDaemon = stop
		signalURL = baseURL
	}

	signalClient := signalcli.NewClient(signalURL, cfg.SignalAccount)
	handle := githubevents.New(cfg.WebhookSecret)

	n := &notifier{
		signal:    signalClient,
		recipient: cfg.SignalRecipient,
		groupID:   cfg.SignalGroupID,
		filter:    cfg.Events,
	}

	// Register event handlers (only for enabled events).
	f := cfg.Events
	if f.EventEnabled("push") {
		handle.OnPushEventAny(n.onPush)
	}
	if f.EventEnabled("issues") {
		handle.OnIssuesEventAny(n.onIssue)
	}
	if f.EventEnabled("issue_comment") {
		handle.OnIssueCommentEventAny(n.onIssueComment)
	}
	if f.EventEnabled("pull_request") {
		handle.OnPullRequestEventAny(n.onPR)
	}
	if f.EventEnabled("pull_request_review") {
		handle.OnPullRequestReviewEventAny(n.onPRReview)
	}
	if f.EventEnabled("pull_request_review_comment") {
		handle.OnPullRequestReviewCommentEventAny(n.onPRReviewComment)
	}
	if f.EventEnabled("release") {
		handle.OnReleaseEventAny(n.onRelease)
	}
	if f.EventEnabled("star") {
		handle.OnStarEventAny(n.onStar)
	}
	if f.EventEnabled("fork") {
		handle.OnForkEventAny(n.onFork)
	}
	if f.EventEnabled("workflow_run") {
		handle.OnWorkflowRunEventAny(n.onWorkflowRun)
	}
	if f.EventEnabled("create") {
		handle.OnCreateEventAny(n.onCreate)
	}
	if f.EventEnabled("delete") {
		handle.OnDeleteEventAny(n.onDelete)
	}

	if f.IsEmpty() {
		log.Println("event filter: all events enabled")
	} else {
		log.Printf("event filter: %v", f.rules)
	}

	handle.OnError(func(_ context.Context, _ string, _ string, _ interface{}, err error) error {
		log.Printf("webhook error: %v", err)
		return nil
	})

	mux := http.NewServeMux()
	mux.HandleFunc("POST "+webhookPath, func(w http.ResponseWriter, r *http.Request) {
		if err := handle.HandleEventRequest(r); err != nil {
			log.Printf("handle event: %v", err)
			http.Error(w, "webhook processing failed", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	for _, ep := range cfg.Endpoints {
		mux.HandleFunc("POST "+ep.Slug, n.handleCustom(cfg.CISecret, ep.GroupIDs))
		log.Printf("custom endpoint enabled: POST %s -> %d group(s)", ep.Slug, len(ep.GroupIDs))
	}
	mux.HandleFunc("GET "+healthPath, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "ok")
	})

	log.Printf("listening on %s", cfg.ListenAddr)
	srv := &http.Server{Addr: cfg.ListenAddr, Handler: mux}

	serverErr := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		stopDaemon()
		log.Fatal(err)
	case <-ctx.Done():
		log.Println("shutting down")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("http shutdown error: %v", err)
	}
	stopDaemon()
}

type notifier struct {
	signal    *signalcli.Client
	recipient string
	groupID   string
	filter    EventFilter
}

func (n *notifier) send(ctx context.Context, msg string) {
	if msg == "" {
		return
	}
	params := signalcli.SendParams{Message: msg}
	if n.groupID != "" {
		params.GroupID = n.groupID
	} else {
		params.Recipient = n.recipient
	}
	_, err := n.signal.Send(ctx, params)
	if err != nil {
		log.Printf("signal send error: %v", err)
	}
}

func (n *notifier) onPush(ctx context.Context, _ string, _ string, event *github.PushEvent) error {
	if !n.filter.Allowed("push", "") {
		return nil
	}
	n.send(ctx, formatPush(event))
	return nil
}

func (n *notifier) onIssue(ctx context.Context, _ string, _ string, event *github.IssuesEvent) error {
	if !n.filter.Allowed("issues", event.GetAction()) {
		return nil
	}
	n.send(ctx, formatIssue(event))
	return nil
}

func (n *notifier) onIssueComment(ctx context.Context, _ string, _ string, event *github.IssueCommentEvent) error {
	if !n.filter.Allowed("issue_comment", event.GetAction()) {
		return nil
	}
	n.send(ctx, formatIssueComment(event))
	return nil
}

func (n *notifier) onPR(ctx context.Context, _ string, _ string, event *github.PullRequestEvent) error {
	if !n.filter.Allowed("pull_request", event.GetAction()) {
		return nil
	}
	n.send(ctx, formatPR(event))
	return nil
}

func (n *notifier) onPRReview(ctx context.Context, _ string, _ string, event *github.PullRequestReviewEvent) error {
	if !n.filter.Allowed("pull_request_review", event.GetAction()) {
		return nil
	}
	n.send(ctx, formatPRReview(event))
	return nil
}

func (n *notifier) onPRReviewComment(ctx context.Context, _ string, _ string, event *github.PullRequestReviewCommentEvent) error {
	if !n.filter.Allowed("pull_request_review_comment", event.GetAction()) {
		return nil
	}
	n.send(ctx, formatPRReviewComment(event))
	return nil
}

func (n *notifier) onRelease(ctx context.Context, _ string, _ string, event *github.ReleaseEvent) error {
	if !n.filter.Allowed("release", event.GetAction()) {
		return nil
	}
	n.send(ctx, formatRelease(event))
	return nil
}

func (n *notifier) onStar(ctx context.Context, _ string, _ string, event *github.StarEvent) error {
	if !n.filter.Allowed("star", event.GetAction()) {
		return nil
	}
	n.send(ctx, formatStar(event))
	return nil
}

func (n *notifier) onFork(ctx context.Context, _ string, _ string, event *github.ForkEvent) error {
	if !n.filter.Allowed("fork", "") {
		return nil
	}
	n.send(ctx, formatFork(event))
	return nil
}

func (n *notifier) onWorkflowRun(ctx context.Context, _ string, _ string, event *github.WorkflowRunEvent) error {
	if !n.filter.Allowed("workflow_run", event.GetAction()) {
		return nil
	}
	n.send(ctx, formatWorkflowRun(event))
	return nil
}

func (n *notifier) onCreate(ctx context.Context, _ string, _ string, event *github.CreateEvent) error {
	if !n.filter.Allowed("create", "") {
		return nil
	}
	n.send(ctx, formatCreate(event))
	return nil
}

func (n *notifier) onDelete(ctx context.Context, _ string, _ string, event *github.DeleteEvent) error {
	if !n.filter.Allowed("delete", "") {
		return nil
	}
	n.send(ctx, formatDelete(event))
	return nil
}

type customMessage struct {
	Source  string `json:"source"`
	Message string `json:"message"`
}

func (n *notifier) handleCustom(secret string, groupIDs []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if secret != "" {
			provided := r.Header.Get("X-CI-Secret")
			if subtle.ConstantTimeCompare([]byte(provided), []byte(secret)) != 1 {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		}

		var msg customMessage
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		if msg.Message == "" {
			http.Error(w, "message is required", http.StatusBadRequest)
			return
		}

		text := msg.Message
		if msg.Source != "" {
			text = fmt.Sprintf("[%s] %s", msg.Source, msg.Message)
		}

		n.sendToGroups(r.Context(), text, groupIDs)
		w.WriteHeader(http.StatusOK)
	}
}

const maxMessageLen = 2000

func (n *notifier) sendToGroups(ctx context.Context, msg string, groupIDs []string) {
	chunks := splitMessage(msg)
	for _, gid := range groupIDs {
		for _, chunk := range chunks {
			params := signalcli.SendParams{Message: chunk, GroupID: gid}
			if _, err := n.signal.Send(ctx, params); err != nil {
				log.Printf("signal send error (group %s): %v", gid, err)
			}
		}
	}
}

func splitMessage(msg string) []string {
	runes := []rune(msg)
	if len(runes) <= maxMessageLen {
		return []string{msg}
	}

	var chunks []string
	for len(runes) > 0 {
		end := maxMessageLen
		if end > len(runes) {
			end = len(runes)
		}
		if end < len(runes) {
			// Prefer splitting on a newline, searching by rune index so
			// multibyte characters don't shift the boundary out of range.
			for i := end - 1; i > 0; i-- {
				if runes[i] == '\n' {
					end = i + 1
					break
				}
			}
		}
		chunks = append(chunks, string(runes[:end]))
		runes = runes[end:]
	}
	return chunks
}
