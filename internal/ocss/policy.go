package ocss

import (
	ocss_context "github.com/comp590/ocss/internal/context"

	"github.com/comp590/ocss/internal/logger"
)

func computeScore(assignment map[string]string, traffic ocss_context.TrafficMatrix) int {
	score := 0
	for s1, m := range traffic {
		// Ensure s1 is assigned.
		tor1, ok1 := assignment[s1]
		if !ok1 {
			continue
		}
		for s2, val := range m {
			if tor2, ok2 := assignment[s2]; ok2 && tor1 == tor2 {
				score += val
			}
		}
	}
	return score
}

func initPolicies(user *ocss_context.UserView) map[string]ocss_context.PolicyFunc {

	MyActions := make(map[string]ocss_context.PolicyFunc)
	// Init end

	MyActions["allreduce"] = ocss_context.PolicyFunc(func(torCapacity map[string]int, op *ocss_context.Operation) map[string]string {
		servers := make(map[string]*ocss_context.Server)
		serverMap := make(map[string]bool)
		for _, server := range op.Servers {
			serverMap[server] = true
		}

		logger.ActionLog.Errorf("Policy for: %v", op.Op)
		logger.ActionLog.Errorf("Servers: %v", op.Servers)

		for server, ok := range serverMap {
			if ok {
				servers[server] = user.Servers[server]
			}
		}

		traffic := op.IterTraffic[len(op.IterTraffic)-1].GetIterTrafficMatrix()

		bestAssignment := make(map[string]string)
		bestScore := 0

		var rec func(index int, currentAssignment map[string]string, remaining map[string]int)
		rec = func(index int, currentAssignment map[string]string, remaining map[string]int) {
			// If all servers have been assigned, compute the score.
			if index == len(op.Servers) {
				score := computeScore(currentAssignment, traffic)
				if score >= bestScore {
					bestScore = score
					// Copy the current assignment to bestAssignment.
					bestAssignment = make(map[string]string)
					for k, v := range currentAssignment {
						bestAssignment[k] = v
					}
				}
				return
			}

			// Get the current server to assign.
			server := op.Servers[index]

			// Try assigning this server to each TOR that still has capacity.
			for tor, cap := range remaining {
				if cap > 0 {
					// Assign server to the TOR.
					currentAssignment[server] = tor
					remaining[tor]-- // Use one port on this TOR.

					// Recurse to assign the next server.
					rec(index+1, currentAssignment, remaining)

					// Backtrack: remove the assignment and restore capacity.
					remaining[tor]++
					delete(currentAssignment, server)

				}
			}
		}

		remaining := make(map[string]int)
		for tor, cap := range torCapacity {
			remaining[tor] = cap
		}

		// Start the recursive search.
		rec(0, make(map[string]string), remaining)

		return bestAssignment
	})

	return MyActions
}
