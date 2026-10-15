package review

import (
	"sort"

	"github.com/lamngockhuong/muninn/internal/content"
)

// Displayed states. Modified and outdated are derived from hashes; they are
// never written to files.
const (
	StateDraft            = content.ReviewDraft
	StateApproved         = content.ReviewApproved
	StateChangesRequested = content.ReviewChangesRequested
	StateModified         = "modified"
	StateOutdated         = "outdated"
)

// States in display order.
var States = []string{StateDraft, StateChangesRequested, StateModified, StateOutdated, StateApproved}

// Reason explains why an approved entity is no longer approved.
type Reason struct {
	Kind string `json:"kind"`           // content | source-changed | source-missing | source-added
	Path string `json:"path,omitempty"` // source path for source reasons
}

// Status is the computed review state of one entity.
type Status struct {
	State   string   `json:"state"`
	Reasons []Reason `json:"reasons,omitempty"`
}

// Compute derives the displayed state:
//   - approved with a content hash mismatch is modified (edited after review);
//   - approved with a changed, missing or newly listed source is outdated.
//
// Modified wins because the reviewed text itself is gone; source reasons are
// still listed so the reviewer sees everything that moved.
func Compute(ix *content.Index, h *Hasher, e *content.Entity) Status {
	switch e.Review.Status {
	case content.ReviewApproved:
	case content.ReviewChangesRequested:
		return Status{State: StateChangesRequested}
	default:
		return Status{State: StateDraft}
	}
	var reasons []Reason
	modified := false
	if hash, err := ContentHash(ix, e); err != nil || hash != e.Review.ContentHash {
		modified = true
		reasons = append(reasons, Reason{Kind: "content"})
	}
	for _, src := range e.Sources {
		want, listed := e.Review.SourceHashes[src]
		got, exists, err := h.Source(src)
		switch {
		case err != nil || !exists:
			reasons = append(reasons, Reason{Kind: "source-missing", Path: src})
		case !listed:
			reasons = append(reasons, Reason{Kind: "source-added", Path: src})
		case got != want:
			reasons = append(reasons, Reason{Kind: "source-changed", Path: src})
		}
	}
	sort.SliceStable(reasons, func(i, j int) bool { return reasons[i].Path < reasons[j].Path })
	switch {
	case modified:
		return Status{State: StateModified, Reasons: reasons}
	case len(reasons) > 0:
		return Status{State: StateOutdated, Reasons: reasons}
	}
	return Status{State: StateApproved}
}

// ComputeAll returns the state of every entity keyed by "kind/id".
func ComputeAll(ix *content.Index, h *Hasher) map[string]Status {
	out := make(map[string]Status)
	for _, e := range ix.All() {
		out[e.Key()] = Compute(ix, h, e)
	}
	return out
}
