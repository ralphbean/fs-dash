package ui

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/fullsend-ai/fs-dash/internal/config"
	"github.com/fullsend-ai/fs-dash/internal/domain"
)

type quotaTestClock struct{ now time.Time }

func (c quotaTestClock) Now() time.Time { return c.now }

func quotaTestModel(now time.Time) Model {
	cfg := config.Default()
	cfg.Color = "never"
	return New(Options{Config: cfg, Clock: quotaTestClock{now: now}})
}

func TestQuotaObservationFlowsThroughModelAndKeepsLastValidValue(t *testing.T) {
	now := time.Date(2026, 11, 6, 17, 0, 0, 0, time.UTC)
	m := quotaTestModel(now)
	quota := &domain.QuotaObservation{Limit: 5000, Used: 250, Remaining: 4750, ResetAt: now.Add(time.Hour), Resource: "graphql", ObservedAt: now}
	m.quota = nil
	m.observeQuota(quota)
	if m.quota == nil || m.quota.Remaining != 4750 {
		t.Fatalf("valid quota was not accepted: %+v", m.quota)
	}

	for _, invalid := range []*domain.QuotaObservation{
		nil,
		{Limit: 0, Remaining: 0, ObservedAt: now},
		{Limit: 5000, Remaining: 5001, ObservedAt: now.Add(time.Minute)},
		{Limit: 5000, Remaining: 1},
		{Limit: 5000, Remaining: 1, ObservedAt: now.Add(-time.Minute)},
	} {
		m.observeQuota(invalid)
		if m.quota == nil || m.quota.Remaining != 4750 || !m.quota.ObservedAt.Equal(now) {
			t.Fatalf("invalid or older observation replaced the last valid value: %+v", m.quota)
		}
	}
}

func TestListCompletionCarriesQuotaIntoModel(t *testing.T) {
	now := time.Date(2026, 11, 6, 17, 0, 0, 0, time.UTC)
	m := New(Options{Config: config.Default(), Repos: []string{"owner/repo"}, Clock: quotaTestClock{now: now}})
	quota := &domain.QuotaObservation{Limit: 5000, Used: 250, Remaining: 4750, ResetAt: now.Add(time.Hour), Resource: "graphql", ObservedAt: now}
	next, _ := m.Update(listMsg{repo: "owner/repo", generation: m.sections[0].field.Generation, quota: quota, at: now})
	m = next.(Model)
	if m.quota == nil || m.quota.Remaining != 4750 || m.quota.Resource != "graphql" {
		t.Fatalf("list completion did not update shared quota state: %+v", m.quota)
	}
}

func TestQuotaModelRejectsLateSnapshotsAndAcceptsWindowRollover(t *testing.T) {
	now := time.Date(2026, 11, 6, 17, 0, 0, 0, time.UTC)
	m := quotaTestModel(now)
	current := &domain.QuotaObservation{Limit: 5000, Used: 1161, Remaining: 3839, ResetAt: now.Add(time.Hour), Resource: "graphql", ObservedAt: now}
	if !m.observeQuota(current) {
		t.Fatal("initial quota observation was rejected")
	}

	late := &domain.QuotaObservation{Limit: 5000, Used: 397, Remaining: 4603, ResetAt: current.ResetAt, Resource: "graphql", ObservedAt: now.Add(time.Minute)}
	if m.observeQuota(late) {
		t.Fatal("late lower-used response replaced the current quota")
	}
	if m.quota.Used != 1161 || m.quota.Remaining != 3839 {
		t.Fatalf("quota moved backward: %+v", m.quota)
	}

	equal := *current
	equal.ObservedAt = now.Add(2 * time.Minute)
	if !m.observeQuota(&equal) || !m.quota.ObservedAt.Equal(equal.ObservedAt) {
		t.Fatalf("equal counters did not refresh observation age: %+v", m.quota)
	}

	nextWindow := &domain.QuotaObservation{Limit: 5000, Used: 397, Remaining: 4603, ResetAt: current.ResetAt.Add(time.Hour), Resource: "graphql", ObservedAt: now.Add(3 * time.Minute)}
	if !m.observeQuota(nextWindow) || m.quota.Remaining != 4603 || !m.quota.ResetAt.Equal(nextWindow.ResetAt) {
		t.Fatalf("later reset window was not accepted: %+v", m.quota)
	}
}

func TestQuotaStatusStatesAtCommonAndNarrowWidths(t *testing.T) {
	now := time.Date(2026, 11, 6, 17, 0, 0, 0, time.UTC)
	cases := []struct {
		name        string
		observation *domain.QuotaObservation
		contains    string
	}{
		{name: "unknown", contains: "quota: unknown"},
		{name: "current", observation: &domain.QuotaObservation{Limit: 1000, Used: 750, Remaining: 250, ResetAt: now.Add(time.Hour), Resource: "graphql", ObservedAt: now}, contains: "CURRENT"},
		{name: "low", observation: &domain.QuotaObservation{Limit: 1000, Used: 801, Remaining: 199, ResetAt: now.Add(time.Hour), Resource: "graphql", ObservedAt: now}, contains: "LOW"},
		{name: "exhausted", observation: &domain.QuotaObservation{Limit: 1000, Used: 1000, Remaining: 0, ResetAt: now.Add(time.Hour), Resource: "graphql", ObservedAt: now}, contains: "EXHAUSTED"},
		{name: "stale", observation: &domain.QuotaObservation{Limit: 1000, Used: 700, Remaining: 300, ResetAt: now.Add(time.Hour), Resource: "graphql", ObservedAt: now.Add(-6 * time.Minute)}, contains: "STALE"},
	}

	for _, width := range []int{120, 80, 55} {
		for _, tc := range cases {
			t.Run(tc.name+"/width-"+fmt.Sprint(width), func(t *testing.T) {
				m := quotaTestModel(now)
				m.width = width
				m.visual.compact = width < 64
				m.quota = tc.observation
				line := ansi.Strip(m.quotaStatusLine())
				contains := tc.contains
				if width >= 64 && tc.observation != nil {
					contains = strings.ToLower(contains)
				}
				if !strings.Contains(line, contains) {
					t.Fatalf("quota line %q does not show %q", line, contains)
				}
				if got := ansi.StringWidth(line); got > width {
					t.Fatalf("quota line is %d cells at width %d: %q", got, width, line)
				}
				if tc.observation != nil && (!strings.Contains(line, "/1000") || !strings.Contains(line, "used")) {
					t.Fatalf("quota values are missing: %q", line)
				}
			})
		}
	}
}

func TestQuotaStatusAppearsInListAndSurvivesMissingFieldMetadata(t *testing.T) {
	now := time.Date(2026, 11, 6, 17, 0, 0, 0, time.UTC)
	cfg := config.Default()
	cfg.Color = "never"
	m := New(Options{Config: cfg, Repos: []string{"owner/repo"}, Clock: quotaTestClock{now: now}})
	key := domain.FieldKey{Item: domain.ItemID{Source: "github", Host: "github.com", Repo: "owner/repo", Number: 1}, Name: "checks", Revision: "head"}
	quota := &domain.QuotaObservation{Limit: 5000, Used: 4990, Remaining: 10, ResetAt: now.Add(time.Hour), Resource: "graphql", ObservedAt: now}
	next, _ := m.Update(fieldMsg{key: key, generation: 1, provenance: domain.Provenance{Quota: quota}, err: errors.New("request failed"), at: now})
	m = next.(Model)
	if m.quota == nil || m.quota.Remaining != 10 {
		t.Fatalf("quota from an errored field request was lost: %+v", m.quota)
	}
	view := ansi.Strip(m.listView())
	if !strings.Contains(strings.ToLower(view), "low") || !strings.Contains(view, "10/5000") {
		t.Fatalf("quota status is missing from the dashboard: %s", view)
	}

	next, _ = m.Update(fieldMsg{key: key, generation: 2, provenance: domain.Provenance{}, err: errors.New("request failed"), at: now})
	m = next.(Model)
	if m.quota == nil || m.quota.Remaining != 10 {
		t.Fatalf("missing metadata replaced the last valid quota: %+v", m.quota)
	}
}

func TestQuotaResetTextUsesRelativeRoundedUnits(t *testing.T) {
	now := time.Date(2026, 11, 6, 17, 0, 0, 0, time.UTC)
	cases := []struct {
		name    string
		after   time.Duration
		want    string
		compact string
	}{
		{name: "one second", after: time.Second, want: "reset in 1 second", compact: "reset in 1s"},
		{name: "rounded seconds", after: 1500 * time.Millisecond, want: "reset in 2 seconds", compact: "reset in 2s"},
		{name: "one minute", after: time.Minute, want: "reset in 1 minute", compact: "reset in 1m"},
		{name: "rounded minutes", after: 12*time.Minute + time.Second, want: "reset in 13 minutes", compact: "reset in 13m"},
		{name: "one hour", after: time.Hour, want: "reset in 1 hour", compact: "reset in 1h"},
		{name: "multiple hours", after: 3 * time.Hour, want: "reset in 3 hours", compact: "reset in 3h"},
		{name: "one day", after: 24 * time.Hour, want: "reset in 1 day", compact: "reset in 1d"},
		{name: "multiple days", after: 48 * time.Hour, want: "reset in 2 days", compact: "reset in 2d"},
		{name: "elapsed", after: -time.Second, want: "reset now", compact: "reset now"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resetAt := now.Add(tc.after)
			if got := quotaResetText(resetAt, now, false); got != tc.want {
				t.Errorf("normal countdown = %q, want %q", got, tc.want)
			}
			if got := quotaResetText(resetAt, now, true); got != tc.compact {
				t.Errorf("compact countdown = %q, want %q", got, tc.compact)
			}
		})
	}
}

func TestQuotaRefreshContinuesToResetThenStops(t *testing.T) {
	now := time.Date(2026, 11, 6, 17, 0, 0, 0, time.UTC)
	m := quotaTestModel(now)
	m.quota = &domain.QuotaObservation{Limit: 5000, Used: 100, Remaining: 4900, ResetAt: now.Add(2 * time.Hour), Resource: "graphql", ObservedAt: now.Add(-time.Hour)}
	if cmd := m.quotaRefreshCmd(); cmd == nil {
		t.Fatal("stale quota countdown stopped before reset")
	}
	m.opts.Clock = quotaTestClock{now: m.quota.ResetAt}
	if cmd := m.quotaRefreshCmd(); cmd != nil {
		t.Fatal("quota countdown kept refreshing after reset")
	}
}
