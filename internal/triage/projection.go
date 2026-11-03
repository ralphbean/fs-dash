package triage

import (
	"sort"
	"strings"
	"time"

	"github.com/fullsend-ai/fs-dash/internal/domain"
)

type Filter string

const (
	FilterAll             Filter = "all"
	FilterAssigned        Filter = "assigned-to-me"
	FilterReviewRequested Filter = "review-requested-from-me"
	FilterAuthored        Filter = "authored-by-me"
)

func (f Filter) Next() Filter {
	switch f {
	case FilterAll:
		return FilterAssigned
	case FilterAssigned:
		return FilterReviewRequested
	case FilterReviewRequested:
		return FilterAuthored
	default:
		return FilterAll
	}
}

func (f Filter) Label() string {
	switch f {
	case FilterAssigned:
		return "assigned to me"
	case FilterReviewRequested:
		return "review requested from me"
	case FilterAuthored:
		return "authored by me"
	default:
		return "all open PRs"
	}
}

type Input struct {
	PR            domain.PR
	Assigned      domain.DerivedBool
	ReviewRequest domain.DerivedBool
	AuthoredByMe  domain.DerivedBool
	Checks        domain.CheckSummary
	Threads       domain.ThreadSummary
	BotApproval   domain.DerivedBool
	Activity      domain.ActivityEvidence
	Starred       bool
	SnoozedUntil  time.Time
}

type Reason struct {
	Code   string
	Points int
}

type Row struct {
	PR            domain.PR
	Score         int
	Reasons       []Reason
	FilterPending bool
	Starred       bool
	SnoozedUntil  time.Time
	Activity      domain.ActivityEvidence
}

type Sections struct {
	Review  []Row
	Active  []Row
	Snoozed []Row
}

const staleAfter = 7 * 24 * time.Hour

func Project(inputs []Input, filter Filter, showSnoozed bool, now time.Time) Sections {
	var out Sections
	for _, in := range inputs {
		snoozed := in.SnoozedUntil.After(now)
		if showSnoozed {
			if snoozed {
				out.Snoozed = append(out.Snoozed, makeRow(in, now, false))
			}
			continue
		}
		if snoozed {
			continue
		}
		matches, pending := matchesFilter(in, filter)
		if !matches {
			continue
		}
		row := makeRow(in, now, pending)
		if in.Activity.State == domain.ActivityActive {
			out.Active = append(out.Active, row)
		} else {
			out.Review = append(out.Review, row)
		}
	}
	sortRows(out.Review)
	sortRows(out.Active)
	sort.Slice(out.Snoozed, func(i, j int) bool {
		if !out.Snoozed[i].SnoozedUntil.Equal(out.Snoozed[j].SnoozedUntil) {
			return out.Snoozed[i].SnoozedUntil.Before(out.Snoozed[j].SnoozedUntil)
		}
		return lessID(out.Snoozed[i].PR.ID, out.Snoozed[j].PR.ID)
	})
	return out
}

func matchesFilter(in Input, filter Filter) (bool, bool) {
	switch filter {
	case FilterAssigned:
		if !in.Assigned.Known {
			return true, true
		}
		return in.Assigned.Value, false
	case FilterReviewRequested:
		if !in.ReviewRequest.Known {
			return true, true
		}
		return in.ReviewRequest.Value, false
	case FilterAuthored:
		if !in.AuthoredByMe.Known {
			return true, true
		}
		return in.AuthoredByMe.Value, false
	default:
		return true, false
	}
}

func makeRow(in Input, now time.Time, pending bool) Row {
	row := Row{PR: in.PR, FilterPending: pending, Starred: in.Starred, SnoozedUntil: in.SnoozedUntil, Activity: in.Activity}
	if in.ReviewRequest.Known && in.ReviewRequest.Value {
		row.add("review_requested", 100)
	}
	if in.Checks.Failed > 0 {
		row.add("checks_failed", 80)
	}
	if in.Threads.Unresolved > 0 {
		row.add("unresolved_threads", 60)
	}
	if in.PR.UpdatedAt.IsZero() || now.Sub(in.PR.UpdatedAt) >= staleAfter {
		row.add("updated_over_7_days", 20)
	}
	if in.BotApproval.Known && in.BotApproval.Value {
		row.add("bot_approved", -30)
	}
	return row
}

func (r *Row) add(code string, points int) {
	r.Score += points
	r.Reasons = append(r.Reasons, Reason{Code: code, Points: points})
}

func sortRows(rows []Row) {
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Score != rows[j].Score {
			return rows[i].Score > rows[j].Score
		}
		return lessID(rows[i].PR.ID, rows[j].PR.ID)
	})
}

func lessID(a, b domain.ItemID) bool {
	for _, pair := range [][2]string{{a.Source, b.Source}, {a.Host, b.Host}, {a.Repo, b.Repo}} {
		left, right := strings.ToLower(pair[0]), strings.ToLower(pair[1])
		if left != right {
			return left < right
		}
	}
	return a.Number < b.Number
}
