package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/fullsend-ai/fs-dash/internal/domain"
)

const listQuery = `query($owner: String!, $name: String!, $endCursor: String) {
  repository(owner: $owner, name: $name) {
    pullRequests(first: 100, after: $endCursor, states: OPEN, orderBy: {field: UPDATED_AT, direction: DESC}) {
      nodes { number title url updatedAt headRefOid author { login } }
      pageInfo { hasNextPage endCursor }
    }
  }
}`

type page struct {
	Data struct {
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

type PRLister struct{ CLI string }

func (l PRLister) ListOpenPRs(ctx context.Context, repository string) ([]domain.PR, error) {
	parts := strings.Split(repository, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, &domain.FetchError{Kind: domain.ErrorPermanent, Message: "repository must be owner/name"}
	}
	cli := l.CLI
	if cli == "" {
		cli = "gh"
	}
	cmd := exec.CommandContext(ctx, cli, "api", "graphql", "--paginate", "--slurp", "-f", "query="+listQuery, "-F", "owner="+parts[0], "-F", "name="+parts[1])
	output, err := cmd.CombinedOutput()
	if err != nil {
		issue := classifyCLIError(err, string(output))
		if issue.Kind == domain.ErrorRateLimit {
			issue.RetryAt = rateLimitReset(ctx, cli)
		}
		return nil, issue
	}
	var pages []page
	if err := json.Unmarshal(output, &pages); err != nil {
		return nil, &domain.FetchError{Kind: domain.ErrorTransient, Message: "GitHub returned unreadable PR data"}
	}
	if len(pages) == 0 {
		return nil, &domain.FetchError{Kind: domain.ErrorTransient, Message: "GitHub returned no PR pages"}
	}
	prs := make([]domain.PR, 0)
	for _, p := range pages {
		if len(p.Errors) > 0 {
			issue := classifyCLIError(errors.New("GraphQL error"), p.Errors[0].Message)
			if issue.Kind == domain.ErrorRateLimit {
				issue.RetryAt = rateLimitReset(ctx, cli)
			}
			return nil, issue
		}
		if p.Data.Repository == nil {
			return nil, &domain.FetchError{Kind: domain.ErrorPermission, Message: "GitHub repository unavailable or access denied: " + repository}
		}
		for _, node := range p.Data.Repository.PullRequests.Nodes {
			author := "[deleted]"
			if node.Author != nil {
				author = node.Author.Login
			}
			prs = append(prs, domain.PR{ID: domain.ItemID{Source: "github", Host: "github.com", Repo: repository, Number: node.Number}, Title: node.Title, URL: node.URL, UpdatedAt: node.UpdatedAt, HeadSHA: node.HeadSHA, Author: author})
		}
	}
	if pages[len(pages)-1].Data.Repository.PullRequests.PageInfo.HasNextPage {
		return nil, &domain.FetchError{Kind: domain.ErrorTransient, Message: "GitHub PR pagination incomplete"}
	}
	return prs, nil
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

func rateLimitReset(ctx context.Context, cli string) time.Time {
	fallback := time.Now().Add(time.Minute)
	output, err := exec.CommandContext(ctx, cli, "api", "rate_limit", "--jq", ".resources.graphql.reset").Output()
	if err != nil {
		return fallback
	}
	seconds, err := strconv.ParseInt(strings.TrimSpace(string(output)), 10, 64)
	if err != nil {
		return fallback
	}
	reset := time.Unix(seconds, 0)
	if reset.Before(time.Now()) {
		return fallback
	}
	return reset
}
