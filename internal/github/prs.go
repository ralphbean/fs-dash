package github

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/fullsend-ai/fs-dash/internal/domain"
	"github.com/fullsend-ai/fs-dash/internal/fetch"
)

const listQuery = `query($owner: String!, $name: String!, $endCursor: String) {
  viewer { login }
  repository(owner: $owner, name: $name) {
    pullRequests(first: 100, after: $endCursor, states: OPEN, orderBy: {field: UPDATED_AT, direction: DESC}) {
      nodes { number title url updatedAt headRefOid author { login } assignees(first: 20) { nodes { login } pageInfo { hasNextPage } } }
      pageInfo { hasNextPage endCursor }
    }
  }
}`

type page struct {
	Data struct {
		Viewer struct {
			Login string `json:"login"`
		} `json:"viewer"`
		Repository *struct {
			PullRequests struct {
				Nodes []struct {
					Number    int       `json:"number"`
					Title     string    `json:"title"`
					URL       string    `json:"url"`
					UpdatedAt time.Time `json:"updatedAt"`
					HeadSHA   string    `json:"headRefOid"`
					Author    *struct {
						Login string `json:"login"`
					} `json:"author"`
					Assignees *struct {
						Nodes []struct {
							Login string `json:"login"`
						} `json:"nodes"`
						PageInfo struct {
							HasNextPage bool `json:"hasNextPage"`
						} `json:"pageInfo"`
					} `json:"assignees"`
				} `json:"nodes"`
				PageInfo struct {
					HasNextPage bool   `json:"hasNextPage"`
					EndCursor   string `json:"endCursor"`
				} `json:"pageInfo"`
			} `json:"pullRequests"`
		} `json:"repository"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

type PRLister struct {
	CLI  string
	Gate *fetch.RateLimitGate
}

func (l PRLister) ListOpenPRs(ctx context.Context, repository string) (domain.PRList, error) {
	parts := strings.Split(repository, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return domain.PRList{}, &domain.FetchError{Kind: domain.ErrorPermanent, Message: "repository must be owner/name"}
	}
	cli := l.CLI
	if cli == "" {
		cli = "gh"
	}
	result := domain.PRList{PullRequests: make([]domain.PR, 0)}
	cursor := ""
	seenCursors := map[string]struct{}{}
	for {
		if l.Gate != nil {
			if err := l.Gate.Wait(ctx); err != nil {
				return result, fetch.Classify(err)
			}
		}
		args := []string{"api", "graphql", "--include", "-f", "query=" + listQuery, "-F", "owner=" + parts[0], "-F", "name=" + parts[1]}
		if cursor != "" {
			args = append(args, "-F", "endCursor="+cursor)
		}
		output, commandErr := exec.CommandContext(ctx, cli, args...).CombinedOutput()
		observedAt := time.Now()
		headers, body, responseErr := parseIncludedResponse(output)
		observation := parseQuotaObservation(headers, observedAt)
		result.Quota = conservativeQuota(result.Quota, observation)
		if commandErr != nil {
			issue := classifyCLIError(commandErr, string(output))
			applyRateLimitMetadata(l.Gate, issue, headers, observation, observedAt)
			return result, issue
		}
		if responseErr != nil {
			applyRateLimitMetadata(l.Gate, nil, headers, observation, observedAt)
			return result, &domain.FetchError{Kind: domain.ErrorTransient, Message: "GitHub returned unreadable response metadata"}
		}
		var response page
		if err := json.NewDecoder(bytes.NewReader(body)).Decode(&response); err != nil {
			applyRateLimitMetadata(l.Gate, nil, headers, observation, observedAt)
			return result, &domain.FetchError{Kind: domain.ErrorTransient, Message: "GitHub returned unreadable PR data"}
		}
		if len(response.Errors) > 0 {
			issue := classifyCLIError(errors.New("GraphQL error"), response.Errors[0].Message)
			applyRateLimitMetadata(l.Gate, issue, headers, observation, observedAt)
			return result, issue
		}
		applyRateLimitMetadata(l.Gate, nil, headers, observation, observedAt)
		if response.Data.Repository == nil {
			return result, &domain.FetchError{Kind: domain.ErrorPermission, Message: "GitHub repository unavailable or access denied: " + repository}
		}
		if response.Data.Viewer.Login != "" {
			result.ViewerLogin = response.Data.Viewer.Login
		}
		for _, node := range response.Data.Repository.PullRequests.Nodes {
			author := "[deleted]"
			if node.Author != nil {
				author = node.Author.Login
			}
			assignees := domain.AssigneeSet{}
			if node.Assignees != nil {
				for _, assignee := range node.Assignees.Nodes {
					assignees.Logins = append(assignees.Logins, assignee.Login)
				}
				assignees.Complete = !node.Assignees.PageInfo.HasNextPage
			}
			result.PullRequests = append(result.PullRequests, domain.PR{ID: domain.ItemID{Source: "github", Host: "github.com", Repo: repository, Number: node.Number}, Title: node.Title, URL: node.URL, UpdatedAt: node.UpdatedAt, HeadSHA: node.HeadSHA, Author: author, Assignees: assignees})
		}
		pullRequests := response.Data.Repository.PullRequests
		if !pullRequests.PageInfo.HasNextPage {
			return result, nil
		}
		if pullRequests.PageInfo.EndCursor == "" {
			return result, &domain.FetchError{Kind: domain.ErrorTransient, Message: "GitHub PR pagination incomplete"}
		}
		if _, exists := seenCursors[pullRequests.PageInfo.EndCursor]; exists {
			return result, &domain.FetchError{Kind: domain.ErrorTransient, Message: "GitHub PR pagination did not advance"}
		}
		seenCursors[pullRequests.PageInfo.EndCursor] = struct{}{}
		cursor = pullRequests.PageInfo.EndCursor
	}
}

func parseIncludedResponse(output []byte) (http.Header, []byte, error) {
	reader := bufio.NewReader(bytes.NewReader(output))
	statusLine, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, nil, err
	}
	if !strings.HasPrefix(strings.TrimSpace(statusLine), "HTTP/") {
		return nil, nil, errors.New("HTTP status line missing")
	}
	header := make(http.Header)
	for {
		line, readErr := reader.ReadString('\n')
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			return nil, nil, readErr
		}
		line = strings.TrimSpace(line)
		if line == "" {
			body, _ := io.ReadAll(reader)
			return header, body, nil
		}
		name, value, ok := strings.Cut(line, ":")
		if !ok || strings.TrimSpace(name) == "" {
			return nil, nil, errors.New("malformed HTTP response header")
		}
		header.Add(strings.TrimSpace(name), strings.TrimSpace(value))
		if errors.Is(readErr, io.EOF) {
			return nil, nil, errors.New("HTTP response body missing")
		}
	}
}

func parseQuotaObservation(headers http.Header, observedAt time.Time) *domain.QuotaObservation {
	parseInt := func(name string) (int, bool) {
		value, err := strconv.Atoi(strings.TrimSpace(headers.Get(name)))
		return value, err == nil
	}
	limit, hasLimit := parseInt("x-ratelimit-limit")
	used, hasUsed := parseInt("x-ratelimit-used")
	remaining, hasRemaining := parseInt("x-ratelimit-remaining")
	resetSeconds, hasReset := parseInt("x-ratelimit-reset")
	if !hasLimit || !hasUsed || !hasRemaining || !hasReset || limit <= 0 || used < 0 || used > limit || remaining < 0 || remaining > limit || resetSeconds <= 0 {
		return nil
	}
	return &domain.QuotaObservation{
		Limit:      limit,
		Used:       used,
		Remaining:  remaining,
		ResetAt:    time.Unix(int64(resetSeconds), 0),
		Resource:   strings.TrimSpace(headers.Get("x-ratelimit-resource")),
		ObservedAt: observedAt,
	}
}

func conservativeQuota(current, candidate *domain.QuotaObservation) *domain.QuotaObservation {
	if candidate == nil {
		return current
	}
	if current == nil || candidate.Remaining < current.Remaining || (candidate.Remaining == current.Remaining && candidate.ObservedAt.After(current.ObservedAt)) {
		return candidate
	}
	return current
}

func applyQuotaRetry(issue *domain.FetchError, observation *domain.QuotaObservation) {
	if issue != nil && issue.Kind == domain.ErrorRateLimit && observation != nil && observation.Remaining == 0 {
		issue.RetryAt = observation.ResetAt
	}
}

func parseRetryAfter(headers http.Header, now time.Time) time.Duration {
	value := strings.TrimSpace(headers.Get("retry-after"))
	if value == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(value); err == nil {
		if seconds <= 0 {
			return 0
		}
		return time.Duration(seconds) * time.Second
	}
	if at, err := http.ParseTime(value); err == nil && at.After(now) {
		return at.Sub(now)
	}
	return 0
}

func applyRateLimitMetadata(gate *fetch.RateLimitGate, issue *domain.FetchError, headers http.Header, quota *domain.QuotaObservation, now time.Time) {
	retryAfter := parseRetryAfter(headers, now)
	if issue != nil && issue.Kind == domain.ErrorRateLimit {
		issue.RetryAfter = retryAfter
		switch {
		case quota != nil && quota.Remaining == 0 && quota.ResetAt.After(now):
			issue.RetryAt = quota.ResetAt
		case retryAfter > 0:
			issue.RetryAt = now.Add(retryAfter)
		case gate == nil:
			issue.RetryAt = now.Add(time.Minute)
		}
	}
	if gate != nil {
		gate.ObserveResponse(issue, quota, retryAfter)
	}
}

var secretPattern = regexp.MustCompile(`(?i)(gh[opusr]_[a-z0-9]+|github_pat_[a-z0-9_]+|authorization:\s*\S+)`)

func sanitize(message string) string {
	message = secretPattern.ReplaceAllString(message, "[REDACTED_SECRET]")
	message = strings.TrimSpace(message)
	if len(message) > 300 {
		message = message[:300] + "…"
	}
	return message
}

func classifyCLIError(err error, output string) *domain.FetchError {
	if errors.Is(err, exec.ErrNotFound) {
		return &domain.FetchError{Kind: domain.ErrorAuth, Message: "gh CLI is not installed; install and authenticate gh"}
	}
	if errors.Is(err, context.Canceled) {
		return &domain.FetchError{Kind: domain.ErrorCanceled, Message: "GitHub request canceled"}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return &domain.FetchError{Kind: domain.ErrorTransient, Message: "GitHub request timed out"}
	}
	lower := strings.ToLower(output)
	switch {
	case strings.Contains(lower, "authentication") || strings.Contains(lower, "not logged") || strings.Contains(lower, "http 401"):
		return &domain.FetchError{Kind: domain.ErrorAuth, Message: "GitHub authentication failed; run gh auth login"}
	case strings.Contains(lower, "rate limit"):
		return &domain.FetchError{Kind: domain.ErrorRateLimit, Message: "GitHub rate limit reached; retry later"}
	case strings.Contains(lower, "http 403") || strings.Contains(lower, "resource not accessible"):
		return &domain.FetchError{Kind: domain.ErrorPermission, Message: "GitHub access denied; check repository permissions"}
	case strings.Contains(lower, "could not resolve to a repository") || strings.Contains(lower, "not found"):
		return &domain.FetchError{Kind: domain.ErrorPermanent, Message: "GitHub repository not found; check owner/name and access"}
	case strings.Contains(lower, "http 502") || strings.Contains(lower, "http 503") || strings.Contains(lower, "http 504") || strings.Contains(lower, "bad gateway") || strings.Contains(lower, "gateway timeout"):
		return &domain.FetchError{Kind: domain.ErrorTransient, Message: "GitHub is temporarily unavailable"}
	default:
		return &domain.FetchError{Kind: domain.ErrorTransient, Message: fmt.Sprintf("GitHub request failed: %s", sanitize(output))}
	}
}
