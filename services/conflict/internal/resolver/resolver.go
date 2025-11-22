package resolver

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/yourorg/collab/services/conflict/internal/models"
)

// ConflictResolver provides conflict resolution strategies
type ConflictResolver struct{}

// NewConflictResolver creates a new conflict resolver
func NewConflictResolver() *ConflictResolver {
	return &ConflictResolver{}
}

// Resolve resolves a conflict using the specified strategy
func (r *ConflictResolver) Resolve(
	conflict *models.Conflict,
	versions []*models.ConflictVersion,
	strategy models.ResolutionStrategy,
) ([]byte, error) {
	if len(versions) < 2 {
		return nil, fmt.Errorf("need at least 2 versions to resolve conflict")
	}

	switch strategy {
	case models.ResolutionStrategyAcceptTheirs:
		return r.acceptTheirs(versions)
	case models.ResolutionStrategyAcceptMine:
		return r.acceptMine(versions)
	case models.ResolutionStrategyLastWriteWins:
		return r.lastWriteWins(versions)
	case models.ResolutionStrategyFirstWriteWins:
		return r.firstWriteWins(versions)
	case models.ResolutionStrategyAutoMerge:
		return r.autoMerge(versions)
	default:
		return nil, fmt.Errorf("unsupported resolution strategy: %s", strategy)
	}
}

// DetectConflict checks if there's a conflict between versions
func (r *ConflictResolver) DetectConflict(
	currentContent []byte,
	currentHash string,
	newContent []byte,
	baseVersion int64,
) (bool, models.ConflictType) {
	// Calculate hash of new content
	newHash := r.CalculateHash(newContent)

	// If hashes match, no conflict
	if currentHash == newHash {
		return false, models.ConflictTypeUnknown
	}

	// If hashes don't match, there's a conflict
	// Determine conflict type based on content analysis
	if len(currentContent) == 0 {
		return true, models.ConflictTypeDeleteEdit
	}

	if len(newContent) == 0 {
		return true, models.ConflictTypeDeleteEdit
	}

	// Default to edit-edit conflict
	return true, models.ConflictTypeEditEdit
}

// CalculateHash calculates SHA256 hash of content
func (r *ConflictResolver) CalculateHash(content []byte) string {
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])
}

// acceptTheirs returns the remote (theirs) version
func (r *ConflictResolver) acceptTheirs(versions []*models.ConflictVersion) ([]byte, error) {
	// Find the latest version (remote)
	latest := versions[0]
	for _, v := range versions {
		if v.Timestamp.After(latest.Timestamp) {
			latest = v
		}
	}
	return latest.Content, nil
}

// acceptMine returns the local (mine) version
func (r *ConflictResolver) acceptMine(versions []*models.ConflictVersion) ([]byte, error) {
	// Find the earliest version (local/mine)
	earliest := versions[0]
	for _, v := range versions {
		if v.Timestamp.Before(earliest.Timestamp) {
			earliest = v
		}
	}
	return earliest.Content, nil
}

// lastWriteWins returns the most recent version
func (r *ConflictResolver) lastWriteWins(versions []*models.ConflictVersion) ([]byte, error) {
	latest := versions[0]
	for _, v := range versions {
		if v.Timestamp.After(latest.Timestamp) {
			latest = v
		}
	}
	return latest.Content, nil
}

// firstWriteWins returns the earliest version
func (r *ConflictResolver) firstWriteWins(versions []*models.ConflictVersion) ([]byte, error) {
	earliest := versions[0]
	for _, v := range versions {
		if v.Timestamp.Before(earliest.Timestamp) {
			earliest = v
		}
	}
	return earliest.Content, nil
}

// autoMerge attempts to automatically merge conflicting versions
func (r *ConflictResolver) autoMerge(versions []*models.ConflictVersion) ([]byte, error) {
	if len(versions) != 2 {
		return nil, fmt.Errorf("auto-merge currently supports only 2 versions")
	}

	v1 := string(versions[0].Content)
	v2 := string(versions[1].Content)

	// Simple line-based merge
	lines1 := strings.Split(v1, "\n")
	lines2 := strings.Split(v2, "\n")

	// Perform a simple 3-way merge simulation
	// In production, use a proper diff3 algorithm
	merged := r.simpleLineMerge(lines1, lines2)

	return []byte(strings.Join(merged, "\n")), nil
}

// simpleLineMerge performs a basic line-based merge
func (r *ConflictResolver) simpleLineMerge(lines1, lines2 []string) []string {
	result := make([]string, 0)
	i, j := 0, 0

	for i < len(lines1) && j < len(lines2) {
		if lines1[i] == lines2[j] {
			result = append(result, lines1[i])
			i++
			j++
		} else {
			// Conflict - add both with markers
			result = append(result, "<<<<<<< MINE")
			result = append(result, lines1[i])
			result = append(result, "=======")
			result = append(result, lines2[j])
			result = append(result, ">>>>>>> THEIRS")
			i++
			j++
		}
	}

	// Add remaining lines from either version
	for i < len(lines1) {
		result = append(result, lines1[i])
		i++
	}
	for j < len(lines2) {
		result = append(result, lines2[j])
		j++
	}

	return result
}

// CanAutoResolve determines if a conflict can be automatically resolved
func (r *ConflictResolver) CanAutoResolve(conflictType models.ConflictType) bool {
	switch conflictType {
	case models.ConflictTypeEditEdit:
		// Can attempt auto-merge for edit-edit conflicts
		return true
	case models.ConflictTypeVersion:
		// Version conflicts can use last-write-wins
		return true
	default:
		// Delete-edit and move-edit require manual resolution
		return false
	}
}
