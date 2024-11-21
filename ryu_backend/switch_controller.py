from pickle import TRUE
from ryu.base import app_manager
from ryu.controller import dpset
from ryu.controller import ofp_event
from ryu.controller.handler import MAIN_DISPATCHER
from ryu.controller.handler import set_ev_cls
from ryu.ofproto import ofproto_v1_3
from ryu.lib.packet import packet
from ryu.lib.packet import ethernet
from ryu.app.wsgi import WSGIApplication
import copy
from ofdpa.config_parser import ConfigParser
from ofdpa.mods import Mods
from ryu.lib import hub

from ryu_backend.router import RDCController
from log import LOG
from collections import defaultdict
import signal

rdc_instance_name = 'rdc_app'

class RDC(app_manager.RyuApp):
    OFP_VERSIONS = [ofproto_v1_3.OFP_VERSION]
    _CONTEXTS = {'wsgi': WSGIApplication}

    def __init__(self, *args, **kwargs):
        super(RDC, self).__init__(*args, **kwargs)
        LOG.info("RDC Init")

        self.CREATE = "create"
        self.UPDATE = "update"
        self.DELETE = "delete"
        DEFAULT_VLAN = 10

        self.dataPaths = {}
        self.connectionObj = None
        
        # TODO: Can be a report instead of traffic matrix
        self.traffic_matrix = {}  # Key: dpid, Value: defaultdict
        self.prev_stats = {}      # Key: dpid, Value: defaultdict
        self.monitor_threads = {} # Key: dpid, Value: hub.spawn thread
        self.switchs = set()
        
        self.tracked_cookie = 0x1000 
        self.untracked_cookie = 0x2000  
        self.tor_tracked_cookies = set()  # Set of integers
        
        wsgi = kwargs['wsgi']
        wsgi.register(RDCController, {rdc_instance_name: self})
        
        self.dataPaths = {}
        config_dir = '../config'
        template_group_l2_interface_filename = "%s/%s.json" % (config_dir, "template_group_l2_interface")
        self.groupConfig = ConfigParser.get_config(template_group_l2_interface_filename)

        l2InterfacePopVlan = "%s/%s.json" % (config_dir, "group_l2_interface_popVlan")
        self.groupConfigPopVlan = ConfigParser.get_config(l2InterfacePopVlan)
        
        template_acl_unicast_vlan_inPort = "%s/%s.json" % (config_dir, "te_acl_unicast_vlan_inPort")
        self.configVlanInPort = ConfigParser.get_config(template_acl_unicast_vlan_inPort)
        
        template_acl_unicast_vlan_inPort_srcIp_desIP = "%s/%s.json" % (config_dir, "te_acl_unicast_vlan_inPort_srcIp_desIP")
        self.configVlanInPortSrcIPDesIP = ConfigParser.get_config(template_acl_unicast_vlan_inPort_srcIp_desIP)
  
        template_vlan_tagged = "%s/%s.json" % (config_dir, "template_vlan_tagged")
        self.configVlanTagged = ConfigParser.get_config(template_vlan_tagged)

        template_vlan_untagged = "%s/%s.json" % (config_dir, "template_vlan_untagged")
        self.configVlanUnTagged = ConfigParser.get_config(template_vlan_untagged)
        
        hub.spawn(self._register_signal_handler)

    def _register_signal_handler(self):
        def shutdown_handler(signum, frame):
            LOG.info("Shutdown signal received. Cleaning up flows...")
            self.cleanup_flows()
            raise SystemExit()

        signal.signal(signal.SIGTERM, shutdown_handler)
        signal.signal(signal.SIGINT, shutdown_handler)
        while True:
            hub.sleep(1)
            
    def cleanup_flows(self):
        LOG.info("TODO: Deleting all flows")

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

    def set_acl_unicast_vlan_inPort(self, dp, vlan, inPort, outputPort, cmd, priority=3):
        LOG.info("Create ACL Unicast for port %d <-> port %d", inPort, outputPort)
        
        acl_unicast = copy.deepcopy(self.configVlanInPort)
        acl_unicast['flow_mod']['_name'] += str(vlan) + '_' + str(inPort) + '_' + str(outputPort)
        acl_unicast['flow_mod']['priority'] += str(priority)
        acl_unicast['flow_mod']['cmd'] = cmd
        acl_unicast['flow_mod']['match']['vlan_vid'] += str(vlan)
        acl_unicast['flow_mod']['match']["in_port"] += str(inPort)
        acl_unicast['flow_mod']['instructions'][0]['write'][0]['actions'][0]['set_queue']['queue_id'] += str(1)
        acl_unicast['flow_mod']['instructions'][0]['write'][0]['actions'][1]['group']['group_id'] += "%03x%04x" % (
        vlan, outputPort)
        acl_unicast['flow_mod']['cookie'] = self.untracked_cookie

        self.install_flow_mod(dp, acl_unicast)
        return
    def set_acl_unicast_vlan_inPort_srcIp_dstIp(self, dp, torid, vlan, inPort, srcIp, dstIp, outputPort, cmd, priority=3):
        acl_unicast = copy.deepcopy(self.configVlanInPortSrcIPDesIP)
        queue = 1
        acl_unicast['flow_mod']['_name'] += str(vlan) + '_' + str(inPort) + '_' + srcIp + dstIp + '_' + str(outputPort)
        acl_unicast['flow_mod']['priority'] += str(priority)
        acl_unicast['flow_mod']['cmd'] = cmd
        acl_unicast['flow_mod']['match']['vlan_vid'] += str(vlan)
        acl_unicast['flow_mod']['match']["in_port"] += str(inPort)
        acl_unicast['flow_mod']['match']['ipv4_src'] += srcIp
        acl_unicast['flow_mod']['match']['ipv4_dst'] += dstIp
        acl_unicast['flow_mod']['instructions'][0]['write'][0]['actions'][0]['set_queue']['queue_id'] += str(queue)
        acl_unicast['flow_mod']['instructions'][0]['write'][0]['actions'][1]['group']['group_id'] += "%03x%04x" % (vlan, outputPort)
        
        acl_unicast['flow_mod']['cookie'] = self.tracked_cookie + torid
        self.tor_tracked_cookies.add(self.tracked_cookie + torid)
        
        self.install_flow_mod(dp, acl_unicast)
        return acl_unicast
    def create_acl_unicast_vlan_inPort(self, dp, vlan, inPort, outputPort, priority=3):
        LOG.info("Create ACL Unicast port %d -> %d", inPort, outputPort)
        return self.set_acl_unicast_vlan_inPort(dp, vlan, inPort, outputPort, 'add', priority)
    
    def delete_acl_unicast_vlan_inPort(self, dp, vlan, inPort, outputPort, priority=3):
        LOG.info("Delete ACL Unicast port %d -> %d", inPort, outputPort)
        return self.set_acl_unicast_vlan_inPort(dp, vlan, inPort, outputPort, 'del', priority)
    
    def create_acl_unicast_vlan_inPort_srcIp_dstIp(self, dp, torid, vlan, inPort, srcIp, dstIp, outputPort, priority=3):
        LOG.info("Create ACL Unicast port %d -> %d, Src IP %s, Dst Ip %s", inPort, outputPort, srcIp, dstIp)
        return self.set_acl_unicast_vlan_inPort_srcIp_dstIp(dp, torid, vlan, inPort, srcIp, dstIp, outputPort, 'add', priority)

    def update_acl_unicast_vlan_inPort_srcIp_dstIp(self, dp, torid, vlan, inPort, srcIp, dstIp, outputPort, priority=3):
        LOG.info("Update ACL Unicast port %d -> %d, Src IP %s, Dst Ip %s", inPort, outputPort,  srcIp, dstIp)
        return self.set_acl_unicast_vlan_inPort_srcIp_dstIp(dp, torid, vlan, inPort, srcIp, dstIp, outputPort, 'mod', priority)
    
    def delete_acl_unicast_vlan_inPort_srcIp_dstIp(self, dp, torid, vlan, inPort, srcIp, dstIp, outputPort, priority=3):
        LOG.info("Delete ACL Unicast port %d -> %d, Src IP %s, Dst Ip %s", inPort, outputPort,  srcIp, dstIp)
        return self.set_acl_unicast_vlan_inPort_srcIp_dstIp(dp, torid, vlan, inPort, srcIp, dstIp, outputPort, 'del', priority)
    
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
        LOG.info("Initializing switch with dpid %d", dpid)
        
        while True:
            if dpid in self.dataPaths:
                break
            LOG.info("Waiting for datapath %d to connect, retry after 1 sec", dpid)
            hub.sleep(1)
        
        dp = self.dataPaths[dpid]
        self.createGroupInterfaces(dp, hostPorts, switchPorts, vlan)
        self.tagVlan(dp, hostPorts + switchPorts, vlan)
        
        self.prev_stats[dpid] = defaultdict(int)
        self.traffic_matrix[dpid] = defaultdict(int)
        
        self.switchs.add(dpid)
        if dpid not in self.monitor_threads:
            self.monitor_threads[dpid] = hub.spawn(self._monitor, dp)

    def _monitor(self, datapath):
        LOG.info("Starting monitoring thread for dpid %d", datapath.id)
        while True:
            self.request_flow_stats(datapath)
            hub.sleep(1) 

    def request_flow_stats(self, datapath):
        LOG.info("Request flow stats for dpid %d", datapath.id)
        parser = datapath.ofproto_parser

        req=parser.OFPFlowStatsRequest(datapath)
        
        datapath.send_msg(req)

    @set_ev_cls(ofp_event.EventOFPFlowStatsReply, MAIN_DISPATCHER)
    def flow_stats_reply_handler(self, ev):
        datapath = ev.msg.datapath
        dpid = datapath.id
        body = ev.msg.body

        if dpid not in self.prev_stats:
            self.prev_stats[dpid] = {}
        if dpid not in self.traffic_matrix:
            self.traffic_matrix[dpid] = {}

        for stat in body:
            if stat.cookie not in self.tor_tracked_cookies:
                continue

            torid = stat.cookie - self.tracked_cookie
            match = stat.match
            byte_count = stat.byte_count
            src_ip = match.get('ipv4_src')
            dst_ip = match.get('ipv4_dst')

            LOG.info(
                "Cookie Flow %d stats for dpid %d: %s -> %s: %d bytes",
                stat.cookie, dpid, src_ip, dst_ip, byte_count
            )

            if src_ip and dst_ip:
                key = (src_ip, dst_ip)
                if torid not in self.prev_stats[dpid]:
                    self.prev_stats[dpid][torid] = {}
                if torid not in self.traffic_matrix[dpid]:
                    self.traffic_matrix[dpid][torid] = {}
                
                prev_byte_count = self.prev_stats.get(key, 0)
                delta = byte_count - prev_byte_count
                if delta < 0:
                    delta = byte_count
                
                self.prev_stats[dpid][torid][key] = byte_count
                self.traffic_matrix[dpid][torid][key] = (
                    self.traffic_matrix[dpid][torid].get(key, 0) + delta
                )

    def build_packets(self, dpid, torid, forwardingTable, cmd, vlan = 10):
        LOG.info("Build Packets for Swiich %d", dpid)
        
        dp = self.dataPaths.get(dpid)
        if dp is None:
            LOG.info("Cannot find %d", dpid)
        
        if dpid not in self.switchs:
            LOG.info("Switch %d not initialized", dpid)
            return
        
        for (inPort, srcIp, dstIp), outPort in forwardingTable.items():
            if cmd == self.CREATE:
                if torid == 0:
                    self.create_acl_unicast_vlan_inPort(dp, vlan, inPort, outPort)
                    self.create_acl_unicast_vlan_inPort(dp, vlan, outPort, inPort)
                else:
                    self.create_acl_unicast_vlan_inPort_srcIp_dstIp(dp, torid, vlan, inPort, srcIp, dstIp, outPort)
            elif cmd == self.UPDATE:
                self.update_acl_unicast_vlan_inPort_srcIp_dstIp(dp, torid, vlan, inPort, srcIp, dstIp, outPort)
            elif cmd == self.DELETE:
                if torid == 0:
                    self.delete_acl_unicast_vlan_inPort(dp, torid, vlan, inPort, outPort)
                    self.delete_acl_unicast_vlan_inPort(dp, torid, vlan, outPort, inPort)
                else:    
                    self.delete_acl_unicast_vlan_inPort_srcIp_dstIp(dp, torid, vlan, inPort, srcIp, dstIp, outPort)
            else:
                raise Exception("Invalid Command")     
        
    @set_ev_cls(dpset.EventDP, dpset.DPSET_EV_DISPATCHER)
    def handler_datapath(self, ev):
        dp = ev.dp
        dpid = dp.id
        if ev.enter:
            LOG.info("Datapath connected: %d", dpid)
            self.dataPaths[dpid] = dp
        else:
            LOG.info("Datapath disconnected: %d", dpid)
            if dpid in self.dataPaths:
                del self.dataPaths[dpid]
            if dpid in self.monitor_threads:
                hub.kill(self.monitor_threads[dpid])
                del self.monitor_threads[dpid]
            if dpid in self.traffic_matrix:
                del self.traffic_matrix[dpid]
            if dpid in self.prev_stats:
                del self.prev_stats[dpid]


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
