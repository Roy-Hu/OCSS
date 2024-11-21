package context

type ToR struct {
	Device         string
	Name           string
	Id             int
	PortConnToMap  map[int]*ConnectedTo
	PortServerConn map[int]*ConnServerInfo
}

func (t *ToR) HavePort(port int) bool {
	for torPort := range t.PortConnToMap {
		if port == torPort {
			return true
		}
	}
	return false
}

func (t *ToR) ConnectedServers() []string {
	servers := make([]string, 0)
	for _, conn := range t.PortServerConn {
		for server, ok := range conn.Server {
			if ok {
				servers = append(servers, server)
			}
		}
	}
	return servers
}
