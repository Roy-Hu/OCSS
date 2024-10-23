package context

type SrcDst struct {
	Source      string
	Destination string
}

type LinkTraffic struct {
	Name    string
	Total   chan int // total traffic
	Src2Dst map[SrcDst]int
	Loss    chan float32
	Latency chan float32
}

type Path struct {
	Links   []Link
	Traffic chan int
}

type ServerPair struct {
	Src2Dst SrcDst
	Paths   []Path
	Totals  chan int
	Loss    chan float32
	Latency chan float32
}

type Traffic struct {
	LinkTraffics map[string]LinkTraffic
	ServerPairs  map[string]ServerPair
}
