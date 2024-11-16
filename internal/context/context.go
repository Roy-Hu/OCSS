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

type RULE_STATUS string

const (
	CREATE RULE_STATUS = "CREATE"
	UPDATE RULE_STATUS = "UPDATE"
	DELETE RULE_STATUS = "DELETE"
	ACTIVE RULE_STATUS = "ACTIVE"
)

type OCSSContext struct {
	Switches   map[string]*Switch
	Servers    map[string]*Server
	OCSs       map[string]*OCS
	DeviceType map[string]DeviceType
	UserView   *UserView
	States     map[string]*State
}

type ConnServerInfo struct {
	FixedServer bool
	Server      map[string]bool
}

type Forward struct {
	Device   string
	SrcPort  int
	DestPort int
	DstIp    string
	SrcIp    string
	Status   RULE_STATUS
}

type Server struct {
	Name          string
	Ip            string
	PortConnToMap map[int]*ConnectedTo
}

type ConnectedTo struct {
	Device string
	Name   string
	Port   int
}

type Link struct {
	Source      string
	SrcPort     int
	Destination string
	DestPort    int
}

func Init() error {
	ocssContext = OCSSContext{
		Switches:   make(map[string]*Switch),
		Servers:    make(map[string]*Server),
		OCSs:       make(map[string]*OCS),
		DeviceType: make(map[string]DeviceType),
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
			Name:          s.Name,
			Ip:            s.IP,
			PortConnToMap: make(map[int]*ConnectedTo),
		}
		server.PortConnToMap[s.Port] = &ConnectedTo{}

		ocssContext.Servers[s.Name] = server
	}

	// Initialize switches
	for _, ps := range configuration.NetworkManager.PacketSwitchs {
		ocssContext.DeviceType[ps.Device] = SWITCH
		s := &Switch{
			Device:         ps.Device,
			Id:             ps.ID,
			PortConnToMap:  make(map[int]*ConnectedTo),
			ForwardingRule: make(map[string]*Forward),
		}

		ports, err := ParsePorts(ps.Ports)
		if err != nil {
			return fmt.Errorf("Error parsing ports for switch %s: %s", ps.Device, err)
		}
		for _, p := range ports {
			s.PortConnToMap[p] = &ConnectedTo{}
		}

		ocssContext.Switches[ps.Device] = s
	}

	for _, os := range configuration.NetworkManager.OpticalSwitchs {
		ocssContext.DeviceType[os.Device] = OPTICAL_SWITCH
		o := &OCS{
			Device:        os.Device,
			PortConnToMap: make(map[int]*ConnectedTo),
			Conn:          &Connection{},
			Ip:            os.Ip,
		}

		ports, err := ParsePorts(os.Ports)
		if err != nil {
			return fmt.Errorf("Error parsing ports for switch %s: %s", os.Device, err)
		}
		for _, p := range ports {
			o.PortConnToMap[p] = &ConnectedTo{}
		}

		ocssContext.OCSs[os.Device] = o
	}

	// Initialize ToRs
	for _, ut := range configuration.User.ToR {
		for _, t := range ut.ToRs {
			if _, ok := ocssContext.DeviceType[ut.Device]; !ok {
				return fmt.Errorf("Device %s not found for ToR %s", ut.Device, t.Name)
			}

			ocssContext.UserView.ToRs[t.Name] = &ToR{
				Device:         ut.Device,
				Name:           t.Name,
				Id:             ocssContext.Switches[ut.Device].Id,
				PortConnToMap:  make(map[int]*ConnectedTo),
				PortServerConn: make(map[int]*ConnServerInfo),
			}

			if _, ok := ocssContext.Switches[ut.Device]; !ok {
				return fmt.Errorf("Switch %s not found for ToR %s", ut.Device, t.Name)
			}

			tor_ports, err := ParsePorts(t.Ports)
			if err != nil {
				return fmt.Errorf("Error parsing ports for ToR %s: %s", t.Name, err)
			}

			for _, port := range tor_ports {
				if _, ok := ocssContext.Switches[ut.Device].PortConnToMap[port]; !ok {
					return fmt.Errorf("Port %d not found for ToR %s", port, t.Name)
				}

				ocssContext.UserView.ToRs[t.Name].PortConnToMap[port] = ocssContext.Switches[ut.Device].PortConnToMap[port]
				ocssContext.UserView.ToRs[t.Name].PortServerConn[port] = &ConnServerInfo{}
			}
		}
	}

	// Initialize hardware links and build server-to-switch-port mapping
	for _, l := range configuration.NetworkManager.Links {
		src := l.Source
		dst := l.Destination

		if src == dst {
			return fmt.Errorf("Link %s -> %s is invalid: source and destination are the same", src, dst)
		}

		logger.CtxLog.Infof("Link %s -> %s", src, dst)
		srcDst := []string{src, dst}
		s_ports, d_ports, err := getSrcDstPortsFromConfig(l.SourcePorts, l.DestinationPorts)
		if err != nil {
			return fmt.Errorf("Error parsing ports for link %s -> %s: %s", l.SourcePorts, l.DestinationPorts, err)
		}
		logger.CtxLog.Infof("Ports: %v -> %v", s_ports, d_ports)
		for i := range len(s_ports) {
			srcDstPorts := []int{s_ports[i], d_ports[i]}
			for j, port := range srcDstPorts {
				var connTo *ConnectedTo

				switch ocssContext.DeviceType[srcDst[1-j]] {
				case SWITCH:
					connTo = ocssContext.Switches[srcDst[1-j]].PortConnToMap[srcDstPorts[1-j]]
				case OPTICAL_SWITCH:
					connTo = ocssContext.OCSs[srcDst[1-j]].PortConnToMap[srcDstPorts[1-j]]
				case SERVER:
					connTo = ocssContext.Servers[srcDst[1-j]].PortConnToMap[srcDstPorts[1-j]]
				default:
					return fmt.Errorf("Link %s -> %s is invalid: source is not a switch, ocs, or server", src, dst)
				}

				if connTo == nil {
					return fmt.Errorf("Link %s -> %s is invalid: port %d not found on %s", src, dst, srcDstPorts[1-j], srcDst[1-j])
				}

				connTo.Device = srcDst[j]
				connTo.Port = port
			}
		}
	}

	for _, uo := range configuration.User.OCS {
		for _, s := range uo.Sections {
			s_ports, d_ports, err := getSrcDstPortsFromConfig(s.SourcePorts, s.DestinationPorts)
			if err != nil {
				return fmt.Errorf("Error parsing ports for link %s -> %s: %s", s.SourcePorts, s.DestinationPorts, err)
			}

			ports, err := ParsePorts(s.Ports)
			if err != nil {
				return fmt.Errorf("Error parsing ports for OCS %s: %s", s.Ports, err)
			}

			ocssContext.UserView.OCSs[s.Name] = &OCS{
				Device:        uo.Device,
				Name:          s.Name,
				PortConnToMap: make(map[int]*ConnectedTo),
				Conn: &Connection{
					In_port:  s_ports,
					Out_port: d_ports,
				},
				Ports:          ports,
				PortServerConn: make(map[int]*ConnServerInfo),
			}

			for _, port := range ports {
				ocssContext.UserView.OCSs[s.Name].PortServerConn[port] = &ConnServerInfo{}
			}

			if _, ok := ocssContext.OCSs[uo.Device]; !ok {
				return fmt.Errorf("OCS %s not found", uo.Device)
			}

			logger.CtxLog.Infof("OCS %s ports %v", s.Name, ports)
			// TODO: consider all ports
			for _, p := range ports {
				if _, ok := ocssContext.OCSs[uo.Device].PortConnToMap[p]; !ok {
					return fmt.Errorf("Port %d not found for OCS %s", p, s.Name)
				}

				ocssContext.UserView.OCSs[s.Name].PortConnToMap[p] = ocssContext.OCSs[uo.Device].PortConnToMap[p]
			}
		}
	}

	for _, l := range configuration.NetworkManager.Links {
		src := l.Source
		dst := l.Destination

		if src == dst {
			return fmt.Errorf("Link %s -> %s is invalid: source and destination are the same", src, dst)
		}

		srcDst := []string{src, dst}
		s_ports, d_ports, err := getSrcDstPortsFromConfig(l.SourcePorts, l.DestinationPorts)
		if err != nil {
			return fmt.Errorf("Error parsing ports for link %s -> %s: %s", l.SourcePorts, l.DestinationPorts, err)
		}

		for i := range len(s_ports) {
			srcDstPorts := []int{s_ports[i], d_ports[i]}
			for j, port := range srcDstPorts {
				var connTo *ConnectedTo
				switch ocssContext.DeviceType[srcDst[1-j]] {
				case SWITCH:
					connTo = ocssContext.Switches[srcDst[1-j]].PortConnToMap[srcDstPorts[1-j]]
				case OPTICAL_SWITCH:
					connTo = ocssContext.OCSs[srcDst[1-j]].PortConnToMap[srcDstPorts[1-j]]
				case SERVER:
					connTo = ocssContext.Servers[srcDst[1-j]].PortConnToMap[srcDstPorts[1-j]]
				default:
					return fmt.Errorf("Link %s -> %s is invalid: source is not a switch, ocs, or server", src, dst)
				}

				switch ocssContext.DeviceType[srcDst[j]] {
				case SWITCH:
					if tor := ocssContext.UserView.FindToRByDeviceAndPort(srcDst[j], port); tor != nil {
						connTo.Name = tor.Name
					} else {
						logger.CtxLog.Debugf("ToR not found for switch %s port %d, it should be a packet switch used to connect ocs and server", srcDst[j], port)
					}
				case OPTICAL_SWITCH:
					if ocs := ocssContext.UserView.FindOCSByDeviceAndPort(srcDst[j], port); ocs != nil {
						connTo.Name = ocs.Name
					} else {
						logger.CtxLog.Errorf("OCS not found for switch %s port %d", srcDst[j], port)
					}
				case SERVER:
					connTo.Name = srcDst[j]
				default:
					return fmt.Errorf("Link %s -> %s is invalid: destination is not a switch, ocs, or server", src, dst)
				}
			}
		}
	}

	for _, l := range configuration.User.Link {
		s_ports, d_ports, err := getSrcDstPortsFromConfig(l.SourcePorts, l.DestinationPorts)
		if err != nil {
			return fmt.Errorf("Error parsing ports for link %s -> %s: %s", l.SourcePorts, l.DestinationPorts, err)
		}

		if ocs, ok := ocssContext.UserView.OCSs[l.Source]; ok {
			for i := range s_ports {
				ocs.PortConnToMap[s_ports[i]] = &ConnectedTo{
					Device: l.Destination,
					Port:   d_ports[i],
				}
			}

			continue
		}

		if _, ok := ocssContext.UserView.OCSs[l.Destination]; ok {
			for i := range d_ports {
				ocssContext.UserView.OCSs[l.Destination].PortConnToMap[d_ports[i]] = &ConnectedTo{
					Device: l.Source,
					Port:   s_ports[i],
				}
			}
			continue
		}

		return fmt.Errorf("Link %s -> %s is invalid: one of the source and destination should be a ocs", l.Source, l.Destination)
	}

	return nil
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

func PrintFowardingRule() {
	rules := make(map[string]map[int]map[int][]string)
	for _, s := range ocssContext.Switches {
		for _, f := range s.ForwardingRule {
			if _, ok := rules[f.Device]; !ok {
				rules[f.Device] = make(map[int]map[int][]string)
			}

			if _, ok := rules[f.Device][f.SrcPort]; !ok {
				rules[f.Device][f.SrcPort] = make(map[int][]string)
			}

			if f.DstIp == "" {
				continue
			}

			rules[f.Device][f.SrcPort][f.DestPort] = append(rules[f.Device][f.SrcPort][f.DestPort], f.DstIp)
		}
	}

	for device, srcPorts := range rules {
		logger.CtxLog.Errorf("Device: %s\n", device)
		for srcPort, destPorts := range srcPorts {
			for destPort, ips := range destPorts {
				logger.CtxLog.Warnf("%d -> %d %s", srcPort, destPort, ips)
			}
		}
	}
}

func Print() {
	data, err := yaml.Marshal(ocssContext.UserView)
	if err != nil {
		logger.CtxLog.Errorf("Error marshalling OCSS context: %s", err)
		return
	}

	logger.CtxLog.Infof("OCSS context initialized: %s", string(data))
}

func getSrcDstPortsFromConfig(srcPortsStr string, dstPortsStr string) ([]int, []int, error) {
	srcPorts, err := ParsePorts(srcPortsStr)
	if err != nil {
		return nil, nil, fmt.Errorf("Error parsing source ports for link %s -> %s: %s", srcPortsStr, dstPortsStr, err)
	}

	dstPorts, err := ParsePorts(dstPortsStr)
	if err != nil {
		return nil, nil, fmt.Errorf("Error parsing destination ports for link %s -> %s: %s", srcPortsStr, dstPortsStr, err)
	}

	if len(srcPorts) != len(dstPorts) {
		return nil, nil, fmt.Errorf("Links %s -> %s is invalid: source and destination ports do not match", srcPortsStr, dstPortsStr)
	}

	return srcPorts, dstPorts, nil
}

func (c *ConnServerInfo) CopyConnServerInfo(src *ConnServerInfo) {
	c.Server = make(map[string]bool)
	for server, ok := range src.Server {
		if ok {
			c.Server[server] = true
		}
	}
}

func (c *ConnServerInfo) AddConnServerInfo(src *ConnServerInfo) {
	if c.Server == nil {
		c.Server = make(map[string]bool)
	}
	for server, ok := range src.Server {
		if ok {
			c.Server[server] = true
		}
	}
}
