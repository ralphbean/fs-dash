package github

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fullsend-ai/fs-dash/internal/domain"
)

func quotaHeaders(limit, used, remaining int, reset time.Time) http.Header {
	headers := make(http.Header)
	headers.Set("X-RateLimit-Limit", fmt.Sprint(limit))
	headers.Set("X-RateLimit-Used", fmt.Sprint(used))
	headers.Set("X-RateLimit-Remaining", fmt.Sprint(remaining))
	headers.Set("X-RateLimit-Reset", fmt.Sprint(reset.Unix()))
	headers.Set("X-RateLimit-Resource", "graphql")
	return headers
}

func responseWithQuota(status string, limit, used, remaining int, reset time.Time, body string) string {
	return fmt.Sprintf("HTTP/2.0 %s\r\nContent-Type: application/json; charset=utf-8\r\nX-RateLimit-Limit: %d\r\nX-RateLimit-Used: %d\r\nX-RateLimit-Remaining: %d\r\nX-RateLimit-Reset: %d\r\nX-RateLimit-Resource: graphql\r\n\r\n%s\n", status, limit, used, remaining, reset.Unix(), body)
}

func fakeGH(t *testing.T, first, next string, exitCode int) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "gh")
	var script strings.Builder
	script.WriteString("#!/bin/sh\n")
	if next != "" {
		script.WriteString("case \"$*\" in\n*'endCursor=cursor-1'*)\n")
		script.WriteString("cat <<'FS_DASH_BODY'\n" + next + "FS_DASH_BODY\n;;\n*)\n")
		script.WriteString("cat <<'FS_DASH_BODY'\n" + first + "FS_DASH_BODY\n;;\nesac\n")
	} else {
		script.WriteString("cat <<'FS_DASH_BODY'\n" + first + "FS_DASH_BODY\n")
	}
	if exitCode != 0 {
		script.WriteString(fmt.Sprintf("exit %d\n", exitCode))
	}
	if err := os.WriteFile(path, []byte(script.String()), 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestIncludedQuotaResponseParsesHeadersAndRejectsMalformedMetadata(t *testing.T) {
	now := time.Date(2026, 11, 6, 17, 0, 0, 0, time.UTC)
	reset := now.Add(time.Hour)
	output := responseWithQuota("200 OK", 5000, 125, 4875, reset, `{"data":{}}`)
	headers, body, err := parseIncludedResponse([]byte(output))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `{"data":{}}`) {
		t.Fatalf("body was not separated from the headers: %q", body)
	}
	got := parseQuotaObservation(headers, now)
	if got == nil || got.Limit != 5000 || got.Used != 125 || got.Remaining != 4875 || got.Resource != "graphql" || !got.ResetAt.Equal(reset) || !got.ObservedAt.Equal(now) {
		t.Fatalf("quota observation = %+v", got)
	}

	if _, _, err := parseIncludedResponse([]byte(`not an HTTP response`)); err == nil {
		t.Fatal("malformed status line was accepted")
	}
	malformed := http.Header{}
	malformed.Set("X-RateLimit-Limit", "5000")
	malformed.Set("X-RateLimit-Used", "not-a-number")
	malformed.Set("X-RateLimit-Remaining", "0")
	malformed.Set("X-RateLimit-Reset", fmt.Sprint(reset.Unix()))
	if got := parseQuotaObservation(malformed, now); got != nil {
		t.Fatalf("malformed quota headers produced an observation: %+v", got)
	}
}

func TestPRListerCollectsConservativeQuotaAcrossPages(t *testing.T) {
	reset := time.Now().Add(30 * time.Minute).Truncate(time.Second)
	first := responseWithQuota("200 OK", 5000, 120, 4880, reset,
		`{"data":{"viewer":{"login":"me"},"repository":{"pullRequests":{"nodes":[{"number":1,"title":"First","url":"https://github.com/o/r/pull/1","updatedAt":"2026-11-06T16:00:00Z","headRefOid":"head-1","author":{"login":"one"},"assignees":{"nodes":[],"pageInfo":{"hasNextPage":false}}}],"pageInfo":{"hasNextPage":true,"endCursor":"cursor-1"}}}}}`)
	second := responseWithQuota("200 OK", 5000, 121, 4879, reset,
		`{"data":{"viewer":{"login":"me"},"repository":{"pullRequests":{"nodes":[{"number":2,"title":"Second","url":"https://github.com/o/r/pull/2","updatedAt":"2026-11-06T15:00:00Z","headRefOid":"head-2","author":{"login":"two"},"assignees":{"nodes":[],"pageInfo":{"hasNextPage":false}}}],"pageInfo":{"hasNextPage":false,"endCursor":null}}}}}`)

	got, err := (PRLister{CLI: fakeGH(t, first, second, 0)}).ListOpenPRs(context.Background(), "o/r")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.PullRequests) != 2 || got.ViewerLogin != "me" {
		t.Fatalf("PR list = %+v", got)
	}
	if got.Quota == nil || got.Quota.Remaining != 4879 || got.Quota.Used != 121 || !got.Quota.ResetAt.Equal(reset) {
		t.Fatalf("pagination did not retain the most conservative quota observation: %+v", got.Quota)
	}
}

func TestPRListerRetainsQuotaFromRateLimitedErrorResponse(t *testing.T) {
	reset := time.Now().Add(time.Hour).Truncate(time.Second)
	body := `{"message":"API rate limit exceeded","documentation_url":"https://docs.github.com/rest/overview/resources-in-the-rest-api#rate-limiting"}`
	output := responseWithQuota("403 Forbidden", 5000, 5000, 0, reset, body)
	got, err := (PRLister{CLI: fakeGH(t, output, "", 1)}).ListOpenPRs(context.Background(), "o/r")
	if err == nil {
		t.Fatal("rate-limited request unexpectedly succeeded")
	}
	issue, ok := err.(*domain.FetchError)
	if !ok || issue.Kind != domain.ErrorRateLimit || !issue.RetryAt.Equal(reset) {
		t.Fatalf("rate-limit error = %#v", err)
	}
	if got.Quota == nil || got.Quota.Remaining != 0 {
		t.Fatalf("rate-limit quota metadata was lost: %+v", got.Quota)
	}
}

func TestEnricherCarriesQuotaFromResponse(t *testing.T) {
	now := time.Date(2026, 11, 6, 17, 0, 0, 0, time.UTC)
	reset := now.Add(time.Hour)
	body := `{"data":{"repository":{"pullRequest":{"timelineItems":{"updatedAt":"2026-11-06T16:45:00Z"}}}}}`
	cli := fakeGH(t, responseWithQuota("200 OK", 5000, 125, 4875, reset, body), "", 0)
	value, provenance, err := (Enricher{CLI: cli, Now: func() time.Time { return now }}).FetchField(context.Background(), domain.FieldKey{
		Item: domain.ItemID{Source: "github", Host: "github.com", Repo: "o/r", Number: 1}, Name: "last_touched", Revision: "head",
	})
	if err != nil {
		t.Fatal(err)
	}
	if activity, ok := value.(domain.TimelineActivity); !ok || activity.UpdatedAt.IsZero() {
		t.Fatalf("timeline value = %#v", value)
	}
	if provenance.Quota == nil || provenance.Quota.Remaining != 4875 || !provenance.Quota.ObservedAt.Equal(now) {
		t.Fatalf("quota was not carried in provenance: %+v", provenance)
	}
}
