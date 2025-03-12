package context

import (
	"container/heap"
)

type UserView struct {
	OCSs      map[string]*OCS
	ToRs      map[string]*ToR
	Servers   map[string]*Server
	AppServer *AppServer
	Traffic   map[string]TrafficMatrix
}

func (u *UserView) FindToRByDeviceAndPort(device string, port int) *ToR {
	for _, toR := range u.ToRs {
		if toR.Device == device && toR.HavePort(port) {
			return toR
		}
	}

	return nil
}

func (u *UserView) FindOCSByDeviceAndPort(device string, port int) *OCS {
	for _, ocs := range u.OCSs {
		if ocs.Device == device && ocs.HavePort(port) {
			return ocs
		}
	}

	return nil
}

func (u *UserView) GetServerByIp(ip string) string {
	self := GetSelf()
	if server, ok := self.IpToServer[ip]; ok {
		return server
	}

	return ""
}
func (u *UserView) GetTopKLinkTraffic(app string, iter int, k int) []TrafficPair {
	var pq PriorityQueue
	heap.Init(&pq)

	for srcIp, TrafficMatrix := range u.AppServer.Apps[app].IterTrafficMatrix[iter] {
		for dstIp, traffic := range TrafficMatrix {
			src := u.GetServerByIp(srcIp)
			dst := u.GetServerByIp(dstIp)

			item := &TrafficPair{
				Src:     src,
				Dst:     dst,
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

type TrafficMatrix map[string]map[string]int
