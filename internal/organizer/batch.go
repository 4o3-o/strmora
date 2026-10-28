package organizer

import (
	"path"
)

type Batch struct {
	SourceDir  string
	TargetDir  string
	IsOverride bool
	Items      []Plan
}

func BuildBatches(plans []Plan, limit int) []Batch {
	if limit <= 0 {
		limit = 50
	}

	// 1. Separate default plans from override plans
	var defaultPlans []Plan
	var overridePlans []Plan

	for _, p := range plans {
		if p.OverridePath != "" {
			overridePlans = append(overridePlans, p)
		} else {
			defaultPlans = append(defaultPlans, p)
		}
	}

	out := make([]Batch, 0)

	// Helper to group by (SourceDir, TargetDir) and chunk
	groupAndChunk := func(list []Plan, isOverride bool) {
		type key struct {
			src string
			dst string
		}
		groups := make(map[key][]Plan)
		order := make([]key, 0)

		for _, p := range list {
			sDir := path.Dir(p.Item.CurrentPath)
			tDir := path.Dir(p.EffectivePath())
			k := key{src: sDir, dst: tDir}
			if _, exists := groups[k]; !exists {
				order = append(order, k)
			}
			groups[k] = append(groups[k], p)
		}

		for _, k := range order {
			items := groups[k]
			for i := 0; i < len(items); i += limit {
				end := i + limit
				if end > len(items) {
					end = len(items)
				}
				out = append(out, Batch{
					SourceDir:  k.src,
					TargetDir:  k.dst,
					IsOverride: isOverride,
					Items:      items[i:end],
				})
			}
		}
	}

	// Execute default target plans first!
	groupAndChunk(defaultPlans, false)
	// Execute override target plans after default!
	groupAndChunk(overridePlans, true)

	return out
}
