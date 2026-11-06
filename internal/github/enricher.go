package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/fullsend-ai/fs-dash/internal/domain"
)

const reviewQuery = `query($owner: String!, $name: String!, $number: Int!) {
  viewer { login }
  repository(owner: $owner, name: $name) {
    pullRequest(number: $number) {
      reviews(last: 100) {
        nodes { databaseId author { login __typename } state submittedAt }
        pageInfo { hasPreviousPage }
      }
    }
  }
}`

const reviewRequestQuery = `query($owner: String!, $name: String!, $number: Int!) {
  viewer { login }
  repository(owner: $owner, name: $name) {
    pullRequest(number: $number) {
      reviewRequests(first: 100) {
        nodes { requestedReviewer { __typename ... on User { login } ... on Team { slug organization { login } } } }
        pageInfo { hasNextPage }
      }
    }
  }
}`

const assigneeQuery = `query($owner: String!, $name: String!, $number: Int!) {
  viewer { login }
  repository(owner: $owner, name: $name) {
    pullRequest(number: $number) {
      assignees(first: 100) { nodes { login } pageInfo { hasNextPage } }
    }
  }
}`

const checksQuery = `query($owner: String!, $name: String!, $number: Int!) {
  repository(owner: $owner, name: $name) {
    pullRequest(number: $number) {
      commits(last: 1) { nodes { commit { statusCheckRollup {
        contexts(first: 100) {
          nodes { __typename ... on CheckRun { status conclusion } ... on StatusContext { state } }
          pageInfo { hasNextPage }
        }
      } } } }
    }
  }
}`

const threadQuery = `query($owner: String!, $name: String!, $number: Int!) {
  repository(owner: $owner, name: $name) {
    pullRequest(number: $number) {
      reviewThreads(first: 100) { nodes { isResolved } pageInfo { hasNextPage } }
    }
  }
}`

const activityQuery = `query($owner: String!, $name: String!, $number: Int!) {
  repository(owner: $owner, name: $name) {
    pullRequest(number: $number) {
      comments(last: 100) {
        nodes { body createdAt updatedAt url author { login } }
        pageInfo { hasPreviousPage }
      }
    }
  }
}`

const lastTouchedQuery = `query($owner: String!, $name: String!, $number: Int!) {
  repository(owner: $owner, name: $name) {
    pullRequest(number: $number) {
      timelineItems { updatedAt }
    }
  }
}`

type Enricher struct {
	CLI string
	Now func() time.Time
}

type graphPage struct {
	Data struct {
		Viewer struct {
			Login string `json:"login"`
		} `json:"viewer"`
		Repository *struct {
			PullRequest *struct {
				TimelineItems struct {
					UpdatedAt time.Time `json:"updatedAt"`
				} `json:"timelineItems"`
				Reviews struct {
					Nodes []struct {
						DatabaseID int64 `json:"databaseId"`
						Author     *struct {
							Login    string `json:"login"`
							Typename string `json:"__typename"`
						} `json:"author"`
						State       string     `json:"state"`
						SubmittedAt *time.Time `json:"submittedAt"`
					} `json:"nodes"`
					PageInfo struct {
						HasPreviousPage bool `json:"hasPreviousPage"`
					} `json:"pageInfo"`
				} `json:"reviews"`
				ReviewRequests struct {
					Nodes []struct {
						RequestedReviewer *struct {
							Typename     string `json:"__typename"`
							Login        string `json:"login"`
							Slug         string `json:"slug"`
							Organization *struct {
								Login string `json:"login"`
							} `json:"organization"`
						} `json:"requestedReviewer"`
					} `json:"nodes"`
					PageInfo struct {
						HasNextPage bool `json:"hasNextPage"`
					} `json:"pageInfo"`
				} `json:"reviewRequests"`
				Assignees struct {
					Nodes []struct {
						Login string `json:"login"`
					} `json:"nodes"`
					PageInfo struct {
						HasNextPage bool `json:"hasNextPage"`
					} `json:"pageInfo"`
				} `json:"assignees"`
				Commits struct {
					Nodes []struct {
						Commit struct {
							StatusCheckRollup *struct {
								Contexts struct {
									Nodes []struct {
										Typename   string `json:"__typename"`
										Status     string `json:"status"`
										Conclusion string `json:"conclusion"`
										State      string `json:"state"`
									} `json:"nodes"`
									PageInfo struct {
										HasNextPage bool `json:"hasNextPage"`
									} `json:"pageInfo"`
								} `json:"contexts"`
							} `json:"statusCheckRollup"`
						} `json:"commit"`
					} `json:"nodes"`
				} `json:"commits"`
				ReviewThreads struct {
					Nodes []struct {
						IsResolved bool `json:"isResolved"`
					} `json:"nodes"`
					PageInfo struct {
						HasNextPage bool `json:"hasNextPage"`
					} `json:"pageInfo"`
				} `json:"reviewThreads"`
				Comments struct {
					Nodes []struct {
						Body      string    `json:"body"`
						CreatedAt time.Time `json:"createdAt"`
						UpdatedAt time.Time `json:"updatedAt"`
						URL       string    `json:"url"`
						Author    *struct {
							Login string `json:"login"`
						} `json:"author"`
					} `json:"nodes"`
					PageInfo struct {
						HasPreviousPage bool `json:"hasPreviousPage"`
					} `json:"pageInfo"`
				} `json:"comments"`
			} `json:"pullRequest"`
		} `json:"repository"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

var fullsendStatusMarker = regexp.MustCompile(`<!-- fullsend:agent-status:([^ >]+) -->`)

var fullsendAgentLogins = map[string]struct{}{
	"fullsend-ai-review":     {},
	"fullsend-ai-coder":      {},
	"fullsend-ai-triage":     {},
	"fullsend-ai-retro":      {},
	"fullsend-ai-prioritize": {},
}

const (
	fullsendTerminalMarker = "<!-- fullsend:status:terminal -->"
	activityStaleAfter     = 6 * time.Hour
	fieldFreshFor          = 2 * time.Minute
)

func (e Enricher) FetchField(ctx context.Context, key domain.FieldKey) (any, domain.Provenance, error) {
	owner, name, err := splitRepo(key.Item.Repo)
	if err != nil {
		return nil, domain.Provenance{}, err
	}
	query, ok := fieldQueries[key.Name]
	if !ok {
		return nil, domain.Provenance{}, &domain.FetchError{Kind: domain.ErrorPermanent, Message: "unsupported GitHub field: " + key.Name}
	}
	cli := e.CLI
	if cli == "" {
		cli = "gh"
	}
	cmd := exec.CommandContext(ctx, cli, "api", "graphql", "--include", "-f", "query="+query,
		"-F", "owner="+owner, "-F", "name="+name, "-F", fmt.Sprintf("number=%d", key.Item.Number))
	output, err := cmd.CombinedOutput()
	observedAt := time.Now()
	if e.Now != nil {
		observedAt = e.Now()
	}
	headers, body, responseErr := parseIncludedResponse(output)
	quota := parseQuotaObservation(headers, observedAt)
	provenance := domain.Provenance{Source: "github", Quota: quota}
	if err != nil {
		issue := classifyCLIError(err, string(output))
		applyQuotaRetry(issue, quota)
		return nil, provenance, issue
	}
	if responseErr != nil {
		return nil, provenance, &domain.FetchError{Kind: domain.ErrorTransient, Message: "GitHub returned unreadable response metadata"}
	}
	var page graphPage
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&page); err != nil {
		return nil, provenance, &domain.FetchError{Kind: domain.ErrorTransient, Message: "GitHub returned unreadable triage data"}
	}
	if len(page.Errors) > 0 {
		issue := classifyCLIError(fmt.Errorf("GraphQL error"), page.Errors[0].Message)
		applyQuotaRetry(issue, quota)
		return nil, provenance, issue
	}
	if page.Data.Repository == nil || page.Data.Repository.PullRequest == nil {
		return nil, provenance, &domain.FetchError{Kind: domain.ErrorPermission, Message: "GitHub pull request unavailable or access denied"}
	}
	pr := page.Data.Repository.PullRequest
	now := observedAt
	var value any
	if key.Name == "last_touched" {
		if pr.TimelineItems.UpdatedAt.IsZero() {
			return nil, provenance, &domain.FetchError{Kind: domain.ErrorTransient, Message: "GitHub timeline timestamp unavailable"}
		}
		value = domain.TimelineActivity{UpdatedAt: pr.TimelineItems.UpdatedAt}
	} else {
		value, err = fieldValue(key.Name, page.Data.Viewer.Login, pr, now)
		if err != nil {
			return nil, provenance, err
		}
	}
	provenance.FetchedAt = now
	provenance.Version = now.Format(time.RFC3339Nano)
	provenance.FreshUntil = now.Add(fieldFreshFor)
	return value, provenance, nil
}

var fieldQueries = map[string]string{
	"reviews":         reviewQuery,
	"review_requests": reviewRequestQuery,
	"assignees":       assigneeQuery,
	"checks":          checksQuery,
	"review_threads":  threadQuery,
	"activity":        activityQuery,
	"last_touched":    lastTouchedQuery,
}

func splitRepo(repo string) (string, string, error) {
	parts := strings.Split(repo, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", &domain.FetchError{Kind: domain.ErrorPermanent, Message: "repository must be owner/name"}
	}
	return parts[0], parts[1], nil
}

func fieldValue(name, viewer string, pr *struct {
	TimelineItems struct {
		UpdatedAt time.Time `json:"updatedAt"`
	} `json:"timelineItems"`
	Reviews struct {
		Nodes []struct {
			DatabaseID int64 `json:"databaseId"`
			Author     *struct {
				Login    string `json:"login"`
				Typename string `json:"__typename"`
			} `json:"author"`
			State       string     `json:"state"`
			SubmittedAt *time.Time `json:"submittedAt"`
		} `json:"nodes"`
		PageInfo struct {
			HasPreviousPage bool `json:"hasPreviousPage"`
		} `json:"pageInfo"`
	} `json:"reviews"`
	ReviewRequests struct {
		Nodes []struct {
			RequestedReviewer *struct {
				Typename     string `json:"__typename"`
				Login        string `json:"login"`
				Slug         string `json:"slug"`
				Organization *struct {
					Login string `json:"login"`
				} `json:"organization"`
			} `json:"requestedReviewer"`
		} `json:"nodes"`
		PageInfo struct {
			HasNextPage bool `json:"hasNextPage"`
		} `json:"pageInfo"`
	} `json:"reviewRequests"`
	Assignees struct {
		Nodes []struct {
			Login string `json:"login"`
		} `json:"nodes"`
		PageInfo struct {
			HasNextPage bool `json:"hasNextPage"`
		} `json:"pageInfo"`
	} `json:"assignees"`
	Commits struct {
		Nodes []struct {
			Commit struct {
				StatusCheckRollup *struct {
					Contexts struct {
						Nodes []struct {
							Typename   string `json:"__typename"`
							Status     string `json:"status"`
							Conclusion string `json:"conclusion"`
							State      string `json:"state"`
						} `json:"nodes"`
						PageInfo struct {
							HasNextPage bool `json:"hasNextPage"`
						} `json:"pageInfo"`
					} `json:"contexts"`
				} `json:"statusCheckRollup"`
			} `json:"commit"`
		} `json:"nodes"`
	} `json:"commits"`
	ReviewThreads struct {
		Nodes []struct {
			IsResolved bool `json:"isResolved"`
		} `json:"nodes"`
		PageInfo struct {
			HasNextPage bool `json:"hasNextPage"`
		} `json:"pageInfo"`
	} `json:"reviewThreads"`
	Comments struct {
		Nodes []struct {
			Body      string    `json:"body"`
			CreatedAt time.Time `json:"createdAt"`
			UpdatedAt time.Time `json:"updatedAt"`
			URL       string    `json:"url"`
			Author    *struct {
				Login string `json:"login"`
			} `json:"author"`
		} `json:"nodes"`
		PageInfo struct {
			HasPreviousPage bool `json:"hasPreviousPage"`
		} `json:"pageInfo"`
	} `json:"comments"`
}, now time.Time) (any, error) {
	switch name {
	case "reviews":
		set := domain.ReviewSet{Complete: !pr.Reviews.PageInfo.HasPreviousPage, ViewerLogin: viewer}
		for _, raw := range pr.Reviews.Nodes {
			if raw.SubmittedAt == nil || strings.EqualFold(raw.State, "PENDING") || raw.Author == nil {
				continue
			}
			set.Reviews = append(set.Reviews, domain.Review{Author: raw.Author.Login, IsBot: raw.Author.Typename == "Bot" || strings.HasSuffix(strings.ToLower(raw.Author.Login), "[bot]"), State: raw.State, At: *raw.SubmittedAt, ID: fmt.Sprint(raw.DatabaseID)})
		}
		return set, nil
	case "review_requests":
		set := domain.LoginSet{Complete: !pr.ReviewRequests.PageInfo.HasNextPage, ViewerLogin: viewer}
		for _, raw := range pr.ReviewRequests.Nodes {
			if raw.RequestedReviewer == nil {
				continue
			}
			switch raw.RequestedReviewer.Typename {
			case "User":
				set.Logins = append(set.Logins, raw.RequestedReviewer.Login)
			case "Team":
				// Team requests may include the current user, so a missing direct match is not conclusive.
				set.Complete = false
			}
		}
		return set, nil
	case "assignees":
		set := domain.LoginSet{Complete: !pr.Assignees.PageInfo.HasNextPage, ViewerLogin: viewer}
		for _, raw := range pr.Assignees.Nodes {
			set.Logins = append(set.Logins, raw.Login)
		}
		return set, nil
	case "checks":
		summary := domain.CheckSummary{Complete: true}
		if len(pr.Commits.Nodes) == 0 || pr.Commits.Nodes[len(pr.Commits.Nodes)-1].Commit.StatusCheckRollup == nil {
			return summary, nil
		}
		contexts := pr.Commits.Nodes[len(pr.Commits.Nodes)-1].Commit.StatusCheckRollup.Contexts
		summary.Complete = !contexts.PageInfo.HasNextPage
		for _, check := range contexts.Nodes {
			summary.Total++
			if check.Typename == "CheckRun" {
				switch strings.ToUpper(check.Status) {
				case "QUEUED", "IN_PROGRESS", "PENDING", "REQUESTED", "WAITING", "EXPECTED":
					summary.Pending++
				default:
					switch strings.ToUpper(check.Conclusion) {
					case "FAILURE", "TIMED_OUT", "ACTION_REQUIRED", "STARTUP_FAILURE":
						summary.Failed++
					}
				}
			} else if check.Typename == "StatusContext" {
				switch strings.ToUpper(check.State) {
				case "ERROR", "FAILURE":
					summary.Failed++
				case "PENDING", "EXPECTED":
					summary.Pending++
				}
			}
		}
		return summary, nil
	case "review_threads":
		summary := domain.ThreadSummary{Complete: !pr.ReviewThreads.PageInfo.HasNextPage}
		for _, thread := range pr.ReviewThreads.Nodes {
			if !thread.IsResolved {
				summary.Unresolved++
			}
		}
		return summary, nil
	case "activity":
		comments := make([]activityComment, 0, len(pr.Comments.Nodes))
		for _, comment := range pr.Comments.Nodes {
			author := ""
			if comment.Author != nil {
				author = comment.Author.Login
			}
			comments = append(comments, activityComment{Body: comment.Body, CreatedAt: comment.CreatedAt, URL: comment.URL, Author: author})
		}
		return classifyActivity(comments, !pr.Comments.PageInfo.HasPreviousPage, now), nil
	default:
		return nil, &domain.FetchError{Kind: domain.ErrorPermanent, Message: "unsupported GitHub field: " + name}
	}
}

type activityComment struct {
	Body      string
	CreatedAt time.Time
	URL       string
	Author    string
}

func classifyActivity(comments []activityComment, complete bool, now time.Time) domain.ActivityEvidence {
	out := domain.ActivityEvidence{State: domain.ActivityUnknown, Complete: complete, FetchedAt: now, FreshUntil: now.Add(fieldFreshFor)}
	type candidate struct {
		body, url, author string
		createdAt         time.Time
		runID             string
		index             int
	}
	var latest *candidate
	for i, comment := range comments {
		if _, ok := fullsendAgentLogins[comment.Author]; !ok {
			continue
		}
		match := fullsendStatusMarker.FindStringSubmatch(comment.Body)
		if len(match) != 2 {
			continue
		}
		item := candidate{body: comment.Body, url: comment.URL, author: comment.Author, createdAt: comment.CreatedAt, runID: match[1], index: i}
		if latest == nil || item.createdAt.After(latest.createdAt) || (item.createdAt.Equal(latest.createdAt) && item.index > latest.index) {
			latest = &item
		}
	}
	if latest == nil {
		if complete {
			out.State = domain.ActivityInactive
		}
		return out
	}
	out.URL, out.Author, out.RunID, out.StartedAt = latest.url, latest.author, latest.runID, latest.createdAt
	out.Complete = true // The page contains the newest comments; older markers cannot supersede this one.
	if latest.createdAt.IsZero() || latest.createdAt.After(now) {
		out.State = domain.ActivityUnknown
		out.Complete = false
	} else if strings.Contains(latest.body, fullsendTerminalMarker) {
		out.State = domain.ActivityInactive
	} else if !now.Before(latest.createdAt) && now.Sub(latest.createdAt) >= activityStaleAfter {
		out.State = domain.ActivityStale
	} else {
		out.State = domain.ActivityActive
	}
	return out
}
