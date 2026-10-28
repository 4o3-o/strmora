package inventory

type DiffResult struct {
	Added     []Entry `json:"added"`
	Removed   []Entry `json:"removed"`
	Unchanged []Entry `json:"unchanged"`
}

// ComputeDiff compares oldTree and newTree by normalized Entry Path.
func ComputeDiff(oldTree, newTree Tree) DiffResult {
	oldMap := make(map[string]Entry, len(oldTree.Entries))
	for _, e := range oldTree.Entries {
		oldMap[e.Path] = e
	}

	newMap := make(map[string]Entry, len(newTree.Entries))
	for _, e := range newTree.Entries {
		newMap[e.Path] = e
	}

	var res DiffResult

	// Check items in newTree
	for _, e := range newTree.Entries {
		if _, exists := oldMap[e.Path]; exists {
			res.Unchanged = append(res.Unchanged, e)
		} else {
			res.Added = append(res.Added, e)
		}
	}

	// Check items in oldTree that disappeared
	for _, e := range oldTree.Entries {
		if _, exists := newMap[e.Path]; !exists {
			res.Removed = append(res.Removed, e)
		}
	}

	return res
}
