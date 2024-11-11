package ocss

import (
	"fmt"
	"os"

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

// ocs_in_port should setup server pointer before calling this function
// setup the sever pointer for the switch port that connects to the ocs port
func setOcsNxtSwitch(self *ocss_context.OCSSContext, ocs *ocss_context.OCS, ocs_in_port int) error {
	ocs_out_port := ocs.ConnectedPort(ocs_in_port)
	if self.UserView.OCSs[ocs.Name].PortServerConn[ocs_out_port].FixedServer {
		logger.ProcessorLog.Errorf("OCS [%s] Port [%d] and Port [%d] both connects to a fixed server, skip", ocs.Name, ocs_in_port, ocs_out_port)
		os.Exit(1)

		return nil
	}

	ocs.PortServerConn[ocs_out_port].CopyConnServerInfo(ocs.PortServerConn[ocs_in_port])

	if self.DeviceType[ocs.PortConnToMap[ocs_out_port].Device] == ocss_context.SWITCH {
		// the switch port that connects to ocs_out_port should also connect to the server
		// but it is not psycially connected to the ocs in port(it may change if the user change the connection in ocs)
		// so we need to set the server pointer
		device := ocs.PortConnToMap[ocs_out_port].Device
		port := ocs.PortConnToMap[ocs_out_port].Port

		tor := self.UserView.FindToRByDeviceAndPort(device, port)
		tor.PortServerConn[port].CopyConnServerInfo(ocs.PortServerConn[ocs_out_port])
	} else {
		logger.ProcessorLog.Errorf("Currently OCS port %d should connect to a switch instead of %s", ocs_out_port, ocs.PortConnToMap[ocs_out_port].Device)
		return fmt.Errorf("Currently OCS should connect to a switch")
	}

	return nil
}

func (p *Processor) CreateForwardingTables() error {
	self := ocss_context.GetSelf()
	ocs_finished_setup := make(map[string]bool)

	// Note: ports connect through physical link should share the same server pointer
	// for logical edge ocs that connects to the server
	for _, ocs := range self.UserView.OCSs {
		connPorts := ocs.GetConnPorts()
		for _, ocs_port := range connPorts {
			connTo := ocs.PortConnToMap[ocs_port]
			// ocs logically connects to the server
			// ocs_port is the port in ocs that connects to the server
			if self.DeviceType[connTo.Device] == ocss_context.SERVER {
				logger.ProcessorLog.Infof("OCS [%s] Port [%d] -> [%s] Port [%d]", ocs.Name, ocs_port, connTo.Device, connTo.Port)

				// TODO: Support two or more switch between ocs and server
				server := connTo.Device
				// what the server actually connects to
				device := self.Servers[server].PortConnToMap[connTo.Port].Device
				in_port := self.Servers[server].PortConnToMap[connTo.Port].Port

				// All others port that connects to this ocs port should also connect to the server
				switch self.DeviceType[device] {
				case ocss_context.OPTICAL_SWITCH:
					// TODO: Support optical switch
					logger.ProcessorLog.Errorf("Currently Server must connect to a switch")
				// server actually connects to a switch instead of the ocs
				// TODO: currently only support one switch between server and ocs
				case ocss_context.SWITCH:

					sw := self.Switches[device]
					for port, sw_connTo := range sw.PortConnToMap {
						logger.ProcessorLog.Debugf("Switch [%s] Port [%d] -> Server [%s] Port [%d]", device, port, sw_connTo.Device, sw_connTo.Port)

						// switch port connects to the current ocs
						if sw_connTo.Device == ocs.Device && sw_connTo.Port == ocs_port {
							logger.ProcessorLog.Infof("OCS [%s] Port [%d] -> Switch [%s] Port [%d]", ocs.Name, ocs_port, device, port)
							out_port := port

							// the in port connects to the server and the out port connects to the ocs beed to be treated as a psycial link

							sw.AddForwardingRule(in_port, out_port, "")

							// server - > sw in port -> sw out port -> ocs in port share the same sever pointer
							// since we logically connect the server to the ocs
							// the forwarding of in/out port of the sw be treated as psycial link
							ocs.PortServerConn[ocs_port] = &ocss_context.ConnServerInfo{
								Server:      make(map[string]bool),
								FixedServer: true,
							}
							ocs.PortServerConn[ocs_port].Server[sw.PortConnToMap[in_port].Device] = true

							if err := setOcsNxtSwitch(self, ocs, ocs_port); err != nil {
								logger.ProcessorLog.Errorf("Error setting OCS [%s] Port [%d] -> [%s] Port [%d]: ", ocs.Name, ocs_port, connTo.Device, connTo.Port, err)
								return err
							}

							continue
						}
					}
				}

				ocs_finished_setup[ocs.Name] = true
			}
		}
	}

	core_ocs := make(map[string]bool)
	for _, ocs := range self.UserView.OCSs {
		if !ocs_finished_setup[ocs.Name] {
			core_ocs[ocs.Name] = true
		}
	}

	updateCoreOCSAndTor(core_ocs)
	p.createController()

	return nil
}

func (p *Processor) UpdateForwardingTables() error {
	logger.ProcessorLog.Info("Update Forwarding Tables")
	self := ocss_context.GetSelf()

	for _, tor := range self.UserView.ToRs {
		for _, serverInfo := range tor.PortServerConn {
			if !serverInfo.FixedServer {
				serverInfo.Server = make(map[string]bool)
			}
		}
	}

	ocs_finished_setup := make(map[string]bool)

	for _, ocs := range self.UserView.OCSs {
		// if !ocs.Changed {
		// 	ocs_finished_setup[ocs.Name] = true
		// 	logger.ProcessorLog.Infof("OCS [%s] not changed, skip", ocs.Name)
		// 	continue
		// }

		ocs_finished_setup[ocs.Name] = false
		connPorts := ocs.GetConnPorts()
		for _, ocs_port := range connPorts {
			if ocs.PortServerConn[ocs_port].FixedServer {
				if err := setOcsNxtSwitch(self, ocs, ocs_port); err != nil {
					logger.ProcessorLog.Errorf("Error setting OCS [%s] Port [%d] <-> Port [%d]", ocs.Name, ocs_port, ocs.ConnectedPort(ocs_port))
					return err
				}

				// TODO: This may be more dedicated, maybe per port instead of per ocs
				ocs_finished_setup[ocs.Name] = true
			}
		}
	}

	core_ocs := make(map[string]bool)
	for _, ocs := range self.UserView.OCSs {
		if !ocs_finished_setup[ocs.Name] {
			core_ocs[ocs.Name] = true
		}
	}

	updateCoreOCSAndTor(core_ocs)

	p.updateController()

	return nil
}

func updateCoreOCSAndTor(core_ocs map[string]bool) {
	logger.ProcessorLog.Info("Update Core OCS and ToR")
	self := ocss_context.GetSelf()
	// for logical core ocs that connects to the switch
	for _, ocs := range self.UserView.OCSs {
		// ocs without logical connection to the server wull be the core
		setup := make(map[int]bool)
		if core_ocs[ocs.Name] {
			logger.ProcessorLog.Infof("OCS [%s] Update Forwarding Table", ocs.Name)

			for i := range len(ocs.Conn.In_port) {
				inPort := ocs.Conn.In_port[i]
				outPort := ocs.Conn.Out_port[i]
				if setup[inPort] || setup[outPort] {
					continue
				} else {
					setup[inPort] = true
					setup[outPort] = true
				}

				logger.ProcessorLog.Infof("OCS [%s] Connection Port [%d](Connect to %s %d) <-> Port [%d](Connect to %s %d)",
					ocs.Name, inPort, ocs.PortConnToMap[inPort].Name, ocs.PortConnToMap[inPort].Port, outPort, ocs.PortConnToMap[outPort].Name, ocs.PortConnToMap[outPort].Port)

				// setup foward rule from ocs -> tor -> server
				ports := []int{inPort, outPort}
				for _, port := range ports {
					connTo := ocs.PortConnToMap[port]
					device := connTo.Device
					// TODO: What if OCS connected to another OCS?
					// ocs in/out port connects to a switch
					if self.DeviceType[device] == ocss_context.SWITCH {
						sw_port := connTo.Port
						// Find the tor that connects to the ocs in/out port
						tor := self.UserView.FindToRByDeviceAndPort(device, sw_port)
						if tor == nil {
							logger.ProcessorLog.Errorf("Switch [%s] Port [%d] is not a ToR", device, sw_port)
							continue
						}

						for dst_port, dst_connTo := range tor.PortConnToMap {
							// skip the tor port that connects to the current ocs in/out port
							if dst_port == sw_port {
								continue
							}

							// May delete?
							// dst_port and sw_port connects to the same ocs, skip
							if ocs.HavePort(dst_connTo.Port) {
								logger.ProcessorLog.Debugf("%s Port [%d] and Port [%d] both connect to OCS [%s], skip ", tor.Name, connTo.Device, dst_port, sw_port, ocs.Name)
								continue
							}

							sw := self.Switches[device]
							// since the tor port connects to the ocs port, the tor port should also connect to the server
							tor.PortServerConn[sw_port].AddConnServerInfo(tor.PortServerConn[dst_port])

							// Add forwarding rule for the sw_port ->  dst_port, ip: dst_port.Servers.ip
							for server, ok := range tor.PortServerConn[dst_port].Server {
								if ok {
									sw.AddForwardingRule(sw_port, dst_port, self.Servers[server].Ip)
									logger.ProcessorLog.Infof("tor [%s] Port [%d] -> Port [%d], Server [%s]", tor.Name, sw_port, dst_port, server)
								}
							}
						}
					} else {
						logger.ProcessorLog.Errorf("Currently OCS port %d should connect to a switch instead of %s", port, device)
					}
				}

				// setup foward rule from server -> tor -> ocs
				// currently, we want to setup the forwarding rule from server -> dst tor -> ocs -> src tor
				for i, _ := range ports {
					srcConnTo := ocs.PortConnToMap[ports[i]]
					dstConnTo := ocs.PortConnToMap[ports[1-i]]
					dstPort := dstConnTo.Port
					dstDevice := dstConnTo.Device

					// find the tor that connects to the ocs src port
					srcTor := self.UserView.FindToRByDeviceAndPort(srcConnTo.Device, srcConnTo.Port)
					// find the servers that src tor can connect to
					srcServers := srcTor.PortServerConn[srcConnTo.Port].Server

					// TODO: What if OCS connected to another OCS?
					if self.DeviceType[dstDevice] == ocss_context.SWITCH {
						dstTor := self.UserView.FindToRByDeviceAndPort(dstDevice, dstPort)
						if dstTor == nil {
							logger.ProcessorLog.Errorf("Switch [%s] Port [%d] is not a ToR", dstDevice, dstPort)
							continue
						}

						for sw_port, sw_connTo := range dstTor.PortConnToMap {
							if sw_port == dstPort {
								continue
							}

							if ocs.HavePort(sw_connTo.Port) {
								logger.ProcessorLog.Debugf("Switch [%s] Port [%d] and Port [%d] both connect to OCS [%s], skip ", dstDevice, sw_port, dstPort, ocs.Name)
								continue
							}

							sw := self.Switches[dstDevice]

							for server, ok := range srcServers {
								if ok {
									sw.AddForwardingRule(sw_port, dstPort, self.Servers[server].Ip)
									logger.ProcessorLog.Infof("Switch [%s] Port [%d] -> Port [%d], Server [%s] ", dstDevice, sw_port, dstPort, server)
								}
							}
						}
					}
				}
			}
		}

		ocs.Changed = false
	}
}

func (p *Processor) setupSwitch() {
	self := ocss_context.GetSelf()
	forwarder := p.Forwarder()

	for _, sw := range self.Switches {
		hostsPorts := make([]int, 0)
		switchPorts := make([]int, 0)

		for port, connTo := range sw.PortConnToMap {
			if self.DeviceType[connTo.Device] == ocss_context.SERVER {
				hostsPorts = append(hostsPorts, port)
			} else {
				switchPorts = append(switchPorts, port)
			}
		}

		logger.CfgLog.Infof("Switch Info: %d, %v, %v", sw.Id, hostsPorts, switchPorts)
		forwarder.SetSwitchInfo(sw.Id, hostsPorts, switchPorts)
	}
}

func (p *Processor) setupOCS() {
	self := ocss_context.GetSelf()
	forwarder := p.Forwarder()

	ocs_in_port := make([]int, 0)
	ocs_out_port := make([]int, 0)

	for _, ocs := range self.UserView.OCSs {
		// ip := self.OCSs[ocs.Device].Ip
		ocs_out_port = append(ocs_out_port, ocs.Conn.Out_port...)
		ocs_in_port = append(ocs_in_port, ocs.Conn.In_port...)
	}

	forwarder.SetOCS("10.224.92.88", ocs_in_port, ocs_out_port)
	logger.CfgLog.Infof("OCS Info: %s, %v, %v", "10.224.92.88", ocs_in_port, ocs_out_port)
}

func (p *Processor) setupForwardingTable() {
	self := ocss_context.GetSelf()
	forwarder := p.Forwarder()

	for _, sw := range self.Switches {
		in_port := make([]int, 0)
		out_port := make([]int, 0)

		in_port_ip := make([]int, 0)
		out_port_ip := make([]int, 0)
		ips := make([]string, 0)

		for _, rule := range sw.ForwardingRule {
			if !rule.Init {
				continue
			}

			if rule.Ip != "" {
				in_port_ip = append(in_port_ip, rule.SrcPort)
				out_port_ip = append(out_port_ip, rule.DestPort)
				ips = append(ips, rule.Ip)
			} else {
				in_port = append(in_port, rule.SrcPort)
				out_port = append(out_port, rule.DestPort)
			}

			rule.Init = false
		}

		if len(in_port) != 0 && len(out_port) != 0 {
			logger.ProcessorLog.Infof("Forwarding Table: %d, %v, %v", sw.Id, in_port, out_port)

			forwarder.SetForwardingTable(sw.Id, in_port, out_port)
		}

		if len(in_port_ip) != 0 && len(out_port_ip) != 0 {
			logger.ProcessorLog.Infof("Forwarding Table with IP: %d, %v, %v, %v", sw.Id, in_port_ip, out_port_ip, ips)

			forwarder.SetForwardingTableWithIp(sw.Id, in_port_ip, out_port_ip, ips)
		}
	}

}

func (p *Processor) createController() {
	p.setupSwitch()
	p.setupOCS()
	p.setupForwardingTable()
}

func (p *Processor) updateController() {
	p.setupOCS()
	p.setupForwardingTable()
}
