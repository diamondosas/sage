package ai

import "time"

// Decision captures an architectural or product choice distilled from PR debates.
type Decision struct {
	ID         string   // e.g. "DEC-104"
	Title      string   // Brief summary of what was decided
	Rationale  string   // Why this choice was made (the "because...")
	ApprovedBy string   // Lead or reviewer who signed off
	PRNumber   int      // Originating PR
	AppliesTo  []string // Affected files or packages
}

// FileAnalysis captures AI-synthesized architectural knowledge of a codebase file.
type FileAnalysis struct {
	Path         string    `json:"path"`
	Summary      string    `json:"summary"`      // Concise 1-2 sentence description of file purpose
	Role         string    `json:"role"`         // Architectural role (e.g. "Storage Engine", "CLI Handler", "Model")
	Exports      []string  `json:"exports"`      // Key exported types, functions, or interfaces
	Dependencies []string  `json:"dependencies"` // Key imports or referenced components
	Hash         string    `json:"hash"`         // SHA-256 hash of file content for change tracking
	UpdatedAt    time.Time `json:"updated_at"`
}

