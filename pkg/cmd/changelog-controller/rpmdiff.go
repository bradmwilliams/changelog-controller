package changelog_controller

// RpmDiff represents the difference in RPM packages between two releases
type RpmDiff struct {
	Changed map[string]RpmChangedDiff `json:"changed,omitempty"`
	Added   map[string]string         `json:"added,omitempty"`
	Removed map[string]string         `json:"removed,omitempty"`
}

// RpmChangedDiff represents a package whose version changed between releases
type RpmChangedDiff struct {
	Old string `json:"old,omitempty"`
	New string `json:"new,omitempty"`
}

// ComputeRpmDiff compares two package maps (name→version) and returns the
// added, removed, and changed packages. Returns nil when both maps are
// identical or both are empty
func ComputeRpmDiff(from, to map[string]string) *RpmDiff {
	if len(from) == 0 && len(to) == 0 {
		return nil
	}

	diff := &RpmDiff{}

	for pkg, fromVer := range from {
		toVer, exists := to[pkg]
		if !exists {
			if diff.Removed == nil {
				diff.Removed = make(map[string]string)
			}
			diff.Removed[pkg] = fromVer
		} else if fromVer != toVer {
			if diff.Changed == nil {
				diff.Changed = make(map[string]RpmChangedDiff)
			}
			diff.Changed[pkg] = RpmChangedDiff{Old: fromVer, New: toVer}
		}
	}

	for pkg, toVer := range to {
		if _, exists := from[pkg]; !exists {
			if diff.Added == nil {
				diff.Added = make(map[string]string)
			}
			diff.Added[pkg] = toVer
		}
	}

	if len(diff.Changed) == 0 && len(diff.Added) == 0 && len(diff.Removed) == 0 {
		return nil
	}

	return diff
}
