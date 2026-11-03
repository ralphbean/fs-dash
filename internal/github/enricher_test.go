package github

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/fullsend-ai/fs-dash/internal/domain"
)

func TestClassifyActivityRequiresTrustedCurrentStatus(t *testing.T) {
	now := time.Date(2026, 11, 3, 15, 0, 0, 0, time.UTC)
	start := now.Add(-20 * time.Minute)
	active := activityComment{Body: "<!-- fullsend:agent-status:run-42 -->\n🤖 Code · Started", CreatedAt: start, URL: "https://github.com/o/r/pull/1#issuecomment-1", Author: "fullsend-ai-coder"}
	got := classifyActivity([]activityComment{active}, true, now)
	if got.State != domain.ActivityActive || got.RunID != "run-42" || got.URL != active.URL || got.Author != active.Author || !got.StartedAt.Equal(start) {
		t.Fatalf("active evidence = %+v", got)
	}
	got = classifyActivity([]activityComment{{Body: active.Body, CreatedAt: start, Author: "human"}}, true, now)
	if got.State != domain.ActivityInactive {
		t.Fatalf("human marker was trusted: %+v", got)
	}
}

func TestClassifyActivityLatestTerminalWinsAndFreshnessIsExplicit(t *testing.T) {
	now := time.Date(2026, 11, 3, 15, 0, 0, 0, time.UTC)
	old := activityComment{Body: "<!-- fullsend:agent-status:old -->", CreatedAt: now.Add(-7 * time.Hour), URL: "old", Author: "fullsend-ai-review"}
	newer := activityComment{Body: "<!-- fullsend:agent-status:new -->\n<!-- fullsend:status:terminal -->", CreatedAt: now.Add(-time.Minute), URL: "new", Author: "fullsend-ai-review"}
	got := classifyActivity([]activityComment{newer, old}, true, now)
	if got.State != domain.ActivityInactive || got.RunID != "new" || got.URL != "new" {
		t.Fatalf("latest terminal evidence did not win: %+v", got)
	}
	got = classifyActivity([]activityComment{old}, true, now)
	if got.State != domain.ActivityStale {
		t.Fatalf("old non-terminal status = %+v, want stale", got)
	}
	if got.FreshUntil.Sub(got.FetchedAt) != fieldFreshFor {
		t.Fatalf("activity cache freshness = %s", got.FreshUntil.Sub(got.FetchedAt))
	}
}

func TestClassifyActivityDistinguishesUnknownFromInactive(t *testing.T) {
	now := time.Now()
	if got := classifyActivity(nil, true, now); got.State != domain.ActivityInactive || !got.Complete {
		t.Fatalf("complete negative scan = %+v", got)
	}
	if got := classifyActivity(nil, false, now); got.State != domain.ActivityUnknown || got.Complete {
		t.Fatalf("incomplete negative scan = %+v", got)
	}
}

func TestTriageQueriesCoverDeferredFields(t *testing.T) {
	for _, name := range []string{"reviews", "review_requests", "assignees", "checks", "review_threads", "activity"} {
		if strings.TrimSpace(fieldQueries[name]) == "" {
			t.Errorf("missing query for %s", name)
		}
	}
	for _, fragment := range []string{"reviews(last: 100)", "reviewRequests(first: 100)", "assignees(first: 100)", "statusCheckRollup", "reviewThreads(first: 100)", "comments(last: 100)"} {
		found := false
		for _, query := range fieldQueries {
			if strings.Contains(query, fragment) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("no deferred query requests %q", fragment)
		}
	}
}

func TestGraphQLTriageResponsesDecodeEachDeferredField(t *testing.T) {
	const fixture = `{"data":{"viewer":{"login":"me"},"repository":{"pullRequest":{"reviews":{"nodes":[{"databaseId":12,"author":{"login":"fullsend-ai-review","__typename":"Bot"},"state":"APPROVED","submittedAt":"2026-11-03T14:00:00Z"}],"pageInfo":{"hasPreviousPage":false}},"reviewRequests":{"nodes":[{"requestedReviewer":{"__typename":"User","login":"me"}}],"pageInfo":{"hasNextPage":false}},"assignees":{"nodes":[{"login":"me"}],"pageInfo":{"hasNextPage":false}},"commits":{"nodes":[{"commit":{"statusCheckRollup":{"contexts":{"nodes":[{"__typename":"CheckRun","status":"COMPLETED","conclusion":"FAILURE"},{"__typename":"StatusContext","state":"PENDING"}],"pageInfo":{"hasNextPage":false}}}}}]},"reviewThreads":{"nodes":[{"isResolved":false},{"isResolved":true}],"pageInfo":{"hasNextPage":false}},"comments":{"nodes":[{"body":"<!-- fullsend:agent-status:run-1 -->","createdAt":"2026-11-03T14:30:00Z","updatedAt":"2026-11-03T14:30:00Z","url":"https://github.com/o/r/pull/1#issuecomment-1","author":{"login":"fullsend-ai-coder"}}],"pageInfo":{"hasPreviousPage":false}}}}}}`
	var page graphPage
	if err := json.Unmarshal([]byte(fixture), &page); err != nil {
		t.Fatal(err)
	}
	pr := page.Data.Repository.PullRequest
	now := time.Date(2026, 11, 3, 15, 0, 0, 0, time.UTC)
	value, err := fieldValue("reviews", page.Data.Viewer.Login, pr, now)
	if err != nil {
		t.Fatal(err)
	}
	reviews := value.(domain.ReviewSet)
	if !reviews.Complete || reviews.ViewerLogin != "me" || len(reviews.Reviews) != 1 || !reviews.Reviews[0].IsBot {
		t.Fatalf("reviews = %+v", reviews)
	}
	value, err = fieldValue("review_requests", page.Data.Viewer.Login, pr, now)
	if err != nil {
		t.Fatal(err)
	}
	requests := value.(domain.LoginSet)
	if !requests.Complete || requests.ViewerLogin != "me" || len(requests.Logins) != 1 || requests.Logins[0] != "me" {
		t.Fatalf("review requests = %+v", requests)
	}
	teamFixture := strings.Replace(fixture, `"__typename":"User","login":"me"`, `"__typename":"Team","slug":"reviewers","organization":{"login":"owner"}`, 1)
	var teamPage graphPage
	if err := json.Unmarshal([]byte(teamFixture), &teamPage); err != nil {
		t.Fatal(err)
	}
	value, err = fieldValue("review_requests", teamPage.Data.Viewer.Login, teamPage.Data.Repository.PullRequest, now)
	if err != nil {
		t.Fatal(err)
	}
	if got := value.(domain.LoginSet); got.Complete || len(got.Logins) != 0 {
		t.Fatalf("team review request should remain inconclusive: %+v", got)
	}
	value, err = fieldValue("assignees", page.Data.Viewer.Login, pr, now)
	if err != nil {
		t.Fatal(err)
	}
	if got := value.(domain.LoginSet); !got.Complete || len(got.Logins) != 1 || got.Logins[0] != "me" {
		t.Fatalf("assignees = %+v", got)
	}
	value, err = fieldValue("checks", page.Data.Viewer.Login, pr, now)
	if err != nil {
		t.Fatal(err)
	}
	if got := value.(domain.CheckSummary); !got.Complete || got.Failed != 1 || got.Pending != 1 || got.Total != 2 {
		t.Fatalf("checks = %+v", got)
	}
	value, err = fieldValue("review_threads", page.Data.Viewer.Login, pr, now)
	if err != nil {
		t.Fatal(err)
	}
	if got := value.(domain.ThreadSummary); !got.Complete || got.Unresolved != 1 {
		t.Fatalf("threads = %+v", got)
	}
	value, err = fieldValue("activity", page.Data.Viewer.Login, pr, now)
	if err != nil {
		t.Fatal(err)
	}
	if got := value.(domain.ActivityEvidence); got.State != domain.ActivityActive || got.RunID != "run-1" {
		t.Fatalf("activity = %+v", got)
	}
}
