package ai

// Decision captures an architectural or product choice distilled from PR debates.
type Decision struct {
	ID        string   // e.g. "DEC-104"
	Title     string   // Brief summary of what was decided
	Rationale string   // Why this choice was made (the "because...")
	ApprovedBy string  // Lead or reviewer who signed off
	PRNumber  int      // Originating PR
	AppliesTo []string // Affected files or packages
}
