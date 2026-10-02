package domain

import (
	"fmt"
	"time"
)

// Now returns the current UTC time truncated to whole seconds, the
// canonical timestamp for storage (formatted with FormatTime).
func Now() time.Time {
	return time.Now().UTC().Truncate(time.Second)
}

// FormatTime renders t for SQLite TEXT columns (UTC, RFC 3339).
func FormatTime(t time.Time) string {
	return t.UTC().Truncate(time.Second).Format(time.RFC3339)
}

// ParseTime parses a storage timestamp back to UTC.
func ParseTime(s string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid timestamp %q: %w", s, err)
	}
	return t.UTC(), nil
}

// PriorityRank orders priorities high → normal → low for sorting.
func PriorityRank(p Priority) int {
	switch p {
	case PriorityHigh:
		return 0
	case PriorityNormal:
		return 1
	default:
		return 2
	}
}

// ChildCounts tallies records attached to a deletable entity.
type ChildCounts struct {
	Topics int
	Tasks  int
	Files  int
}

// Empty reports whether nothing is attached.
func (c ChildCounts) Empty() bool {
	return c.Topics == 0 && c.Tasks == 0 && c.Files == 0
}

// ConfirmRequiredError signals a delete blocked for lack of explicit
// confirmation. Fronts translate it into a message plus a --yes hint.
type ConfirmRequiredError struct {
	Entity string // e.g. "unit unit-001"
	Counts ChildCounts
}

func (e *ConfirmRequiredError) Error() string {
	return fmt.Sprintf("%s still has %d topic(s), %d task(s) and %d file(s): re-run with explicit confirmation, or archive it instead",
		e.Entity, e.Counts.Topics, e.Counts.Tasks, e.Counts.Files)
}
