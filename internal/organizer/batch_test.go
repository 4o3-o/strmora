package organizer

import (
	"fmt"
	"testing"
)

func TestBatchGroupingAndChunking(t *testing.T) {
	// Create 126 movie plans:
	// 100 with default target (/115/电影) from same source (/115/待整理)
	// 26 with manual override target (/115/电影/动画) from same source (/115/待整理)
	plans := make([]Plan, 0, 126)
	for i := 1; i <= 100; i++ {
		plans = append(plans, Plan{
			Item: PlanItem{
				ID:          fmt.Sprintf("movie-%d", i),
				CurrentPath: fmt.Sprintf("/115/待整理/Movie%d", i),
			},
			TargetPath: fmt.Sprintf("/115/电影/Movie%d", i),
			FolderRel:  fmt.Sprintf("Movie%d", i),
		})
	}
	for i := 101; i <= 126; i++ {
		p := Plan{
			Item: PlanItem{
				ID:          fmt.Sprintf("movie-%d", i),
				CurrentPath: fmt.Sprintf("/115/待整理/Movie%d", i),
			},
			TargetPath: fmt.Sprintf("/115/电影/Movie%d", i),
			FolderRel:  fmt.Sprintf("Movie%d", i),
		}
		p.OverridePath = fmt.Sprintf("/115/电影/动画/Movie%d", i)
		plans = append(plans, p)
	}

	// 1. Group and chunk with batch size 50
	batches := BuildBatches(plans, 50)

	// We expect:
	// Group 1 (default target): 100 items -> 2 batches (50, 50)
	// Group 2 (override target): 26 items -> 1 batch (26)
	// Default target group must come BEFORE override group
	if len(batches) != 3 {
		t.Fatalf("expected 3 batches, got %d", len(batches))
	}
	if len(batches[0].Items) != 50 || batches[0].TargetDir != "/115/电影" {
		t.Fatalf("unexpected batch 0: len=%d target=%s", len(batches[0].Items), batches[0].TargetDir)
	}
	if len(batches[1].Items) != 50 || batches[1].TargetDir != "/115/电影" {
		t.Fatalf("unexpected batch 1: len=%d target=%s", len(batches[1].Items), batches[1].TargetDir)
	}
	if len(batches[2].Items) != 26 || batches[2].TargetDir != "/115/电影/动画" {
		t.Fatalf("unexpected batch 2: len=%d target=%s", len(batches[2].Items), batches[2].TargetDir)
	}
}
