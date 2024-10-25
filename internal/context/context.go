package context

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/comp590/ocss/internal/logger"
	"github.com/comp590/ocss/pkg/factory"
	"gopkg.in/yaml.v2"
)

var ocssContext OCSSContext

type DeviceType string

const (
	SERVER         DeviceType = "SERVER"
	SWITCH         DeviceType = "SWITCH"
	OPTICAL_SWITCH DeviceType = "OPTICAL_SWITCH"
)

type OCSSContext struct {
	Switches         map[string]*Switch
	Servers          map[string]*Server
	OCSs             map[string]*OCS
	DeviceType       map[string]DeviceType
	ForwardingTables map[string][]*Forward
	UserView         *UserView
}

type Forward struct {
	Device   string
	SrcPort  int
	DestPort int
	Ip       string
}

type Server struct {
	Name  string
	Ip    string
	Ports map[int]*ConnectedTo
}

type ConnectedTo struct {
	Device string
	Name   string
	Port   int
	Server []string
}

type Link struct {
	Source      string
	SrcPort     int
	Destination string
	DestPort    int
}

func Init() {
	ocssContext = OCSSContext{
		Switches:         make(map[string]*Switch),
		Servers:          make(map[string]*Server),
		OCSs:             make(map[string]*OCS),
		DeviceType:       make(map[string]DeviceType),
		ForwardingTables: make(map[string][]*Forward),
		UserView: &UserView{
			OCSs: make(map[string]*OCS),
			ToRs: make(map[string]*ToR),
		},
	}

	configuration := factory.OcssConfig.Configuration

	// Initialize servers
	for _, s := range configuration.NetworkManager.Servers {
		ocssContext.DeviceType[s.Name] = SERVER
		server := &Server{
			Name:  s.Name,
			Ip:    s.IP,
			Ports: make(map[int]*ConnectedTo),
		}
		server.Ports[s.Port] = &ConnectedTo{}

		ocssContext.Servers[s.Name] = server
	}

	// Initialize switches
	for _, ps := range configuration.NetworkManager.PacketSwitchs {
		ocssContext.DeviceType[ps.Name] = SWITCH
		s := &Switch{
			Name:  ps.Name,
			Id:    ps.ID,
			Ports: make(map[int]*ConnectedTo),
		}

		ports, err := ParsePorts(ps.Ports)
		if err != nil {
			logger.CtxLog.Errorf("Error parsing ports for switch %s: %s", ps.Name, err)
			continue
		}
		for _, p := range ports {
			s.Ports[p] = &ConnectedTo{}
		}

		ocssContext.Switches[ps.Name] = s
	}

	for _, os := range configuration.NetworkManager.OpticalSwitchs {
		ocssContext.DeviceType[os.Name] = OPTICAL_SWITCH
		o := &OCS{
			Device: os.Name,
			Name:   os.Name,
			Ports:  make(map[int]*ConnectedTo),
			Conn:   &Connection{},
			Ip:     os.Ip,
		}

		ports, err := ParsePorts(os.Ports)
		if err != nil {
			logger.CtxLog.Errorf("Error parsing ports for switch %s: %s", os.Name, err)
			continue
		}
		for _, p := range ports {
			o.Ports[p] = &ConnectedTo{}
		}

		ocssContext.OCSs[os.Name] = o
	}

	for _, ut := range configuration.User.ToR {
		for _, t := range ut.ToRs {
			ocssContext.UserView.ToRs[t.Name] = &ToR{
				Device: ut.Name,
				Name:   t.Name,
				Id:     ocssContext.Switches[ut.Name].Id,
				Ports:  make(map[int]*ConnectedTo),
			}

			if _, ok := ocssContext.Switches[ut.Name]; !ok {
				logger.CtxLog.Errorf("Switch %s not found for ToR %s", ut.Name, t.Name)
				continue
			}

			for _, port := range t.Ports {
				ocssContext.UserView.ToRs[t.Name].Ports[port] = ocssContext.Switches[ut.Name].Ports[port]
			}
		}
	}

	// Initialize hardware links and build server-to-switch-port mapping
	for _, l := range configuration.NetworkManager.Links {
		src := l.Source
		dst := l.Destination

		if src == dst {
			logger.CtxLog.Errorf("Link %s -> %s is invalid: source and destination are the same", src, dst)
			continue
		}

		for i := range len(l.SourcePorts) {
			srcPort := l.SourcePorts[i]
			dstPort := l.DestinationPorts[i]

			connTo := &ConnectedTo{
				Device: dst,
				Port:   dstPort,
			}

			for tor_name, tor := range ocssContext.UserView.ToRs {
				if tor.Device == l.Destination {
					if _, ok := tor.Ports[l.DestinationPorts[i]]; ok {
						connTo.Name = tor_name
						tor.Ports[l.DestinationPorts[i]] = connTo
					}
				}
			}

			if ocssContext.DeviceType[dst] == SERVER {
				connTo.Server = append(connTo.Server, dst)
			}

			switch ocssContext.DeviceType[src] {
			case SWITCH:
				ocssContext.Switches[src].Ports[srcPort] = connTo
			case OPTICAL_SWITCH:
				ocssContext.OCSs[src].Ports[srcPort] = connTo
			case SERVER:
				ocssContext.Servers[src].Ports[srcPort] = connTo
			default:
				logger.CtxLog.Errorf("Link %s -> %s is invalid: source is not a switch, ocs, or server", src, dst)
				continue
			}

			connTo = &ConnectedTo{
				Device: src,
				Port:   srcPort,
				Server: []string{},
			}

			for tor_name, tor := range ocssContext.UserView.ToRs {
				if tor.Device == l.Source {
					if _, ok := tor.Ports[l.SourcePorts[i]]; ok {
						connTo.Name = tor_name
						tor.Ports[l.SourcePorts[i]] = connTo
					}
				}
			}

			if ocssContext.DeviceType[src] == SERVER {
				connTo.Server = append(connTo.Server, src)
			}

			switch ocssContext.DeviceType[dst] {
			case SWITCH:
				ocssContext.Switches[dst].Ports[dstPort] = connTo
			case OPTICAL_SWITCH:
				ocssContext.OCSs[dst].Ports[dstPort] = connTo
			case SERVER:
				ocssContext.Servers[dst].Ports[dstPort] = connTo
			default:
				logger.CtxLog.Errorf("Link %s -> %s is invalid: source is not a switch, ocs, or server", src, dst)
				continue
			}
		}
	}

	for _, uo := range configuration.User.OCS {
		for _, s := range uo.Sections {
			ocssContext.UserView.OCSs[s.Name] = &OCS{
				Device: uo.Name,
				Name:   s.Name,
				Ports:  make(map[int]*ConnectedTo),
				Conn: &Connection{
					In_port:  s.SourcePorts,
					Out_port: s.DestinationPorts,
				},
			}

			if _, ok := ocssContext.OCSs[uo.Name]; !ok {
				logger.CtxLog.Errorf("OCS %s not found", uo.Name)
				continue
			}

			for port, connTo := range ocssContext.OCSs[uo.Name].Ports {
				for i := range s.SourcePorts {
					if s.SourcePorts[i] == port {
						ocssContext.UserView.OCSs[s.Name].Ports[port] = connTo
					} else if s.DestinationPorts[i] == port {
						ocssContext.UserView.OCSs[s.Name].Ports[port] = connTo
					}
				}
			}

			ocssContext.OCSs[uo.Name].Conn.In_port = append(ocssContext.OCSs[uo.Name].Conn.In_port, s.SourcePorts...)
			ocssContext.OCSs[uo.Name].Conn.Out_port = append(ocssContext.OCSs[uo.Name].Conn.Out_port, s.DestinationPorts...)
		}
	}

	for _, l := range configuration.User.Link {
		if ocs, ok := ocssContext.UserView.OCSs[l.Source]; ok {
			for i := range l.SourcePorts {
				ocs.Ports[l.SourcePorts[i]] = &ConnectedTo{
					Device: l.Destination,
					Port:   l.DestinationPorts[i],
				}
				if ocssContext.DeviceType[l.Destination] == SERVER {
					ocs.Ports[l.SourcePorts[i]].Server = append(ocs.Ports[l.SourcePorts[i]].Server, l.Destination)
					ocssContext.OCSs[ocs.Device].Ports[l.SourcePorts[i]].Server = append(ocssContext.OCSs[ocs.Device].Ports[l.SourcePorts[i]].Server, l.Destination)
				}
			}

			continue
		}

		if ocs, ok := ocssContext.UserView.OCSs[l.Destination]; ok {
			for i := range l.DestinationPorts {
				ocssContext.UserView.OCSs[l.Destination].Ports[l.DestinationPorts[i]] = &ConnectedTo{
					Device: l.Source,
					Port:   l.SourcePorts[i],
				}
				if ocssContext.DeviceType[l.Source] == SERVER {
					ocs.Ports[l.DestinationPorts[i]].Server = append(ocs.Ports[l.DestinationPorts[i]].Server, l.Source)
					ocssContext.OCSs[ocs.Device].Ports[l.DestinationPorts[i]].Server = append(ocssContext.OCSs[ocs.Device].Ports[l.DestinationPorts[i]].Server, l.Source)
				}
			}
			continue
		}

		logger.CtxLog.Errorf("Link %s -> %s is invalid: one of the source and destination should be a ocs", l.Source, l.Destination)
	}
}

// ParsePorts parses a string of ports like "1000:2000,3001" into a slice of integers.
func ParsePorts(portStr string) ([]int, error) {
	var ports []int
	segments := strings.Split(portStr, ",")

	for _, segment := range segments {
		if strings.Contains(segment, ":") {
			// This is a range like "1000:2000"
			rangeParts := strings.Split(segment, ":")
			if len(rangeParts) != 2 {
				return nil, fmt.Errorf("invalid range format: %s", segment)
			}

			start, err1 := strconv.Atoi(rangeParts[0])
			end, err2 := strconv.Atoi(rangeParts[1])

			if err1 != nil || err2 != nil || start > end {
				return nil, fmt.Errorf("invalid range: %s", segment)
			}

			// Add the ports in the range to the list
			for i := start; i <= end; i++ {
				ports = append(ports, i)
			}
		} else {
			// This is a single port like "3001"
			port, err := strconv.Atoi(segment)
			if err != nil {
				return nil, fmt.Errorf("invalid port: %s", segment)
			}
			ports = append(ports, port)
		}
	}

	return ports, nil
}

// GetSelf returns the OCSSContext instance.
func GetSelf() *OCSSContext {
	return &ocssContext
}

func Print() {
	data, err := yaml.Marshal(ocssContext)
	if err != nil {
		logger.CtxLog.Errorf("Error marshalling OCSS context: %s", err)
		return
	}

	logger.CtxLog.Infof("OCSS context initialized: %s", string(data))
}
