package context

type UserView struct {
	OCSs map[string]*OCS
	ToRs map[string]*ToR
}

func (u *UserView) FindToRByDeviceAndPort(device string, port int) *ToR {
	for _, toR := range u.ToRs {
		if toR.Device == device {
			if toR.HavePort(port) {
				return toR
			}
		}
	}

	return nil
}

type OCS struct {
	Device string
	Name   string
	Ports  map[int]*ConnectedTo
	Conn   *Connection
	Ip     string
}

type Connection struct {
	In_port  []int
	Out_port []int
}

func (o *OCS) HavePort(port int) bool {
	for ocsPort := range o.Ports {
		if port == ocsPort {
			return true
		}
	}
	return false
}

type ToR struct {
	Device string
	Name   string
	Id     int
	Ports  map[int]*ConnectedTo
}

func (t *ToR) HavePort(port int) bool {
	for torPort := range t.Ports {
		if port == torPort {
			return true
		}
	}
	return false
}
