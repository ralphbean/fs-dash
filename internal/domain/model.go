package domain

import (
	"fmt"
	"strings"
	"time"
)

type ItemID struct {
	Source string
	Host   string
	Repo   string
	Number int
}

func (id ItemID) String() string {
	return fmt.Sprintf("%s:%s/%s#%d", id.Source, id.Host, id.Repo, id.Number)
}

type PR struct {
	ID        ItemID
	Title     string
	Author    string
	UpdatedAt time.Time
	URL       string
	HeadSHA   string
}

type FieldKey struct {
	Item     ItemID
	Name     string
	Revision string
}

func (k FieldKey) String() string { return k.Item.String() + ":" + k.Name + ":" + k.Revision }

type Phase string

const (
	Unrequested Phase = "unrequested"
	Queued      Phase = "queued"
	Fetching    Phase = "fetching"
	Ready       Phase = "ready"
	Cooldown    Phase = "cooldown"
	Failed      Phase = "failed"
	Canceled    Phase = "canceled"
)

func (p Phase) Active() bool { return p == Queued || p == Fetching }

type Provenance struct {
	Source     string
	FetchedAt  time.Time
	Version    string
	FreshUntil time.Time
}

type ErrorKind string

const (
	ErrorTransient  ErrorKind = "transient"
	ErrorRateLimit  ErrorKind = "rate_limit"
	ErrorAuth       ErrorKind = "auth"
	ErrorPermission ErrorKind = "permission"
	ErrorPermanent  ErrorKind = "permanent"
	ErrorCanceled   ErrorKind = "canceled"
)

type FetchError struct {
	Kind    ErrorKind
	Message string
	RetryAt time.Time
}

func (e *FetchError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func (e *FetchError) Retryable() bool {
	return e != nil && (e.Kind == ErrorTransient || e.Kind == ErrorRateLimit)
}

type Field[T any] struct {
	Value      T
	HasValue   bool
	Provenance Provenance
	Phase      Phase
	Error      *FetchError
	RetryAt    time.Time
	Generation uint64
}

func NewField[T any]() Field[T] { return Field[T]{Phase: Unrequested} }

func (f *Field[T]) Queue() uint64 {
	f.Generation++
	f.Phase = Queued
	f.Error = nil
	f.RetryAt = time.Time{}
	return f.Generation
}

func (f *Field[T]) Begin(generation uint64) bool {
	if f.Generation != generation || f.Phase != Queued {
		return false
	}
	f.Phase = Fetching
	return true
}

func (f *Field[T]) Succeed(generation uint64, value T, provenance Provenance) bool {
	if f.Generation != generation || !f.Phase.Active() {
		return false
	}
	f.Value, f.HasValue, f.Provenance = value, true, provenance
	f.Phase, f.Error, f.RetryAt = Ready, nil, time.Time{}
	return true
}

func (f *Field[T]) Fail(generation uint64, issue *FetchError, retryAt time.Time) bool {
	if f.Generation != generation || !f.Phase.Active() {
		return false
	}
	f.Error, f.RetryAt = issue, retryAt
	if !retryAt.IsZero() && issue.Retryable() {
		f.Phase = Cooldown
	} else {
		f.Phase = Failed
	}
	return true
}

func (f *Field[T]) Cancel(generation uint64) bool {
	if f.Generation != generation || !f.Phase.Active() {
		return false
	}
	f.Phase = Canceled
	return true
}

func (f Field[T]) Stale(now time.Time) bool {
	return f.HasValue && !f.Provenance.FreshUntil.IsZero() && !now.Before(f.Provenance.FreshUntil)
}

func (f Field[T]) DisplayState(now time.Time) string {
	if f.Phase.Active() {
		return string(f.Phase)
	}
	if f.Phase == Cooldown {
		return string(Cooldown)
	}
	if f.Phase == Failed {
		return string(Failed)
	}
	if f.Stale(now) {
		return "stale"
	}
	return string(f.Phase)
}

type Review struct {
	Author string
	IsBot  bool
	State  string
	At     time.Time
	ID     string
}

type Dependency struct {
	Name    string
	Version string
}

type DerivedBool struct {
	Known        bool
	Value        bool
	Dependencies []Dependency
}

type BoolDerivation struct {
	result DerivedBool
	valid  bool
}

func (d *BoolDerivation) Resolve(dependencies []Dependency, compute func() DerivedBool) DerivedBool {
	if !d.valid || !sameDependencies(d.result.Dependencies, dependencies) {
		d.result = compute()
		d.result.Dependencies = append([]Dependency(nil), dependencies...)
		d.valid = true
	}
	return d.result
}

func sameDependencies(a, b []Dependency) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// BotApproval is definitive only when the complete current review set is known.
func BotApproval(reviews []Review, complete bool, version string) DerivedBool {
	out := DerivedBool{Dependencies: []Dependency{{Name: "reviews", Version: version}}}
	if !complete || version == "" {
		return out
	}
	latest := map[string]Review{}
	for _, review := range reviews {
		key := strings.ToLower(review.Author)
		if prev, ok := latest[key]; !ok || review.At.After(prev.At) || (review.At.Equal(prev.At) && review.ID > prev.ID) {
			latest[key] = review
		}
	}
	out.Known = true
	for _, review := range latest {
		if review.IsBot && strings.EqualFold(review.State, "APPROVED") {
			out.Value = true
			break
		}
	}
	return out
}
