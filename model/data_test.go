package model

import (
	"testing"
)

func TestCleanCollectionName(t *testing.T) {
	col := CleanCollectionName("tasks_770_")
	if col != "tasks" {
		t.Errorf("expected col to be tasks got %s", col)
	}
}
