package ocss

import (
	"testing"

	ocss_context "github.com/comp590/ocss/internal/context"
)

func TestAllreducePolicyComplicated(t *testing.T) {
	// Define available TOR ports:
	// tor1 can support 3 servers, tor2 can support 2, and tor3 can support 1.
	availableTORPorts := map[string]int{
		"tor1": 3,
		"tor2": 2,
		"tor3": 1,
	}

	// Define the Start matrix as all zeros for servers A-F.
	start := ocss_context.TrafficMatrix{
		"A": {"B": 0, "C": 0, "D": 0, "E": 0, "F": 0},
		"B": {"A": 0, "C": 0, "D": 0, "E": 0, "F": 0},
		"C": {"A": 0, "B": 0, "D": 0, "E": 0, "F": 0},
		"D": {"A": 0, "B": 0, "C": 0, "E": 0, "F": 0},
		"E": {"A": 0, "B": 0, "C": 0, "D": 0, "F": 0},
		"F": {"A": 0, "B": 0, "C": 0, "D": 0, "E": 0},
	}

	// Define the End matrix with non-symmetric traffic values.
	end := ocss_context.TrafficMatrix{
		"A": {"B": 100, "C": 50, "D": 20, "E": 0, "F": 10},
		"B": {"A": 80, "C": 70, "D": 0, "E": 30, "F": 0},
		"C": {"A": 40, "B": 60, "D": 0, "E": 20, "F": 0},
		"D": {"A": 0, "B": 0, "C": 0, "E": 150, "F": 40},
		"E": {"A": 0, "B": 30, "C": 20, "D": 120, "F": 20},
		"F": {"A": 10, "B": 0, "C": 0, "D": 40, "E": 30},
	}

	// Create an IterTraffic instance.
	iterTraffic := ocss_context.IterTraffic{
		Start: start,
		End:   end,
	}

	// Create the Operation with six servers.
	op := &ocss_context.Operation{
		Op:          "allreduce",
		Servers:     []string{"A", "B", "C", "D", "E", "F"},
		IterTraffic: []ocss_context.IterTraffic{iterTraffic},
	}

	// Build a dummy UserView with server objects for each server.
	user := &ocss_context.UserView{
		Servers: map[string]*ocss_context.Server{
			"A": {},
			"B": {},
			"C": {},
			"D": {},
			"E": {},
			"F": {},
		},
	}

	// Initialize the policies and obtain the "allreduce" policy function.
	policies := initPolicies(user)
	allreducePolicy := policies["allreduce"]

	// Run the policy function.
	assignment := allreducePolicy(availableTORPorts, op)

	// Retrieve the computed traffic matrix.
	trafficMatrix := op.IterTraffic[len(op.IterTraffic)-1].GetIterTrafficMatrix()

	// Compute the localized traffic score using the computeScore function.
	score := computeScore(assignment, trafficMatrix)

	// In this scenario the best localized traffic is achieved by:
	// Grouping A, B, C together: (A<->B: 100+80, A<->C: 50+40, B<->C: 70+60) = 400.
	// Grouping D, E together: (D<->E: 150+120) = 270.
	// F remains alone (0).
	// Total expected score: 400 + 270 = 670.
	expectedScore := 670
	if score != expectedScore {
		t.Errorf("Expected localized traffic score %d, got %d", expectedScore, score)
	}

	// Verify that all servers are assigned.
	if len(assignment) != len(op.Servers) {
		t.Errorf("Expected assignment for %d servers, got %d", len(op.Servers), len(assignment))
	}

	// Build a mapping from tor names to the list of servers assigned.
	torGroups := make(map[string][]string)
	for server, tor := range assignment {
		torGroups[tor] = append(torGroups[tor], server)
	}

	// We expect three groups: one of size 3 (A, B, C), one of size 2 (D, E), and one of size 1 (F).
	groupSizes := make(map[int]bool)
	for _, group := range torGroups {
		groupSizes[len(group)] = true
	}

	if !groupSizes[3] || !groupSizes[2] || !groupSizes[1] {
		t.Errorf("Expected groups of sizes 3, 2, and 1 but got group sizes: %+v", torGroups)
	}

	// Verify that A, B, and C are in the same group.
	var groupABC string
	for tor, group := range torGroups {
		hasA, hasB, hasC := false, false, false
		for _, s := range group {
			if s == "A" {
				hasA = true
			}
			if s == "B" {
				hasB = true
			}
			if s == "C" {
				hasC = true
			}
		}
		if hasA && hasB && hasC {
			groupABC = tor
			break
		}
	}
	if groupABC == "" {
		t.Errorf("Expected servers A, B, and C to be on the same TOR, got assignment: %+v", assignment)
	}

	// Verify that D and E are in the same group.
	var groupDE string
	for tor, group := range torGroups {
		hasD, hasE := false, false
		for _, s := range group {
			if s == "D" {
				hasD = true
			}
			if s == "E" {
				hasE = true
			}
		}
		if hasD && hasE {
			groupDE = tor
			break
		}
	}
	if groupDE == "" {
		t.Errorf("Expected servers D and E to be on the same TOR, got assignment: %+v", assignment)
	}

	// Verify that F is alone in its group.
	var groupF []string
	for _, group := range torGroups {
		for _, s := range group {
			if s == "F" {
				groupF = group
				break
			}
		}
	}
	if len(groupF) != 1 {
		t.Errorf("Expected server F to be isolated, but group has servers: %v", groupF)
	}

	t.Logf("Complicated Assignment: %v", assignment)
	t.Logf("Complicated Localized Traffic Score: %d", score)
}

func TestAllreducePolicyZeroTraffic(t *testing.T) {
	// Define available TOR ports.
	// For this test, we'll use two TORs: tor1 (capacity 2) and tor2 (capacity 1).
	availableTORPorts := map[string]int{
		"tor1": 2,
		"tor2": 1,
	}

	// Define Start and End matrices where all traffic values are 0.
	start := ocss_context.TrafficMatrix{
		"A": {"B": 0, "C": 0},
		"B": {"A": 0, "C": 0},
		"C": {"A": 0, "B": 0},
	}
	end := ocss_context.TrafficMatrix{
		"A": {"B": 0, "C": 0},
		"B": {"A": 0, "C": 0},
		"C": {"A": 0, "B": 0},
	}

	// Create an IterTraffic instance.
	iterTraffic := ocss_context.IterTraffic{
		Start: start,
		End:   end,
	}

	// Create the Operation with three servers.
	op := &ocss_context.Operation{
		Op:          "allreduce",
		Servers:     []string{"A", "B", "C"},
		IterTraffic: []ocss_context.IterTraffic{iterTraffic},
	}

	// Build a dummy UserView with server objects.
	user := &ocss_context.UserView{
		Servers: map[string]*ocss_context.Server{
			"A": {},
			"B": {},
			"C": {},
		},
	}

	// Initialize policies and get the "allreduce" policy function.
	policies := initPolicies(user)
	allreducePolicy := policies["allreduce"]

	// Run the policy function.
	assignment := allreducePolicy(availableTORPorts, op)

	// Retrieve the computed traffic matrix.
	trafficMatrix := op.IterTraffic[len(op.IterTraffic)-1].GetIterTrafficMatrix()
	// Compute the localized traffic score.
	score := computeScore(assignment, trafficMatrix)

	// Check that all servers are assigned a TOR.
	if len(assignment) != len(op.Servers) {
		t.Errorf("Expected assignment for %d servers, got %d", len(op.Servers), len(assignment))
	}

	// Even though all traffic is zero, the score should be 0.
	if score != 0 {
		t.Errorf("Expected localized traffic score 0, got %d", score)
	}

	t.Logf("Zero Traffic Assignment: %v", assignment)
	t.Logf("Zero Traffic Localized Score: %d", score)
}
