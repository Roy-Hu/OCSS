package util

import (
	"testing"
)

func TestCreateHeatmap(t *testing.T) {
	traffic := map[string]map[string]int{
		"server1": {"server2": 2482000},
		"server2": {"server1": 1198000},
	}

	filename := "test_heatmap"

	if err := CreateHeatmap(traffic, filename); err != nil {
		t.Fatalf("CreateHeatmap() failed: %v", err)
	}

	t.Logf("Heatmap successfully written to %q", filename)
}
