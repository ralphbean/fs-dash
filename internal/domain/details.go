package domain

import "time"

// DetailKey identifies one independently fetched PR detail group at a head revision.
type DetailKey struct {
	Item     ItemID
	Group    string
	Revision string
	ThreadID string
}

func (k DetailKey) String() string {
	key := k.Item.String() + ":detail:" + k.Group + ":" + k.Revision
	if k.ThreadID != "" {
		key += ":" + k.ThreadID
	}
	return key
}

type PRDescription struct {
	NodeID         string
	Body           string
	HeadRefName    string
	BaseRefName    string
	State          string
	IsDraft        bool
	Mergeable      string
	ReviewDecision string
}

type PRCommit struct {
	OID        string
	Message    string
	Author     string
	AuthoredAt time.Time
	URL        string
}

type PRCommitList struct {
	Commits  []PRCommit
	Complete bool
}

type PRCheck struct {
	Name       string
	Status     string
	Conclusion string
	URL        string
}

type PRCheckList struct {
	Checks   []PRCheck
	Complete bool
}

type PRComment struct {
	ID        string
	Body      string
	Author    string
	CreatedAt time.Time
	URL       string
}

type PRConversation struct {
	Comments []PRComment
	Complete bool
}

type ReviewThreadComment struct {
	ID        string
	Body      string
	Author    string
	CreatedAt time.Time
	URL       string
}

type ReviewThread struct {
	ID               string
	Path             string
	Line             int
	OriginalLine     int
	IsResolved       bool
	IsOutdated       bool
	Comments         []ReviewThreadComment
	CommentsComplete bool
}

type ReviewThreadList struct {
	Threads  []ReviewThread
	Complete bool
}

type ReviewThreadContext struct {
	ThreadID     string
	Path         string
	Line         int
	OriginalLine int
	DiffHunk     string
	IsOutdated   bool
}

type PRActionState struct {
	NodeID             string
	HeadSHA            string
	State              string
	IsDraft            bool
	Mergeable          string
	ReviewDecision     string
	MergeQueueRequired bool
	MergeQueueKnown    bool
	MergeCommitAllowed bool
	SquashMergeAllowed bool
	RebaseMergeAllowed bool
	AutoMergeEnabled   bool
	QueueEntryPresent  bool
	QueueEntryState    string
}
