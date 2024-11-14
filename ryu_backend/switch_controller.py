from ryu.base import app_manager
from ryu.controller import dpset
from ryu.controller import ofp_event
from ryu.controller.handler import MAIN_DISPATCHER
from ryu.controller.handler import set_ev_cls
from ryu.ofproto import ofproto_v1_3
from ryu.lib.packet import packet
from ryu.lib.packet import ethernet
from ryu.app.wsgi import WSGIApplication
from datetime import datetime
import copy
from ofdpa.config_parser import ConfigParser
from ofdpa.mods import Mods
from ryu.lib import hub

from ryu_backend.router import RDCController
from log import LOG

rdc_instance_name = 'rdc_app'

class RDC(app_manager.RyuApp):
    OFP_VERSIONS = [ofproto_v1_3.OFP_VERSION]
    _CONTEXTS = {'wsgi': WSGIApplication}

    def __init__(self, *args, **kwargs):
        super(RDC, self).__init__(*args, **kwargs)
        LOG.info("RDC Init")

        self.CREATE = "create"
        self.UPDATE = "update"
        DEFAULT_VLAN = 10

        self.dataPaths = {}
        self.connectionObj = None
        
        wsgi = kwargs['wsgi']
        wsgi.register(RDCController, {rdc_instance_name: self})
        
        self.dataPaths = {}
        config_dir = 'config/te_test/rdc'
        template_group_l2_interface_filename = "%s/%s.json" % (config_dir, "template_group_l2_interface")
        self.groupConfig = ConfigParser.get_config(template_group_l2_interface_filename)

        l2InterfacePopVlan = "%s/%s.json" % (config_dir, "group_l2_interface_popVlan")
        self.groupConfigPopVlan = ConfigParser.get_config(l2InterfacePopVlan)
        
        template_acl_unicast_vlan_inPort = "%s/%s.json" % (config_dir, "te_acl_unicast_vlan_inPort")
        self.configVlanInPort = ConfigParser.get_config(template_acl_unicast_vlan_inPort)
        
        template_acl_unicast_vlan_inPort_desIP = "%s/%s.json" % (config_dir, "te_acl_unicast_vlan_inPort_desIP")
        self.configVlanInPortDesIP = ConfigParser.get_config(template_acl_unicast_vlan_inPort_desIP)
  
        template_vlan_tagged = "%s/%s.json" % (config_dir, "template_vlan_tagged")
        self.configVlanTagged = ConfigParser.get_config(template_vlan_tagged)

        template_vlan_untagged = "%s/%s.json" % (config_dir, "template_vlan_untagged")
        self.configVlanUnTagged = ConfigParser.get_config(template_vlan_untagged)
    def create_group_l2_interface(self, template, dp, vlan, outputPort):
        LOG.info("Create Group L2 Interface for dpid %d port %d", dp.id, outputPort)
        group_l2_interface = copy.deepcopy(template)
        group_l2_interface['group_mod']['_name'] += "%03x%04x" % (vlan, outputPort)
        group_l2_interface['group_mod']['group_id'] += "%03x%04x" % (vlan, outputPort)
        group_l2_interface['group_mod']['buckets'][0]['actions'][0]['output']['port'] += str(outputPort)
        self.install_group_mod(dp, group_l2_interface)
        
        return

    @staticmethod
    def install_group_mod(dp, config):
        for type_ in ConfigParser.get_config_type(config):
            if type_ == "group_mod":
                mod_config = ConfigParser.get_group_mod(config)
                mod = Mods.create_group_mod(dp, mod_config)
                dp.send_msg(mod)
                return mod
            else:
                raise Exception("Wrong type", type_)
        return None

    @staticmethod
    def install_flow_mod(dp, config):
        LOG.debug("Install Flow Mod")
        for type_ in ConfigParser.get_config_type(config):
            if type_ == "flow_mod":
                mod_config = ConfigParser.get_flow_mod(config)
                mod = Mods.create_flow_mod(dp, mod_config)
                dp.send_msg(mod)
                return mod
            else:
                raise Exception("Wrong type", type_)
        return None

    def create_acl_unicast_vlan_inPort(self, dp, vlan, inPort, outputPort, priority=3):
        LOG.info("Create ACL Unicast for port %d <-> port %d", inPort, outputPort)
        
        acl_unicast = copy.deepcopy(self.configVlanInPort)
        acl_unicast['flow_mod']['_name'] += str(vlan) + '_' + str(inPort) + '_' + str(outputPort)
        acl_unicast['flow_mod']['priority'] += str(priority)
        acl_unicast['flow_mod']['cmd'] = 'add'
        acl_unicast['flow_mod']['match']['vlan_vid'] += str(vlan)
        acl_unicast['flow_mod']['match']["in_port"] += str(inPort)
        acl_unicast['flow_mod']['instructions'][0]['write'][0]['actions'][0]['set_queue']['queue_id'] += str(1)
        acl_unicast['flow_mod']['instructions'][0]['write'][0]['actions'][1]['group']['group_id'] += "%03x%04x" % (
        vlan, outputPort)
        
        self.install_flow_mod(dp, acl_unicast)
        return

    def set_acl_unicast_vlan_inPort_dstIp(self, dp, vlan, inPort, ip, outputPort, cmd, priority=3):
        acl_unicast = copy.deepcopy(self.configVlanInPortDesIP)
        queue = 1
        acl_unicast['flow_mod']['_name'] += str(vlan) + '_' + str(inPort) + '_' + ip + '_' + str(outputPort)
        acl_unicast['flow_mod']['priority'] += str(priority)
        acl_unicast['flow_mod']['cmd'] = cmd
        acl_unicast['flow_mod']['match']['vlan_vid'] += str(vlan)
        acl_unicast['flow_mod']['match']["in_port"] += str(inPort)
        acl_unicast['flow_mod']['match']['ipv4_dst'] += ip
        acl_unicast['flow_mod']['instructions'][0]['write'][0]['actions'][0]['set_queue']['queue_id'] += str(queue)
        acl_unicast['flow_mod']['instructions'][0]['write'][0]['actions'][1]['group']['group_id'] += "%03x%04x" % (vlan, outputPort)

        self.install_flow_mod(dp, acl_unicast)
        return acl_unicast
    
    def create_acl_unicast_vlan_inPort_dstIp(self, dp, vlan, inPort, ip, outputPort, priority=3):
        LOG.info("Create ACL Unicast port %d -> %d, IP %s", inPort, outputPort, ip)
        return self.set_acl_unicast_vlan_inPort_dstIp(dp, vlan, inPort, ip, outputPort, 'add', priority)

    def update_acl_unicast_vlan_inPort_dstIp(self, dp, vlan, inPort, ip, outputPort, priority=3):
        LOG.info("Update ACL Unicast port %d -> %d, IP %s", inPort, outputPort, ip)
        return self.set_acl_unicast_vlan_inPort_dstIp(dp, vlan, inPort, ip, outputPort, 'mod', priority)
    
    def createGroupInterfaces(self, dp, hostPorts, switchPorts, vlan=10):
        LOG.info("Create Group Interface for ports")
        for port in hostPorts:
            self.create_group_l2_interface(self.groupConfigPopVlan, dp, vlan, port)
        for port in switchPorts:
            self.create_group_l2_interface(self.groupConfig, dp, vlan, port)

    def create_vlan(self, dp, vlan, inPort):
        LOG.debug("Create Vlan %d for port %s", vlan, str(inPort))
        vlan_tagged = copy.deepcopy(self.configVlanTagged)
        vlan_tagged['flow_mod']['_name'] += str(vlan) + "_" + str(inPort)
        vlan_tagged['flow_mod']['match']['in_port'] += str(inPort)
        vlan_tagged['flow_mod']['match']['vlan_vid'] += str(vlan)
        self.install_flow_mod(dp, vlan_tagged)

        vlan_untagged = copy.deepcopy(self.configVlanUnTagged)
        vlan_untagged['flow_mod']['_name'] += str(vlan) + "_" + str(inPort)
        vlan_untagged['flow_mod']['match']['in_port'] += str(inPort)
        vlan_untagged['flow_mod']['instructions'][0]['apply'][0]['actions'][0]['set_field']['vlan_vid'] += str(vlan)
        self.install_flow_mod(dp, vlan_untagged)
        
        return

    def tagVlan(self, dp, ports, vlan=10):
        LOG.info("Tag Vlan")
        # TODO
        for inPort in ports:
            self.create_vlan(dp, vlan, inPort)

    def init_switch(self, dpid, hostPorts, switchPorts, vlan = 10):
        dp = self.dataPaths[dpid]
        self.createGroupInterfaces(dp, hostPorts, switchPorts, vlan)
        self.tagVlan(dp, hostPorts + switchPorts, vlan)

    def build_packets(self, dp, dpid, forwardingTable, cmd, vlan = 10):
        LOG.info("Build Packets for Swiich %d", dpid)
        
        for (inPort, dstIp), outPort in forwardingTable.items():
            if cmd == self.CREATE:
                if dstIp == "":
                    self.create_acl_unicast_vlan_inPort(dp, vlan, inPort, outPort)
                    self.create_acl_unicast_vlan_inPort(dp, vlan, outPort, inPort)
                else:
                    self.create_acl_unicast_vlan_inPort_dstIp(dp, vlan, inPort, dstIp, outPort)
            elif cmd == self.UPDATE:
                self.update_acl_unicast_vlan_inPort_dstIp(dp, vlan, inPort, dstIp, outPort)
            else:
                raise Exception("Invalid Command")     
        
    @set_ev_cls(dpset.EventDP, dpset.DPSET_EV_DISPATCHER)
    def handler_datapath(self, ev):
        LOG.info("Datapath Event Received %d", ev.dp.id)
        self.dataPaths[ev.dp.id] = ev.dp
        # if ev.enter:
        #     self.build_packets(ev.dp, ev.dp.id)

    @set_ev_cls(ofp_event.EventOFPPacketIn, MAIN_DISPATCHER)
    def packet_in_handler(self, ev):
        LOG.info("Event Datapath Id: %i", ev.msg.datapath.id)

        msg = ev.msg
        datapath = msg.datapath
        in_port = msg.match['in_port']

        pkt = packet.Packet(msg.data)
        eth = pkt.get_protocols(ethernet.ethernet)[0]
        LOG.info("Packet in on DPID %s (port %s): %s", datapath.id, in_port, eth)

    def run_OCS_create_initial_connections(self, ocs_in_port, ocs_out_port):
        hub.spawn(self.OCS_create_initial_connections, ocs_in_port, ocs_out_port)
        
    def OCS_create_initial_connections(self, ocs_in_port, ocs_out_port):
        from ocs.connections import GxcConnections
        from ocs.optionsForShareBackup import Options
        LOG.info("OCS_create_initial_connections")
        # PrintConnections(LOG, self.ocs_in_port, self.ocs_out_port)
        
        LOG.info("In Port: %s", ocs_in_port)
        LOG.info("Out Port: %s", ocs_out_port)
        if self.connectionObj is None:
            self.connectionObj = GxcConnections(Options())
        
        self.connectionObj.ent_crs_fiber(ocs_in_port, ocs_out_port)
