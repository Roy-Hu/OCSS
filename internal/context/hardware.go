package context

type HardwareType int
type Protocol int

const (
	OCS HardwareType = iota
	TOR
	SERVER
)

const (
	UDP Protocol = iota
	TCP
)

type Port struct {
	PortNum int
}

type Node struct {
	Name       string
	Type       HardwareType
	Ip         string
	PortNumIn  int
	PortNumOut int

	// Only for OCS
	Splitor     int
	Connections map[int]int

	// private
	Ports []Port
}

type Link struct {
	Name     string
	Source   Node
	SrcPort  Port
	Dest     Node
	DestPort Port
}

type Hardware struct {
	Nodes map[string]Node
	Links []Link
}

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
	Links    []Link
	Traffic  chan int
	Protocol Protocol
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
