package organizer

import (
	"testing"
)

func TestPlanGenerationAndOverride(t *testing.T) {
	// Planner generates move/rename plan for Movie
	planner := NewPlanner(PlannerConfig{
		DefaultMovieDir: "/115/电影",
		DefaultTVDir:    "/115/电视剧",
	})

	item := PlanItem{
		ID:          "item-1",
		CurrentPath: "/115/待整理/your.name.2016.1080p.mkv",
		Title:       "你的名字",
		Year:        2016,
		MediaType:   "movie",
		Ext:         "mkv",
	}

	// 1. Default planned path
	plan := planner.BuildPlan(item)
	expectedDefault := "/115/电影/你的名字 (2016)/你的名字 (2016).mkv"
	if plan.TargetPath != expectedDefault {
		t.Fatalf("expected default target '%s', got '%s'", expectedDefault, plan.TargetPath)
	}

	// 2. Administrator manual override target directory
	planner.SetOverride(&plan, "/115/电影/动画")
	expectedOverride := "/115/电影/动画/你的名字 (2016)/你的名字 (2016).mkv"
	if plan.EffectivePath() != expectedOverride {
		t.Fatalf("expected override path '%s', got '%s'", expectedOverride, plan.EffectivePath())
	}
}
