package context

type UserView struct {
	OCSs      map[string]*OCS
	ToRs      map[string]*ToR
	Servers   map[string]*Server
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
