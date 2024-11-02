package ocss

import (
	"fmt"

	ocss_context "github.com/comp590/ocss/internal/context"
	"github.com/comp590/ocss/internal/logger"
	"github.com/comp590/ocss/pkg/app"
)

type ProcessorOCSS interface {
	app.App

	Forwarder() *Forwarder
}

type Processor struct {
	ProcessorOCSS
}

func NewProcessor(ocss ProcessorOCSS) (*Processor, error) {
	o := &Processor{
		ProcessorOCSS: ocss,
	}

	return o, nil
}

func (p *Processor) CreateForwardingTables() error {
	self := ocss_context.GetSelf()
	ocs_finished_setup := make(map[string]bool)

	// Note: ports connect through physical link should share the same server pointer
	// for logical edge ocs that connects to the server
	for _, ocs := range self.UserView.OCSs {
		for ocs_port, connTo := range ocs.Ports {
			logger.ProcessorLog.Infof("OCS [%s] Port [%d] -> [%s] Port [%d]", ocs.Name, ocs_port, connTo.Device, connTo.Port)

			// ocs logically connects to the server
			// ocs_port is the port in ocs that connects to the server
			if self.DeviceType[connTo.Device] == ocss_context.SERVER {
				// TODO: Support two or more switch between ocs and server
				server := connTo.Device
				// what the server actually connects to
				device := self.Servers[server].Ports[connTo.Port].Device
				in_port := self.Servers[server].Ports[connTo.Port].Port

				// All others port that connects to this ocs port should also connect to the server
				switch self.DeviceType[device] {
				case ocss_context.OPTICAL_SWITCH:
					// TODO: Support optical switch
					logger.ProcessorLog.Warnf("Currently Server must connect to a switch")
				// server actually connects to a switch instead of the ocs
				// TODO: currently only support one switch between server and ocs
				case ocss_context.SWITCH:

					sw := self.Switches[device]
					for port, sw_connTo := range sw.Ports {
						logger.ProcessorLog.Infof("Switch [%s] Port [%d] -> Server [%s] Port [%d]", device, port, sw_connTo.Device, sw_connTo.Port)

						// switch port connects to the current ocs
						if sw_connTo.Device == ocs.Device && sw_connTo.Port == ocs_port {
							logger.ProcessorLog.Warnf("OCS [%s] Port [%d] -> Switch [%s] Port [%d]", ocs.Name, ocs_port, device, port)
							out_port := port

							// the in port connects to the server and the out port connects to the ocs beed to be treated as a psycial link
							self.ForwardingTables[device] = append(self.ForwardingTables[device], &ocss_context.Forward{
								Device:   device,
								SrcPort:  in_port,
								DestPort: out_port,
							})

							// server -> sw in port -> sw out port -> ocs in port share the same sever pointer
							// since we logically connect the server to the ocs
							// the forwarding of in/out port of the sw be treated as psycial link
							sw_connTo.Server = sw.Ports[in_port].Server
							connTo.Server = sw_connTo.Server

							for i := range len(ocs.Conn.In_port) {
								ocs_out_port := -1
								if ocs.Conn.In_port[i] == ocs_port {
									ocs_out_port = ocs.Conn.Out_port[i]
								} else if ocs.Conn.Out_port[i] == ocs_port {
									ocs_out_port = ocs.Conn.In_port[i]
								} else {
									continue
								}

								self.OCSs[ocs.Device].Ports[ocs_out_port].Server = make(map[string]bool)
								self.OCSs[ocs.Device].Ports[ocs_out_port].Server[server] = true
								if self.DeviceType[ocs.Ports[ocs_out_port].Device] == ocss_context.SWITCH {
									// the switch port that connects to ocs_out_port should also connect to the server
									// but it is not psycially connected to the ocs in port(it may change if the user change the connection in ocs)
									// so we need to set the server pointer
									self.Switches[ocs.Ports[ocs_out_port].Device].Ports[ocs.Ports[ocs_out_port].Port].Server = self.OCSs[ocs.Device].Ports[ocs_out_port].Server
								} else {
									return fmt.Errorf("Currently OCS should connect to a switch")
								}
							}
							continue
						}
					}
				}

				ocs_finished_setup[ocs.Name] = true
			}
		}
	}
	ocss_context.Print()
	// for logical core ocs that connects to the switch
	for _, ocs := range self.UserView.OCSs {
		if !ocs_finished_setup[ocs.Name] {
			for i := range len(ocs.Conn.In_port) {
				inPort := ocs.Conn.In_port[i]
				outPort := ocs.Conn.Out_port[i]
				ports := []*ocss_context.ConnectedTo{ocs.Ports[inPort], ocs.Ports[outPort]}
				for _, connTo := range ports {
					device := connTo.Device
					// TODO: What if OCS connected to another OCS?
					// ocs in/out port connects to a switch
					if self.DeviceType[device] == ocss_context.SWITCH {
						port := connTo.Port
						tor := self.UserView.FindToRByDeviceAndPort(device, port)
						if tor == nil {
							logger.ProcessorLog.Errorf("Switch [%s] Port [%d] is not a ToR", device, port)
							continue
						}

						for dst_port, dst_connTo := range tor.Ports {
							// find the tor port that connects to the current ocs port
							if dst_port == port {
								continue
							}

							// May delete?
							if ocs.HavePort(dst_port) {
								logger.ProcessorLog.Errorf("Switch [%s] Port [%d] and Port [%d] both connect to OCS [%s], skip ", connTo.Device, dst_port, port, ocs.Device)
								continue
							}
							// since the switch port connects to the ocs port, the switch port should also connect to the server
							self.Switches[device].Ports[port].Server = dst_connTo.Server
							connTo.Server = dst_connTo.Server
							for server, ok := range dst_connTo.Server {
								if ok {
									self.ForwardingTables[device] = append(self.ForwardingTables[device], &ocss_context.Forward{
										Device:   device,
										SrcPort:  port,
										DestPort: dst_port,
										Ip:       self.Servers[server].Ip,
									})
									logger.ProcessorLog.Warnf("Switch [%s] Port [%d] -> Server [%s] Port [%d]", device, port, server, dst_port)
								}
							}
						}
					}
				}

				ports = []*ocss_context.ConnectedTo{ocs.Ports[inPort], ocs.Ports[outPort]}
				for i, connTo := range ports {
					port := ports[1-i].Port
					device := ports[1-i].Device
					servers := ports[i].Server
					// TODO: What if OCS connected to another OCS?
					if self.DeviceType[device] == ocss_context.SWITCH {
						tor := self.UserView.FindToRByDeviceAndPort(device, port)
						if tor == nil {
							logger.ProcessorLog.Errorf("Switch [%s] Port [%d] is not a ToR", device, port)
							continue
						}

						for dst_port, _ := range tor.Ports {
							if dst_port == port {
								continue
							}

							if ocs.HavePort(dst_port) {
								logger.ProcessorLog.Debugf("Switch [%s] Port [%d] and Port [%d] both connect to OCS [%s], skip ", connTo.Device, dst_port, port, ocs.Device)
								continue
							}

							self.Switches[device].Ports[port].Server = servers
							connTo.Server = servers
							for server, ok := range servers {
								if ok {
									self.ForwardingTables[device] = append(self.ForwardingTables[device], &ocss_context.Forward{
										Device:   device,
										SrcPort:  dst_port,
										DestPort: port,
										Ip:       self.Servers[server].Ip,
									})
									logger.ProcessorLog.Errorf("Switch [%s] Port [%d] -> Server [%s] Port [%d]", device, dst_port, server, port)
								}
							}
						}
					}
				}
			}
		}
	}

	ocss_context.Print()
	return nil
}

func (p *Processor) UpdateForwardingTables() {
	self := ocss_context.GetSelf()

	for _, ocs := range self.UserView.OCSs {
		for _, connTo := range ocs.Ports {
			if self.DeviceType[connTo.Device] != ocss_context.SERVER {

				// Note that the port that used this port will also be cleared since they share the same server pointer
				connTo.Server = make(map[string]bool)
			}
		}
	}

	ocss_context.Print()
}

func (p *Processor) SetupController() {
	self := ocss_context.GetSelf()
	forwarder := p.Forwarder()

	for _, sw := range self.Switches {
		hostsPorts := make([]int, 0)
		switchPorts := make([]int, 0)

		for port, connTo := range sw.Ports {
			if self.DeviceType[connTo.Device] == ocss_context.SERVER {
				hostsPorts = append(hostsPorts, port)
			} else {
				switchPorts = append(switchPorts, port)
			}
		}

		logger.CfgLog.Warnf("Switch Info: %d, %v, %v", sw.Id, hostsPorts, switchPorts)
		forwarder.SetSwitchInfo(sw.Id, hostsPorts, switchPorts)
	}

	ocs_in_port := make([]int, 0)
	ocs_out_port := make([]int, 0)

	for _, ocs := range self.OCSs {

		// ip := self.OCSs[ocs.Device].Ip
		ocs_out_port = append(ocs_out_port, ocs.Conn.Out_port...)
		ocs_in_port = append(ocs_in_port, ocs.Conn.In_port...)

		forwarder.SetOCS(ocs.Ip, ocs_in_port, ocs_out_port)
		logger.CfgLog.Warnf("OCS Info: %s, %v, %v", ocs.Ip, ocs_in_port, ocs_out_port)
	}

	for device, ft := range self.ForwardingTables {

		in_port := make([]int, 0)
		out_port := make([]int, 0)

		in_port_ip := make([]int, 0)
		out_port_ip := make([]int, 0)
		ips := make([]string, 0)

		for _, rule := range ft {
			if rule.Ip != "" {
				in_port_ip = append(in_port_ip, rule.SrcPort)
				out_port_ip = append(out_port_ip, rule.DestPort)
				ips = append(ips, rule.Ip)
			} else {
				in_port = append(in_port, rule.SrcPort)
				out_port = append(out_port, rule.DestPort)
			}
		}

		forwarder.SetForwardingTable(self.Switches[device].Id, in_port, out_port)
		forwarder.SetForwardingTableWithIp(self.Switches[device].Id, in_port_ip, out_port_ip, ips)

		logger.ProcessorLog.Warnf("Forwarding Table: %d, %v, %v", self.Switches[device].Id, in_port, out_port)
		logger.ProcessorLog.Warnf("Forwarding Table with IP: %d, %v, %v, %v", self.Switches[device].Id, in_port_ip, out_port_ip, ips)
	}
}
