package queue

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

// fakeAutoDisableRepo records AutoDisableWebhook calls and returns result.
type fakeAutoDisableRepo struct {
	store.HealthRepository
	result      *store.AutoDisableResult
	calls       int
	minFailures int
	failingFor  time.Duration
}

func (f *fakeAutoDisableRepo) AutoDisableWebhook(ctx context.Context, webhookID uuid.UUID, minFailures int, failingFor time.Duration) (*store.AutoDisableResult, error) {
	f.calls++
	f.minFailures = minFailures
	f.failingFor = failingFor
	return f.result, nil
}

func newAutoDisableWorker(repo *fakeAutoDisableRepo, eventRepo *fakeSystemEventRepo, policy AutoDisablePolicy) *WebhookWorker {
	return &WebhookWorker{
		healthRepo:      repo,
		alertConfigRepo: &fakeAlertConfigRepo{},
		eventRepo:       eventRepo,
		jobInserter:     noopJobInserter{},
		autoDisable:     policy,
	}
}

var testPolicy = AutoDisablePolicy{After: 120 * time.Hour, MinFailures: 10}

func TestMaybeAutoDisable_DisablesAndEmits(t *testing.T) {
	repo := &fakeAutoDisableRepo{result: &store.AutoDisableResult{
		Reason:              "auto-disabled: 12 failed attempts in a row",
		ConsecutiveFailures: 12,
		FailingSince:        time.Now().Add(-6 * 24 * time.Hour),
	}}
	eventRepo := &fakeSystemEventRepo{}
	w := newAutoDisableWorker(repo, eventRepo, testPolicy)

	w.maybeAutoDisable(context.Background(), slog.Default(), uuid.New(), "acme", uuid.New(), "https://x")

	if repo.calls != 1 || repo.minFailures != 10 || repo.failingFor != 120*time.Hour {
		t.Fatalf("AutoDisableWebhook calls=%d min=%d for=%s, want 1/10/120h", repo.calls, repo.minFailures, repo.failingFor)
	}
	if len(eventRepo.stored) != 1 || eventRepo.stored[0] != systemEventWebhookDisabled {
		t.Errorf("expected one %s event, got %v", systemEventWebhookDisabled, eventRepo.stored)
	}
}

func TestMaybeAutoDisable_NothingDisabledEmitsNothing(t *testing.T) {
	repo := &fakeAutoDisableRepo{}
	eventRepo := &fakeSystemEventRepo{}
	w := newAutoDisableWorker(repo, eventRepo, testPolicy)

	w.maybeAutoDisable(context.Background(), slog.Default(), uuid.New(), "acme", uuid.New(), "https://x")

	if repo.calls != 1 {
		t.Errorf("expected the threshold check to run, got %d calls", repo.calls)
	}
	if len(eventRepo.stored) != 0 {
		t.Errorf("expected no event when the webhook stays active, got %v", eventRepo.stored)
	}
}

func TestMaybeAutoDisable_Skips(t *testing.T) {
	tests := []struct {
		name     string
		consumer string
		policy   AutoDisablePolicy
	}{
		{"feature off", "acme", AutoDisablePolicy{}},
		{"zero window", "acme", AutoDisablePolicy{MinFailures: 10}},
		{"system consumer", SystemEventConsumer, testPolicy},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeAutoDisableRepo{result: &store.AutoDisableResult{}}
			w := newAutoDisableWorker(repo, &fakeSystemEventRepo{}, tt.policy)

			w.maybeAutoDisable(context.Background(), slog.Default(), uuid.New(), tt.consumer, uuid.New(), "https://x")

			if repo.calls != 0 {
				t.Errorf("expected no threshold check, got %d calls", repo.calls)
			}
		})
	}
}

func TestHoldReason(t *testing.T) {
	now := time.Now()
	active := &store.WebhookRegistration{Active: true}
	paused := &store.WebhookRegistration{Active: false}
	autoDisabled := &store.WebhookRegistration{Active: false, AutoDisabledAt: &now}
	pausedSub := &store.EventSubscription{PausedAt: &now}
	liveSub := &store.EventSubscription{}

	tests := []struct {
		name    string
		webhook *store.WebhookRegistration
		sub     *store.EventSubscription
		want    string
	}{
		{"active webhook, live subscription", active, liveSub, ""},
		{"active webhook, no subscription", active, nil, ""},
		{"paused webhook", paused, liveSub, "webhook is paused"},
		{"auto-disabled webhook", autoDisabled, liveSub, "auto-disabled"},
		{"paused subscription", active, pausedSub, "subscription is paused"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := holdReason(tt.webhook, tt.sub)
			if tt.want == "" && got != "" {
				t.Fatalf("holdReason = %q, want none", got)
			}
			if tt.want != "" && !strings.Contains(got, tt.want) {
				t.Fatalf("holdReason = %q, want it to mention %q", got, tt.want)
			}
		})
	}
}
