package github

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fullsend-ai/fs-dash/internal/domain"
)

func detailCLI(t *testing.T, script string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "gh")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+script), 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestFetchDetailReviewThreadsKeepsHistoryAndOutdatedLocations(t *testing.T) {
	now := time.Now().UTC()
	body := `{"data":{"repository":{"pullRequest":{"reviewThreads":{"nodes":[{"id":"thread-b","isResolved":false,"path":"z.go","line":null,"originalLine":14,"comments":{"nodes":[{"id":"comment-b","body":"old location","createdAt":"2026-11-09T13:00:00Z","url":"https://github.com/o/r/pull/5#discussion_r2","author":{"login":"reviewer"}}],"pageInfo":{"hasNextPage":false}}},{"id":"thread-a","isResolved":false,"path":"a.go","line":4,"originalLine":4,"comments":{"nodes":[{"id":"comment-a","body":"please update","createdAt":"2026-11-09T12:00:00Z","url":"https://github.com/o/r/pull/5#discussion_r1","author":{"login":"reviewer"}}],"pageInfo":{"hasNextPage":false}}},{"id":"resolved","isResolved":true,"path":"b.go","line":2,"originalLine":2,"comments":{"nodes":[],"pageInfo":{"hasNextPage":false}}}],"pageInfo":{"hasNextPage":false}}}}}}`
	cli := detailCLI(t, fmt.Sprintf("cat <<'FS_DASH_RESPONSE'\n%sFS_DASH_RESPONSE\n", responseWithQuota("200 OK", 5000, 15, 4985, now.Add(time.Hour), body)))
	item := domain.ItemID{Source: "github", Host: "github.com", Repo: "owner/repo", Number: 5}
	value, provenance, err := (DetailClient{CLI: cli, Now: func() time.Time { return now }}).FetchDetail(context.Background(), domain.DetailKey{Item: item, Group: "threads", Revision: "head"})
	if err != nil {
		t.Fatal(err)
	}
	threads := value.(domain.ReviewThreadList)
	if !threads.Complete || len(threads.Threads) != 2 || threads.Threads[0].ID != "thread-a" || threads.Threads[1].ID != "thread-b" {
		t.Fatalf("thread list/order = %+v", threads)
	}
	if !threads.Threads[1].IsOutdated || threads.Threads[1].OriginalLine != 14 || threads.Threads[1].Comments[0].Body != "old location" {
		t.Fatalf("outdated thread history = %+v", threads.Threads[1])
	}
	if provenance.Quota == nil || provenance.FreshUntil.Sub(provenance.FetchedAt) != fieldFreshFor {
		t.Fatalf("detail provenance = %+v", provenance)
	}
}

func TestFetchDetailPaginatesAllReviewThreads(t *testing.T) {
	now := time.Now().UTC()
	callFile := filepath.Join(t.TempDir(), "calls")
	first := responseWithQuota("200 OK", 5000, 10, 4990, now.Add(time.Hour), `{"data":{"repository":{"pullRequest":{"reviewThreads":{"nodes":[{"id":"first","isResolved":false,"path":"a.go","line":1,"originalLine":1,"comments":{"nodes":[],"pageInfo":{"hasPreviousPage":false}}}],"pageInfo":{"hasNextPage":true,"endCursor":"cursor-2"}}}}}}`)
	second := responseWithQuota("200 OK", 5000, 11, 4989, now.Add(time.Hour), `{"data":{"repository":{"pullRequest":{"reviewThreads":{"nodes":[{"id":"second","isResolved":false,"path":"b.go","line":2,"originalLine":2,"comments":{"nodes":[],"pageInfo":{"hasPreviousPage":false}}}],"pageInfo":{"hasNextPage":false,"endCursor":null}}}}}}`)
	firstFile, secondFile := filepath.Join(t.TempDir(), "first"), filepath.Join(t.TempDir(), "second")
	if err := os.WriteFile(firstFile, []byte(first), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(secondFile, []byte(second), 0o600); err != nil {
		t.Fatal(err)
	}
	script := fmt.Sprintf("printf '%%s\\n' \"$*\" >> %q\ncase \"$*\" in *cursor=cursor-2*) cat %q ;; *) cat %q ;; esac\n", callFile, secondFile, firstFile)
	item := domain.ItemID{Source: "github", Host: "github.com", Repo: "owner/repo", Number: 5}
	value, _, err := (DetailClient{CLI: detailCLI(t, script), Now: func() time.Time { return now }}).FetchDetail(context.Background(), domain.DetailKey{Item: item, Group: "threads", Revision: "head"})
	if err != nil {
		t.Fatal(err)
	}
	threads := value.(domain.ReviewThreadList)
	if !threads.Complete || len(threads.Threads) != 2 || threads.Threads[0].ID != "first" || threads.Threads[1].ID != "second" {
		t.Fatalf("paginated thread list = %+v", threads)
	}
	calls, err := os.ReadFile(callFile)
	if err != nil || strings.Count(string(calls), "api graphql") != 2 {
		t.Fatalf("pagination requests = %q, err=%v", calls, err)
	}
}

func TestPostPRCommentSendsOnlyDeclaredMutationVariables(t *testing.T) {
	now := time.Now().UTC()
	callFile := filepath.Join(t.TempDir(), "calls")
	resolve := responseWithQuota("200 OK", 5000, 5, 4995, now.Add(time.Hour), `{"data":{"repository":{"pullRequest":{"id":"PR_node_5"}}}}`)
	posted := responseWithQuota("200 OK", 5000, 6, 4994, now.Add(time.Hour), `{"data":{"addComment":{"commentEdge":{"node":{"id":"IC_comment_1","body":"hello"}}}}}`)
	resolveFile, postedFile := filepath.Join(t.TempDir(), "resolve"), filepath.Join(t.TempDir(), "posted")
	if err := os.WriteFile(resolveFile, []byte(resolve), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(postedFile, []byte(posted), 0o600); err != nil {
		t.Fatal(err)
	}
	script := fmt.Sprintf("for arg do case \"$arg\" in owner=*|name=*|number=*|subject=*|body=*) printf '%%s\\n' \"$arg\" >> %q ;; esac; done\nprintf '\\n' >> %q\ncase \"$*\" in\n  *subjectId*) cat %q ;;\n  *) cat %q ;;\nesac\n", callFile, callFile, postedFile, resolveFile)
	cli := detailCLI(t, script)
	item := domain.ItemID{Source: "github", Host: "github.com", Repo: "owner/repo", Number: 5}
	id, err := (DetailClient{CLI: cli, Now: func() time.Time { return now }}).PostPRComment(context.Background(), item, "hello")
	if err != nil || id != "IC_comment_1" {
		t.Fatalf("comment result = %q, %v", id, err)
	}
	calls, err := os.ReadFile(callFile)
	if err != nil {
		t.Fatal(err)
	}
	callBlocks := strings.Split(strings.TrimSpace(string(calls)), "\n\n")
	if len(callBlocks) != 2 {
		t.Fatalf("GraphQL calls = %q", calls)
	}
	if !strings.Contains(callBlocks[0], "owner=owner") || !strings.Contains(callBlocks[0], "number=5") {
		t.Fatalf("PR ID query missing repository variables: %s", callBlocks[0])
	}
	if !strings.Contains(callBlocks[1], "body=hello") || strings.Contains(callBlocks[1], "owner=") || strings.Contains(callBlocks[1], "number=") {
		t.Fatalf("mutation variables = %s", callBlocks[1])
	}
}

func TestMergeInspectsBranchRulesAndUsesQueuePathAtCurrentHead(t *testing.T) {
	now := time.Now().UTC()
	callFile := filepath.Join(t.TempDir(), "calls")
	graphql := responseWithQuota("200 OK", 5000, 5, 4995, now.Add(time.Hour), `{"data":{"repository":{"mergeCommitAllowed":true,"squashMergeAllowed":true,"rebaseMergeAllowed":true,"pullRequest":{"id":"PR_node_5","headRefOid":"head-current","state":"OPEN","isDraft":false,"mergeable":"MERGEABLE","reviewDecision":"APPROVED","autoMergeRequest":null,"mergeQueueEntry":null,"baseRefName":"feature/branch"}}}}`)
	graphqlFile := filepath.Join(t.TempDir(), "graphql")
	if err := os.WriteFile(graphqlFile, []byte(graphql), 0o600); err != nil {
		t.Fatal(err)
	}
	repoResponse := responseWithQuota("200 OK", 5000, 6, 4994, now.Add(time.Hour), `{"allow_merge_commit":true,"allow_squash_merge":true,"allow_rebase_merge":true}`)
	rulesResponse := responseWithQuota("200 OK", 5000, 7, 4993, now.Add(time.Hour), `[{"type":"merge_queue"}]`)
	repoFile, rulesFile := filepath.Join(t.TempDir(), "repo"), filepath.Join(t.TempDir(), "rules")
	if err := os.WriteFile(repoFile, []byte(repoResponse), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rulesFile, []byte(rulesResponse), 0o600); err != nil {
		t.Fatal(err)
	}
	script := fmt.Sprintf("printf '%%s\\n' \"$*\" >> %q\ncase \"$*\" in\n  *graphql*) cat %q ;;\n  *rules/branches*) cat %q ;;\n  *repos/owner/repo*) cat %q ;;\n  *'pr merge'*) exit 0 ;;\n  *) exit 1 ;;\nesac\n", callFile, graphqlFile, rulesFile, repoFile)
	cli := detailCLI(t, script)
	item := domain.ItemID{Source: "github", Host: "github.com", Repo: "owner/repo", Number: 5}
	client := DetailClient{CLI: cli, Now: func() time.Time { return now }}
	state, err := client.InspectPR(context.Background(), item)
	if err != nil {
		t.Fatal(err)
	}
	if !state.MergeQueueKnown || !state.MergeQueueRequired || state.HeadSHA != "head-current" {
		t.Fatalf("merge capability = %+v", state)
	}
	if err := client.MergePR(context.Background(), item, "head-current", state); err != nil {
		t.Fatal(err)
	}
	calls, err := os.ReadFile(callFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(calls), "rules/branches/feature%2Fbranch") {
		t.Fatalf("branch policy request did not escape slash: %s", calls)
	}
	mergeLine := ""
	for _, line := range strings.Split(string(calls), "\n") {
		if strings.Contains(line, "pr merge") {
			mergeLine = line
		}
	}
	if mergeLine == "" || !strings.Contains(mergeLine, "--match-head-commit head-current") || !strings.Contains(mergeLine, "--auto") || strings.Contains(mergeLine, "--squash") {
		t.Fatalf("queue merge command = %s", mergeLine)
	}
}
