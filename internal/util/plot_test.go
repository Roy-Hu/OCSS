package util

import (
	"testing"
)

func TestCreateHeatmap(t *testing.T) {
	traffic := map[string]map[string]int{
		"server1": {
			"server2": 1615807688,
			"server3": 0,
			"server4": 0,
			"server5": 0,
			"server6": 3262718,
		},
		"server2": {
			"server1": 2324114,
			"server3": 1361904484,
			"server4": 498,
			"server5": 0,
			"server6": 350,
		},
		"server3": {
			"server1": 0,
			"server2": 2818470,
			"server4": 1436170422,
			"server5": 0,
			"server6": 0,
		},
		"server4": {
			"server1": 0,
			"server2": 350,
			"server3": 34799162,
			"server5": 1613234190,
			"server6": 498,
		},
		"server5": {
			"server1": 0,
			"server2": 0,
			"server3": 0,
			"server4": 1089692,
			"server6": 1582508186,
		},
		"server6": {
			"server1": 1371072568,
			"server2": 498,
			"server3": 0,
			"server4": 424,
			"server5": 1587614,
		},
	}

	filename := "test_heatmap"

	if err := CreateHeatmap(traffic, filename); err != nil {
		t.Fatalf("CreateHeatmap() failed: %v", err)
	}

	t.Logf("Heatmap successfully written to %q", filename)
}
