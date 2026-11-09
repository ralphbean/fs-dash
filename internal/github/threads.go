package github

import (
	"context"
	"encoding/json"
	"sort"
	"time"

	"github.com/fullsend-ai/fs-dash/internal/domain"
)

type reviewThreadPage struct {
	ReviewThreads struct {
		Nodes []struct {
			ID           string `json:"id"`
			IsResolved   bool   `json:"isResolved"`
			Path         string `json:"path"`
			Line         *int   `json:"line"`
			OriginalLine *int   `json:"originalLine"`
			Comments     struct {
				Nodes []struct {
					ID        string    `json:"id"`
					Body      string    `json:"body"`
					CreatedAt time.Time `json:"createdAt"`
					URL       string    `json:"url"`
					Author    *struct {
						Login string `json:"login"`
					} `json:"author"`
				} `json:"nodes"`
				PageInfo struct {
					HasPreviousPage bool `json:"hasPreviousPage"`
				} `json:"pageInfo"`
			} `json:"comments"`
		} `json:"nodes"`
		PageInfo struct {
			HasNextPage bool   `json:"hasNextPage"`
			EndCursor   string `json:"endCursor"`
		} `json:"pageInfo"`
	} `json:"reviewThreads"`
}

func (c DetailClient) fetchThreadDetails(ctx context.Context, key domain.DetailKey) (any, domain.Provenance, error) {
	value := domain.ReviewThreadList{Complete: true}
	cursor := ""
	seen := map[string]struct{}{}
	var provenance domain.Provenance
	for page := 0; page < 500; page++ {
		variables := map[string]string{}
		if cursor != "" {
			variables["cursor"] = cursor
		}
		raw, currentProvenance, err := c.graphQL(ctx, key.Item, detailThreadsQuery, variables, false, true)
		if err != nil {
			return nil, currentProvenance, err
		}
		provenance = currentProvenance
		var result reviewThreadPage
		if err := json.Unmarshal(raw, &result); err != nil {
			return nil, provenance, &domain.FetchError{Kind: domain.ErrorTransient, Message: "GitHub returned malformed review thread data"}
		}
		for _, rawThread := range result.ReviewThreads.Nodes {
			if rawThread.IsResolved {
				continue
			}
			thread := domain.ReviewThread{ID: rawThread.ID, Path: rawThread.Path, IsOutdated: rawThread.Line == nil || *rawThread.Line == 0, CommentsComplete: !rawThread.Comments.PageInfo.HasPreviousPage}
			if rawThread.Line != nil {
				thread.Line = *rawThread.Line
			}
			if rawThread.OriginalLine != nil {
				thread.OriginalLine = *rawThread.OriginalLine
			}
			for _, comment := range rawThread.Comments.Nodes {
				author := "[deleted]"
				if comment.Author != nil {
					author = comment.Author.Login
				}
				thread.Comments = append(thread.Comments, domain.ReviewThreadComment{ID: comment.ID, Body: comment.Body, Author: author, CreatedAt: comment.CreatedAt, URL: comment.URL})
			}
			value.Threads = append(value.Threads, thread)
		}
		if !result.ReviewThreads.PageInfo.HasNextPage {
			value.Complete = true
			break
		}
		nextCursor := result.ReviewThreads.PageInfo.EndCursor
		if nextCursor == "" {
			return nil, provenance, &domain.FetchError{Kind: domain.ErrorTransient, Message: "GitHub review thread pagination incomplete"}
		}
		if _, duplicate := seen[nextCursor]; duplicate {
			return nil, provenance, &domain.FetchError{Kind: domain.ErrorTransient, Message: "GitHub review thread pagination did not advance"}
		}
		seen[nextCursor] = struct{}{}
		cursor = nextCursor
		value.Complete = false
		if page == 499 {
			return nil, provenance, &domain.FetchError{Kind: domain.ErrorTransient, Message: "GitHub review thread page limit reached"}
		}
	}
	sort.SliceStable(value.Threads, func(i, j int) bool {
		if value.Threads[i].Path != value.Threads[j].Path {
			return value.Threads[i].Path < value.Threads[j].Path
		}
		line := func(thread domain.ReviewThread) int {
			if thread.Line > 0 {
				return thread.Line
			}
			return thread.OriginalLine
		}
		if line(value.Threads[i]) != line(value.Threads[j]) {
			return line(value.Threads[i]) < line(value.Threads[j])
		}
		return value.Threads[i].ID < value.Threads[j].ID
	})
	return value, provenance, nil
}
