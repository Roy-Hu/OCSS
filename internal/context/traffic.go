package context

import (
	"container/heap"
	"strconv"

	"github.com/comp590/ocss/internal/logger"
	"github.com/comp590/ocss/internal/util"
)

type TrafficMatrix map[string]map[string]int

type IterTraffic struct {
	Start TrafficMatrix
	End   TrafficMatrix
}

func (it IterTraffic) GetIterTrafficMatrix() TrafficMatrix {
	rtn := make(TrafficMatrix)

	for srcIp, dstIps := range it.Start {
		for dstIp, vol := range dstIps {
			if _, ok := rtn[srcIp]; !ok {
				rtn[srcIp] = make(map[string]int)
			}

			rtn[srcIp][dstIp] = it.End[srcIp][dstIp] - vol
		}
	}

	return rtn
}

func ConstructHeapMap(app string, op string, iter int, iterTraffic *IterTraffic) {
	traffic := make(map[string]map[string]int)

	iterTrafficMatrix := iterTraffic.GetIterTrafficMatrix()
	for srcIp, dstIps := range iterTrafficMatrix {
		for dstIp, vol := range dstIps {
			src_server := ocssContext.IpToServer[srcIp]
			dst_server := ocssContext.IpToServer[dstIp]

			if _, ok := traffic[src_server]; !ok {
				traffic[src_server] = make(map[string]int)
			}

			traffic[src_server][dst_server] = vol
		}
	}

	logger.HttpLog.Debugf("Creating heatmap for app %s, op %s,iter %d, traffic %v", app, op, iter, traffic)
	util.CreateHeatmap(traffic, app+"_"+op+"_"+strconv.Itoa(iter))
}

func (t TrafficMatrix) GetTopKLinkTraffic(k int) []TrafficPair {
	var pq PriorityQueue
	heap.Init(&pq)

	for srcServer, TrafficMatrix := range t {
		for dstServer, traffic := range TrafficMatrix {
			if srcServer == dstServer {
				continue
			}

			item := &TrafficPair{
				Src:     srcServer,
				Dst:     dstServer,
				Traffic: traffic,
			}

			// If we haven't reached 6 elements yet, just push.
			if pq.Len() < k {
				heap.Push(&pq, item)
			} else if traffic > pq[0].Traffic {
				// If new traffic is greater than the smallest in the heap, replace it.
				heap.Pop(&pq)
				heap.Push(&pq, item)
			}
		}
	}

	result := make([]TrafficPair, pq.Len())
	for i, item := range pq {
		result[i] = *item
	}

	return result
}

type TrafficPair struct {
	Src     string
	Dst     string
	Traffic int

	index int // Needed by heap.Interface methods.
}

// PriorityQueue implements heap.Interface and holds TrafficPair items.
type PriorityQueue []*TrafficPair

// Len is part of heap.Interface.
func (pq PriorityQueue) Len() int { return len(pq) }

// Less returns true if element i should sort before j.
// Here we implement a min-heap based on the traffic value.
func (pq PriorityQueue) Less(i, j int) bool {
	return pq[i].Traffic < pq[j].Traffic
}

// Swap swaps the elements at indexes i and j.
func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

// Push adds an element to the heap.
func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*TrafficPair)
	item.index = n
	*pq = append(*pq, item)
}

// Pop removes and returns the last element of the heap.
func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	item.index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}
