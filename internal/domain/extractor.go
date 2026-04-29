package domain

import "time"

// SessionState represents the state extracted from a project for saving.
type SessionState struct {
	RecentTurns []Turn
	GitVision   []FileMetadata
}

// Turn represents a single interaction in the history.
type Turn struct {
	Role    string
	Content string
}

// FileMetadata represents a file that has been changed or mentioned.
type FileMetadata struct {
	Path     string
	Status   string    // e.g., "M", "??", "A"
	Priority int       // Final ranking score
	ModTime  time.Time `yaml:"-"` // Used for ranking only, not persisted
}

// HistoryExtractor is the interface that all AI CLI adapters must implement.
type HistoryExtractor interface {
	// Detect returns true if the current project is using this AI CLI.
	Detect(workingDir string) bool

	// Extract reads the history and returns the parsed SessionState.
	// It should limit the extraction to the most recent relevant turns.
	Extract(workingDir string) (SessionState, error)
	
	// Name returns the name of the AI CLI adapter.
	Name() string
}
