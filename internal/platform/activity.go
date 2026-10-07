package platform

import (
	"fmt"
)

// Activity import error codes.
const (
	CodeInvalidActivityExport = "invalid-activity-export"
	CodeCursorMismatch        = "cursor-mismatch"
	CodeAccountNotLinked      = "platform-account-not-linked"
	CodeInvalidMapping        = "invalid-mapping"
)

// ActivityReader is implemented by adapters that declare ActivitySource. Its
// contract is "records since cursor" (ADR-0059): it returns the target's
// activity records after since (nil means from the beginning) and the cursor
// to pass next time. Transport is the adapter's choice; ALP never reads
// platform databases.
type ActivityReader interface {
	ReadActivity(export []byte, target string, since *string) (ActivityBatch, error)
}

// ActivityBatch is one target's activity for one platform user.
type ActivityBatch struct {
	Target         ExternalID
	Instance       string
	PlatformUserID string
	Cursor         CursorRange
	Records        []ActivityRecord
}

// CursorRange is the opaque, platform-owned cursor range a batch covers. Since
// is nil when the batch is a full snapshot.
type CursorRange struct {
	Since *string `json:"since"`
	Next  string  `json:"next"`
}

// CheckCursor verifies that a batch exported since exported can be imported by
// a caller whose last cursor is requested (nil: nothing imported yet). A full
// snapshot covers every cursor; an incremental batch must start exactly where
// the caller left off, or activity could be missed.
func CheckCursor(platformID, target string, requested, exported *string) error {
	if exported == nil {
		return nil
	}
	if requested != nil && *requested == *exported {
		return nil
	}
	have := "no cursor (nothing imported yet)"
	if requested != nil {
		have = fmt.Sprintf("cursor %q", *requested)
	}
	return &Error{
		Code:    CodeCursorMismatch,
		Adapter: platformID,
		Target:  target,
		Message: fmt.Sprintf("activity export starts at cursor %q, but the import was given %s; export again since the cursor returned by the last import, or export a full snapshot", *exported, have),
	}
}

// ActivityRecord is one learner activity record on a platform item.
type ActivityRecord struct {
	Item       string
	Kind       string
	Event      EventIdentity
	ObservedAt string
	Signal     Signal
}

// EventIdentity identifies a platform event and its revision. Platforms with
// real event IDs supply them; state-only platforms supply a Synthetic Event
// Identity (row key plus a content hash of the evidence-relevant fields).
// Timestamps are never part of an identity.
type EventIdentity struct {
	ID        string `json:"id"`
	Revision  string `json:"revision"`
	Synthetic bool   `json:"synthetic,omitempty"`
}

// Signal is the adapter's signal-kind policy applied to one record: what the
// activity shows before Mapping Roles are applied (ADR-0058).
type Signal struct {
	EvidenceType string
	Result       string
	Strength     string
	FailureClass string
	Observation  string
	// Assessable reports whether this kind of signal can be assessment-grade
	// at all (an answered question can; navigation progress cannot).
	Assessable bool
}
