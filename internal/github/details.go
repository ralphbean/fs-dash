package github

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/fullsend-ai/fs-dash/internal/domain"
	"github.com/fullsend-ai/fs-dash/internal/fetch"
)

const detailDescriptionQuery = `query($owner: String!, $name: String!, $number: Int!) {
  repository(owner: $owner, name: $name) {
    pullRequest(number: $number) {
      id body headRefName baseRefName state isDraft mergeable reviewDecision
    }
  }
}`

const detailCommitsQuery = `query($owner: String!, $name: String!, $number: Int!) {
  repository(owner: $owner, name: $name) {
    pullRequest(number: $number) {
      commits(last: 100) {
        nodes { commit { oid url messageHeadline authoredDate author { name user { login } } } }
        pageInfo { hasPreviousPage }
      }
    }
  }
}`

const detailChecksQuery = `query($owner: String!, $name: String!, $number: Int!) {
  repository(owner: $owner, name: $name) {
    pullRequest(number: $number) {
      commits(last: 1) { nodes { commit { statusCheckRollup {
        contexts(first: 100) {
          nodes {
            __typename
            ... on CheckRun { name status conclusion detailsUrl }
            ... on StatusContext { context state targetUrl }
          }
          pageInfo { hasNextPage }
        }
      } } } }
    }
  }
}`

const detailReviewsQuery = `query($owner: String!, $name: String!, $number: Int!) {
  repository(owner: $owner, name: $name) {
    pullRequest(number: $number) {
      reviews(last: 100) {
        nodes { databaseId author { login __typename } state submittedAt body url }
        pageInfo { hasPreviousPage }
      }
    }
  }
}`

const detailConversationQuery = `query($owner: String!, $name: String!, $number: Int!) {
  repository(owner: $owner, name: $name) {
    pullRequest(number: $number) {
      comments(last: 100) {
        nodes { id body createdAt url author { login } }
        pageInfo { hasPreviousPage }
      }
    }
  }
}`

const detailThreadsQuery = `query($owner: String!, $name: String!, $number: Int!, $cursor: String) {
  repository(owner: $owner, name: $name) {
    pullRequest(number: $number) {
      reviewThreads(first: 100, after: $cursor) {
        nodes {
          id isResolved path line originalLine
          comments(last: 100) {
            nodes { id body createdAt url author { login } }
            pageInfo { hasPreviousPage }
          }
        }
        pageInfo { hasNextPage endCursor }
      }
    }
  }
}`

const detailThreadContextQuery = `query($id: ID!) {
  node(id: $id) {
    ... on PullRequestReviewThread {
      id path line originalLine isResolved
      comments(last: 1) { nodes { diffHunk path line originalLine } }
    }
  }
}`

const inspectPRQuery = `query($owner: String!, $name: String!, $number: Int!) {
  repository(owner: $owner, name: $name) {
    mergeCommitAllowed squashMergeAllowed rebaseMergeAllowed
    pullRequest(number: $number) {
      id headRefOid state isDraft mergeable reviewDecision
      autoMergeRequest { enabledAt }
      mergeQueueEntry { state }
      baseRefName
    }
  }
}`

const resolvePRIDQuery = `query($owner: String!, $name: String!, $number: Int!) {
  repository(owner: $owner, name: $name) { pullRequest(number: $number) { id } }
}`

const postPRCommentMutation = `mutation($subject: ID!, $body: String!) {
  addComment(input: {subjectId: $subject, body: $body}) {
    commentEdge { node { id body author { login } createdAt url } }
  }
}`

const replyToReviewThreadMutation = `mutation($thread: ID!, $body: String!) {
  addPullRequestReviewThreadReply(input: {pullRequestReviewThreadId: $thread, body: $body}) {
    comment { id body author { login } createdAt url }
  }
}`

const closePRMutation = `mutation($pullRequest: ID!) {
  closePullRequest(input: {pullRequestId: $pullRequest}) {
    pullRequest { id state closedAt headRefOid }
  }
}`

type DetailClient struct {
	CLI  string
	Now  func() time.Time
	Gate *fetch.RateLimitGate
}

func (c DetailClient) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

func (c DetailClient) cli() string {
	if c.CLI != "" {
		return c.CLI
	}
	return "gh"
}

func (c DetailClient) host(item domain.ItemID) string {
	if item.Host == "" {
		return "github.com"
	}
	return item.Host
}

func (c DetailClient) graphQL(ctx context.Context, item domain.ItemID, query string, variables map[string]string, nodeQuery, repositoryVariables bool) (json.RawMessage, domain.Provenance, error) {
	owner, name, err := splitRepo(item.Repo)
	if err != nil {
		return nil, domain.Provenance{}, err
	}
	if c.Gate != nil {
		if err := c.Gate.Wait(ctx); err != nil {
			return nil, domain.Provenance{}, fetch.Classify(err)
		}
	}
	args := []string{"api", "graphql", "--hostname", c.host(item), "--include", "-f", "query=" + query}
	if repositoryVariables {
		args = append(args, "-F", "owner="+owner, "-F", "name="+name, "-F", "number="+strconv.Itoa(item.Number))
	}
	for name, value := range variables {
		args = append(args, "-F", name+"="+value)
	}
	output, commandErr := exec.CommandContext(ctx, c.cli(), args...).CombinedOutput()
	observedAt := c.now()
	headers, body, responseErr := parseIncludedResponse(output)
	quota := parseQuotaObservation(headers, observedAt)
	provenance := domain.Provenance{Source: "github", Quota: quota}
	if commandErr != nil {
		issue := classifyCLIError(commandErr, string(output))
		applyRateLimitMetadata(c.Gate, issue, headers, quota, observedAt)
		return nil, provenance, issue
	}
	if responseErr != nil {
		applyRateLimitMetadata(c.Gate, nil, headers, quota, observedAt)
		return nil, provenance, &domain.FetchError{Kind: domain.ErrorTransient, Message: "GitHub returned unreadable response metadata"}
	}
	var root struct {
		Data   map[string]json.RawMessage `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&root); err != nil {
		applyRateLimitMetadata(c.Gate, nil, headers, quota, observedAt)
		return nil, provenance, &domain.FetchError{Kind: domain.ErrorTransient, Message: "GitHub returned unreadable PR details"}
	}
	if len(root.Errors) > 0 {
		issue := classifyCLIError(errors.New("GraphQL error"), root.Errors[0].Message)
		applyRateLimitMetadata(c.Gate, issue, headers, quota, observedAt)
		return nil, provenance, issue
	}
	applyRateLimitMetadata(c.Gate, nil, headers, quota, observedAt)
	var raw json.RawMessage
	if nodeQuery {
		raw = root.Data["node"]
	} else if repository := root.Data["repository"]; len(repository) > 0 {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(repository, &fields); err == nil {
			raw = fields["pullRequest"]
			if len(raw) == 0 {
				raw = repository
			}
		}
	} else if len(root.Data) == 1 {
		for _, result := range root.Data {
			raw = result
		}
	}
	if len(raw) == 0 || string(raw) == "null" {
		return nil, provenance, &domain.FetchError{Kind: domain.ErrorPermission, Message: "GitHub pull request unavailable or access denied"}
	}
	provenance.FetchedAt = observedAt
	provenance.Version = observedAt.Format(time.RFC3339Nano)
	provenance.FreshUntil = observedAt.Add(fieldFreshFor)
	return raw, provenance, nil
}

func decodeDetail[T any](raw json.RawMessage) (T, error) {
	var value T
	if err := json.Unmarshal(raw, &value); err != nil {
		return value, &domain.FetchError{Kind: domain.ErrorTransient, Message: "GitHub returned malformed PR detail data"}
	}
	return value, nil
}

func decodeDetailValue[T any](raw json.RawMessage, provenance domain.Provenance) (any, domain.Provenance, error) {
	value, err := decodeDetail[T](raw)
	return value, provenance, err
}

func (c DetailClient) FetchDetail(ctx context.Context, key domain.DetailKey) (any, domain.Provenance, error) {
	if key.Group == "threads" {
		return c.fetchThreadDetails(ctx, key)
	}
	query := ""
	switch key.Group {
	case "description":
		query = detailDescriptionQuery
	case "commits":
		query = detailCommitsQuery
	case "checks":
		query = detailChecksQuery
	case "reviews":
		query = detailReviewsQuery
	case "conversation":
		query = detailConversationQuery
	case "thread_context":
		query = detailThreadContextQuery
	default:
		return nil, domain.Provenance{}, &domain.FetchError{Kind: domain.ErrorPermanent, Message: "unsupported GitHub detail group: " + key.Group}
	}
	variables := map[string]string{}
	if key.Group == "thread_context" {
		variables["id"] = key.ThreadID
	}
	raw, provenance, err := c.graphQL(ctx, key.Item, query, variables, key.Group == "thread_context", key.Group != "thread_context")
	if err != nil {
		return nil, provenance, err
	}
	switch key.Group {
	case "description":
		return decodeDetailValue[domain.PRDescription](raw, provenance)
	case "commits":
		var result struct {
			Commits struct {
				Nodes []struct {
					Commit struct {
						OID        string    `json:"oid"`
						URL        string    `json:"url"`
						Message    string    `json:"messageHeadline"`
						AuthoredAt time.Time `json:"authoredDate"`
						Author     struct {
							Name string `json:"name"`
							User *struct {
								Login string `json:"login"`
							} `json:"user"`
						} `json:"author"`
					} `json:"commit"`
				} `json:"nodes"`
				PageInfo struct {
					HasPreviousPage bool `json:"hasPreviousPage"`
				} `json:"pageInfo"`
			} `json:"commits"`
		}
		if err := json.Unmarshal(raw, &result); err != nil {
			return nil, provenance, &domain.FetchError{Kind: domain.ErrorTransient, Message: "GitHub returned malformed commit data"}
		}
		value := domain.PRCommitList{Complete: !result.Commits.PageInfo.HasPreviousPage}
		for _, node := range result.Commits.Nodes {
			author := node.Commit.Author.Name
			if node.Commit.Author.User != nil && node.Commit.Author.User.Login != "" {
				author = node.Commit.Author.User.Login
			}
			value.Commits = append(value.Commits, domain.PRCommit{OID: node.Commit.OID, Message: node.Commit.Message, Author: author, AuthoredAt: node.Commit.AuthoredAt, URL: node.Commit.URL})
		}
		return value, provenance, nil
	case "checks":
		var result struct {
			Commits struct {
				Nodes []struct {
					Commit struct {
						StatusCheckRollup *struct {
							Contexts struct {
								Nodes []struct {
									Type       string `json:"__typename"`
									Name       string `json:"name"`
									Status     string `json:"status"`
									Conclusion string `json:"conclusion"`
									DetailsURL string `json:"detailsUrl"`
									Context    string `json:"context"`
									State      string `json:"state"`
									TargetURL  string `json:"targetUrl"`
								} `json:"nodes"`
								PageInfo struct {
									HasNextPage bool `json:"hasNextPage"`
								} `json:"pageInfo"`
							} `json:"contexts"`
						} `json:"statusCheckRollup"`
					} `json:"commit"`
				} `json:"nodes"`
			} `json:"commits"`
		}
		if err := json.Unmarshal(raw, &result); err != nil {
			return nil, provenance, &domain.FetchError{Kind: domain.ErrorTransient, Message: "GitHub returned malformed check data"}
		}
		value := domain.PRCheckList{Complete: true}
		if len(result.Commits.Nodes) > 0 && result.Commits.Nodes[0].Commit.StatusCheckRollup != nil {
			contexts := result.Commits.Nodes[0].Commit.StatusCheckRollup.Contexts
			value.Complete = !contexts.PageInfo.HasNextPage
			for _, check := range contexts.Nodes {
				name, status, conclusion, checkURL := check.Name, check.Status, check.Conclusion, check.DetailsURL
				if check.Type == "StatusContext" {
					name, status, checkURL = check.Context, check.State, check.TargetURL
				}
				value.Checks = append(value.Checks, domain.PRCheck{Name: name, Status: status, Conclusion: conclusion, URL: checkURL})
			}
		}
		return value, provenance, nil
	case "reviews":
		var result struct {
			Reviews struct {
				Nodes []struct {
					DatabaseID int64 `json:"databaseId"`
					Author     *struct {
						Login string `json:"login"`
						Type  string `json:"__typename"`
					} `json:"author"`
					State       string     `json:"state"`
					SubmittedAt *time.Time `json:"submittedAt"`
					Body        string     `json:"body"`
					URL         string     `json:"url"`
				} `json:"nodes"`
				PageInfo struct {
					HasPreviousPage bool `json:"hasPreviousPage"`
				} `json:"pageInfo"`
			} `json:"reviews"`
		}
		if err := json.Unmarshal(raw, &result); err != nil {
			return nil, provenance, &domain.FetchError{Kind: domain.ErrorTransient, Message: "GitHub returned malformed review data"}
		}
		value := domain.ReviewSet{Complete: !result.Reviews.PageInfo.HasPreviousPage}
		for _, review := range result.Reviews.Nodes {
			if review.Author == nil || review.SubmittedAt == nil || strings.EqualFold(review.State, "PENDING") {
				continue
			}
			login := review.Author.Login
			value.Reviews = append(value.Reviews, domain.Review{ID: strconv.FormatInt(review.DatabaseID, 10), Author: login, IsBot: review.Author.Type == "Bot" || strings.HasSuffix(strings.ToLower(login), "[bot]"), State: review.State, At: *review.SubmittedAt, Body: review.Body, URL: review.URL})
		}
		return value, provenance, nil
	case "conversation":
		var result struct {
			Comments struct {
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
		}
		if err := json.Unmarshal(raw, &result); err != nil {
			return nil, provenance, &domain.FetchError{Kind: domain.ErrorTransient, Message: "GitHub returned malformed conversation data"}
		}
		value := domain.PRConversation{Complete: !result.Comments.PageInfo.HasPreviousPage}
		for _, comment := range result.Comments.Nodes {
			author := "[deleted]"
			if comment.Author != nil {
				author = comment.Author.Login
			}
			value.Comments = append(value.Comments, domain.PRComment{ID: comment.ID, Body: comment.Body, Author: author, CreatedAt: comment.CreatedAt, URL: comment.URL})
		}
		return value, provenance, nil
	case "thread_context":
		var result *struct {
			ID           string `json:"id"`
			Path         string `json:"path"`
			Line         *int   `json:"line"`
			OriginalLine *int   `json:"originalLine"`
			IsResolved   bool   `json:"isResolved"`
			Comments     struct {
				Nodes []struct {
					DiffHunk     string `json:"diffHunk"`
					Path         string `json:"path"`
					Line         *int   `json:"line"`
					OriginalLine *int   `json:"originalLine"`
				} `json:"nodes"`
			} `json:"comments"`
		}
		if err := json.Unmarshal(raw, &result); err != nil {
			return nil, provenance, &domain.FetchError{Kind: domain.ErrorTransient, Message: "GitHub returned malformed thread context"}
		}
		if result == nil || result.ID != key.ThreadID {
			return nil, provenance, &domain.FetchError{Kind: domain.ErrorPermanent, Message: "review thread is no longer available"}
		}
		context := domain.ReviewThreadContext{ThreadID: result.ID, Path: result.Path}
		if result.Line != nil {
			context.Line = *result.Line
		}
		if result.OriginalLine != nil {
			context.OriginalLine = *result.OriginalLine
		}
		context.IsOutdated = result.Line == nil || *result.Line == 0
		if len(result.Comments.Nodes) > 0 {
			node := result.Comments.Nodes[0]
			context.DiffHunk = node.DiffHunk
			if context.Path == "" {
				context.Path = node.Path
			}
			if context.OriginalLine == 0 && node.OriginalLine != nil {
				context.OriginalLine = *node.OriginalLine
			}
		}
		return context, provenance, nil
	default:
		return nil, provenance, nil
	}
}

func (c DetailClient) InspectPR(ctx context.Context, item domain.ItemID) (domain.PRActionState, error) {
	owner, name, err := splitRepo(item.Repo)
	if err != nil {
		return domain.PRActionState{}, err
	}
	raw, _, err := c.graphQL(ctx, item, inspectPRQuery, nil, false, true)
	if err != nil {
		return domain.PRActionState{}, err
	}
	var result struct {
		ID               string `json:"id"`
		HeadSHA          string `json:"headRefOid"`
		State            string `json:"state"`
		IsDraft          bool   `json:"isDraft"`
		Mergeable        string `json:"mergeable"`
		ReviewDecision   string `json:"reviewDecision"`
		BaseRefName      string `json:"baseRefName"`
		AutoMergeRequest *struct {
			EnabledAt time.Time `json:"enabledAt"`
		} `json:"autoMergeRequest"`
		MergeQueueEntry *struct {
			State string `json:"state"`
		} `json:"mergeQueueEntry"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return domain.PRActionState{}, &domain.FetchError{Kind: domain.ErrorTransient, Message: "GitHub returned malformed merge state"}
	}
	state := domain.PRActionState{NodeID: result.ID, HeadSHA: result.HeadSHA, State: result.State, IsDraft: result.IsDraft, Mergeable: result.Mergeable, ReviewDecision: result.ReviewDecision, MergeQueueKnown: true}
	if result.AutoMergeRequest != nil {
		state.AutoMergeEnabled = true
	}
	if result.MergeQueueEntry != nil {
		state.QueueEntryPresent, state.QueueEntryState = true, result.MergeQueueEntry.State
	}
	// Repository merge-method settings are queried through REST for GHES compatibility.
	repoRaw, err := c.runREST(ctx, item, "repos/"+url.PathEscape(owner)+"/"+url.PathEscape(name))
	if err != nil {
		state.MergeQueueKnown = false
		return state, nil
	}
	var repository struct {
		MergeCommitAllowed bool `json:"allow_merge_commit"`
		SquashMergeAllowed bool `json:"allow_squash_merge"`
		RebaseMergeAllowed bool `json:"allow_rebase_merge"`
	}
	if err := json.Unmarshal(repoRaw, &repository); err != nil {
		state.MergeQueueKnown = false
		return state, nil
	}
	state.MergeCommitAllowed, state.SquashMergeAllowed, state.RebaseMergeAllowed = repository.MergeCommitAllowed, repository.SquashMergeAllowed, repository.RebaseMergeAllowed
	branch := url.PathEscape(result.BaseRefName)
	rulesRaw, err := c.runREST(ctx, item, "repos/"+url.PathEscape(owner)+"/"+url.PathEscape(name)+"/rules/branches/"+branch)
	if err != nil {
		state.MergeQueueKnown = false
		return state, nil
	}
	var rules []struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(rulesRaw, &rules); err != nil {
		state.MergeQueueKnown = false
		return state, nil
	}
	for _, rule := range rules {
		if rule.Type == "merge_queue" {
			state.MergeQueueRequired = true
			break
		}
	}
	return state, nil
}

func (c DetailClient) runREST(ctx context.Context, item domain.ItemID, endpoint string) ([]byte, error) {
	if c.Gate != nil {
		if err := c.Gate.Wait(ctx); err != nil {
			return nil, fetch.Classify(err)
		}
	}
	args := []string{"api", "--hostname", c.host(item), "--include", endpoint}
	output, err := exec.CommandContext(ctx, c.cli(), args...).CombinedOutput()
	observedAt := c.now()
	headers, body, responseErr := parseIncludedResponse(output)
	quota := parseQuotaObservation(headers, observedAt)
	if err != nil {
		issue := classifyCLIError(err, string(output))
		applyRateLimitMetadata(c.Gate, issue, headers, quota, observedAt)
		return nil, issue
	}
	if responseErr != nil {
		applyRateLimitMetadata(c.Gate, nil, headers, quota, observedAt)
		return nil, &domain.FetchError{Kind: domain.ErrorTransient, Message: "GitHub returned unreadable response metadata"}
	}
	applyRateLimitMetadata(c.Gate, nil, headers, quota, observedAt)
	return body, nil
}

func (c DetailClient) mutation(ctx context.Context, item domain.ItemID, query string, variables map[string]string) (json.RawMessage, error) {
	raw, _, err := c.graphQL(ctx, item, query, variables, false, false)
	if err != nil {
		issue := fetch.Classify(err)
		return nil, &domain.FetchError{Kind: issue.Kind, Message: "GitHub write request failed; verify remote state before retrying", RetryAt: issue.RetryAt, RetryAfter: issue.RetryAfter}
	}
	return raw, err
}

func (c DetailClient) resolvePRID(ctx context.Context, item domain.ItemID) (string, error) {
	raw, _, err := c.graphQL(ctx, item, resolvePRIDQuery, nil, false, true)
	if err != nil {
		return "", err
	}
	var result struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(raw, &result); err != nil || result.ID == "" {
		return "", &domain.FetchError{Kind: domain.ErrorTransient, Message: "GitHub returned malformed pull request identity"}
	}
	return result.ID, nil
}

func (c DetailClient) PostPRComment(ctx context.Context, item domain.ItemID, body string) (string, error) {
	id, err := c.resolvePRID(ctx, item)
	if err != nil {
		return "", err
	}
	raw, err := c.mutation(ctx, item, postPRCommentMutation, map[string]string{"subject": id, "body": body})
	if err != nil {
		return "", err
	}
	var result struct {
		CommentEdge struct {
			Node struct {
				ID string `json:"id"`
			} `json:"node"`
		} `json:"commentEdge"`
	}
	if err := json.Unmarshal(raw, &result); err != nil || result.CommentEdge.Node.ID == "" {
		return "", &domain.FetchError{Kind: domain.ErrorTransient, Message: "GitHub accepted the comment request without returning a comment ID"}
	}
	return result.CommentEdge.Node.ID, nil
}

func (c DetailClient) ReplyToReviewThread(ctx context.Context, item domain.ItemID, threadID, body string) (string, error) {
	raw, err := c.mutation(ctx, item, replyToReviewThreadMutation, map[string]string{"thread": threadID, "body": body})
	if err != nil {
		return "", err
	}
	var result struct {
		Reply struct {
			ID string `json:"id"`
		} `json:"comment"`
	}
	if err := json.Unmarshal(raw, &result); err != nil || result.Reply.ID == "" {
		return "", &domain.FetchError{Kind: domain.ErrorTransient, Message: "GitHub accepted the reply request without returning a comment ID"}
	}
	return result.Reply.ID, nil
}

func (c DetailClient) MergePR(ctx context.Context, item domain.ItemID, expectedHead string, previous domain.PRActionState) error {
	current, err := c.InspectPR(ctx, item)
	if err != nil {
		return err
	}
	if current.HeadSHA != expectedHead || current.HeadSHA != previous.HeadSHA {
		return &domain.FetchError{Kind: domain.ErrorPermanent, Message: "PR head changed; reload details before merging"}
	}
	if current.MergeQueueRequired != previous.MergeQueueRequired || current.SquashMergeAllowed != previous.SquashMergeAllowed || current.MergeCommitAllowed != previous.MergeCommitAllowed || current.RebaseMergeAllowed != previous.RebaseMergeAllowed {
		return &domain.FetchError{Kind: domain.ErrorPermanent, Message: "merge policy changed; inspect and confirm the action again"}
	}
	if !strings.EqualFold(current.State, "OPEN") || current.IsDraft {
		return &domain.FetchError{Kind: domain.ErrorPermanent, Message: "PR is no longer open and ready for a merge action"}
	}
	if !current.MergeQueueKnown {
		return &domain.FetchError{Kind: domain.ErrorPermission, Message: "could not determine current merge-queue policy"}
	}
	args := []string{"pr", "merge", strconv.Itoa(item.Number), "--repo", c.host(item) + "/" + item.Repo, "--match-head-commit", expectedHead, "--auto"}
	if !current.MergeQueueRequired {
		switch {
		case current.SquashMergeAllowed:
			args = append(args, "--squash")
		case current.MergeCommitAllowed:
			args = append(args, "--merge")
		case current.RebaseMergeAllowed:
			args = append(args, "--rebase")
		default:
			return &domain.FetchError{Kind: domain.ErrorPermanent, Message: "repository has no enabled merge method"}
		}
	}
	output, err := exec.CommandContext(ctx, c.cli(), args...).CombinedOutput()
	if err != nil {
		return classifyCLIError(err, string(output))
	}
	return nil
}

func (c DetailClient) ClosePR(ctx context.Context, item domain.ItemID, expectedHead string) error {
	current, err := c.InspectPR(ctx, item)
	if err != nil {
		return err
	}
	if current.HeadSHA != expectedHead {
		return &domain.FetchError{Kind: domain.ErrorPermanent, Message: "PR head changed; reload details before closing"}
	}
	if !strings.EqualFold(current.State, "OPEN") {
		return &domain.FetchError{Kind: domain.ErrorPermanent, Message: "PR is no longer open"}
	}
	raw, err := c.mutation(ctx, item, closePRMutation, map[string]string{"pullRequest": current.NodeID})
	if err != nil {
		return err
	}
	var result struct {
		PullRequest struct {
			State   string `json:"state"`
			HeadSHA string `json:"headRefOid"`
		} `json:"pullRequest"`
	}
	if err := json.Unmarshal(raw, &result); err != nil || !strings.EqualFold(result.PullRequest.State, "CLOSED") || result.PullRequest.HeadSHA != expectedHead {
		return &domain.FetchError{Kind: domain.ErrorTransient, Message: "GitHub accepted the close request but returned an unverified PR state"}
	}
	return nil
}
