package context

import "github.com/comp590/ocss/internal/logger"

type OCS struct {
	Device         string
	Name           string
	PortConnToMap  map[int]*ConnectedTo
	Conn           *Connection
	PortServerConn map[int]*ConnServerInfo
	Ip             string
	Changed        bool
	Ports          []int
}

// TODO: Map is better
type Connection struct {
	In_port  []int
	Out_port []int
}

func (o *OCS) UpdateConn(conn *Connection) {
	for i := range len(conn.In_port) {
		if o.PortServerConn[conn.In_port[i]] == nil {
			logger.CtxLog.Errorf("%s Port %d is not setup as expected, check does ocs.port contain this port", o.Name, conn.In_port[i])
			return
		} else if o.PortServerConn[conn.Out_port[i]].Server == nil {
			logger.CtxLog.Debugf("%s Port %d is not used for connection berfore, setup now", o.Name, conn.Out_port[i])
			o.PortServerConn[conn.Out_port[i]].Server = make(map[string]bool)
		}

		if o.PortServerConn[conn.Out_port[i]] == nil {
			logger.CtxLog.Errorf("%s Port %d is not setup as expected, check does ocs.port contain this port", o.Name, conn.Out_port[i])
			return
		} else if o.PortServerConn[conn.Out_port[i]].Server == nil {
			logger.CtxLog.Debugf("%s Port %d is not used for connection berfore, setup now", o.Name, conn.Out_port[i])
			o.PortServerConn[conn.Out_port[i]].Server = make(map[string]bool)
		}
	}

	for i := range len(o.Conn.In_port) {
		o.PortServerConn[o.Conn.In_port[i]].Server[o.Name] = false
		o.PortServerConn[o.Conn.Out_port[i]].Server[o.Name] = false
	}

	for i := range len(conn.In_port) {
		for j := range len(o.Conn.In_port) {
			if conn.In_port[i] == o.Conn.In_port[j] {
				o.Conn.Out_port[j] = conn.Out_port[i]
			}
		}
	}

	cnts := make(map[int]int)

	for i := range len(o.Conn.In_port) {
		cnts[o.Conn.In_port[i]]++
		cnts[o.Conn.Out_port[i]]++
	}

	for idx, cnt := range cnts {
		if cnt != 1 {
			logger.CtxLog.Errorf("OCS Port %d is used more than once in OCS %s", idx, o.Name)
			return
		}
	}
	o.Changed = true
}

func (o *OCS) GetConnPorts() []int {
	return append(o.Conn.In_port, o.Conn.Out_port...)
}

func (o *OCS) HavePort(port int) bool {
	// logger.CtxLog.Infof("Checking if OCS %s has port %d", o.Name, port)
	for _, ocsPort := range o.Ports {
		if port == ocsPort {
			return true
		}
	}
	return false
}

func (o *OCS) ConnectedPort(portIn int) int {
	for i := range len(o.Conn.In_port) {
		if o.Conn.In_port[i] == portIn {
			return o.Conn.Out_port[i]
		} else if o.Conn.Out_port[i] == portIn {
			return o.Conn.In_port[i]
		}
	}

	return -1
}
