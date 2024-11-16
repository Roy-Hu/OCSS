import json
from ryu.app.wsgi import ControllerBase, Response, route
from ryu.lib import dpid as dpid_lib

from log import LOG

rdc_instance_name = 'rdc_app'

class RDCController(ControllerBase):

    def __init__(self, req, link, data, **config):
        super(RDCController, self).__init__(req, link, data, **config)
        self.rdc_app = data[rdc_instance_name]

    # URL patterns
    url_set_switch_info = '/rdc/set_switch_info/{dpid}'
    url_set_ocs = '/rdc/set_ocs/{ip}'
    create_url_forwarding_table = '/rdc/createforwardingtable/{dpid}'
    update_url_forwarding_table = '/rdc/updateforwardingtable/{dpid}'
    delete_url_forwarding_table = '/rdc/deleteforwardingtable/{dpid}'
    
    @route('rdc', url_set_ocs, methods=['POST'])
    def set_ocs(self, req, **kwargs):
        LOG.debug("Set OCS")
        ip = kwargs['ip']

        try:
            body = req.body.decode('utf-8') if req.body else ''
            data = json.loads(body) if body else {}
            LOG.debug("Request JSON payload: %s", data)
        except json.JSONDecodeError as e:
            LOG.error("JSON decode error: %s", e)
            return Response(status=400, body='Invalid JSON payload')

        # Extract and validate 'ocs_in_port' and 'ocs_out_port'
        ocs_in_port = data.get('ocs_in_port')
        ocs_out_port = data.get('ocs_out_port')
        
        LOG.debug("REST OCS in port: %s", ocs_in_port)
        LOG.debug("REST OCS out port: %s", ocs_out_port)
        if ocs_in_port is None or ocs_out_port is None:
            LOG.error("Missing 'ocs_in_port' or 'ocs_out_port' in payload")
            return Response(status=400, body="Missing 'ocs_in_port' or 'ocs_out_port' in payload")

        if not isinstance(ocs_in_port, list) or not isinstance(ocs_out_port, list):
            LOG.error("'ocs_in_port' and 'ocs_out_port' must be lists")
            return Response(status=400, body="'ocs_in_port' and 'ocs_out_port' must be lists")

        if not all(isinstance(port, int) for port in ocs_in_port):
            LOG.error("'ocs_in_port' must be a list of integers")
            return Response(status=400, body="'ocs_in_port' must be a list of integers")

        if not all(isinstance(port, int) for port in ocs_out_port):
            LOG.error("'ocs_out_port' must be a list of integers")
            return Response(status=400, body="'ocs_out_port' must be a list of integers")

        LOG.debug("Successfully updated switch info for IP: %s", ip)

        try:
            self.rdc_app.run_OCS_create_initial_connections(ocs_in_port + ocs_out_port, ocs_out_port + ocs_in_port)
        except Exception as e:
            LOG.exception("Error updating switch info for IP %s: %s", ip, e)
            return Response(status=500, body='Internal Server Error while setting switch info')

        return Response(status=200, body='Switch info set successfully')
    @route('rdc', url_set_switch_info, methods=['POST'], requirements={'dpid': dpid_lib.DPID_PATTERN})
    def set_switch_info(self, req, **kwargs):
        """
        Handle POST requests to update switch information.
        Expected JSON format:
        {
            "hostPorts": [1],
            "switchPorts": [9, 111, 69]
        }
        """
        LOG.debug("Set Switch Info")
        dpid_str = kwargs['dpid']
        dpid = dpid_lib.str_to_dpid(dpid_str)  # Convert string to integer dpid

        try:
            # Parse JSON payload
            new_vars = req.json if req.body else {}
        except ValueError:
            LOG.exception("Failed to parse JSON in set_switch_info")
            return Response(status=400, body='Invalid JSON')

        # Extract fields
        host_ports = new_vars.get('hostPorts')
        switch_ports = new_vars.get('switchPorts')

        # Validate presence of all required fields
        if host_ports is None or switch_ports is None:
            LOG.error("Missing parameters: hostPorts=%s, switchPorts=%s", host_ports, switch_ports)
            return Response(status=400, body='Missing parameters')

        # Validate that both fields are lists
        if not (isinstance(host_ports, list) and isinstance(switch_ports, list)):
            LOG.error("Parameters 'hostPorts' and 'switchPorts' must be lists")
            return Response(status=400, body="'hostPorts' and 'switchPorts' must be lists")

        # Validate that all elements in the lists are integers
        if not all(isinstance(p, int) for p in host_ports):
            LOG.error("'hostPorts' must be a list of integers, got %s", host_ports)
            return Response(status=400, body="'hostPorts' must be a list of integers")
        if not all(isinstance(p, int) for p in switch_ports):
            LOG.error("'switchPorts' must be a list of integers, got %s", switch_ports)
            return Response(status=400, body="'switchPorts' must be a list of integers")

        
        try:
            self.rdc_app.init_switch(dpid, host_ports, switch_ports)
        except Exception as e:
            LOG.exception("Exception while setting switch info for dpid %d: %s", dpid, e)
            return Response(status=500, body='Internal Server Error while setting switch info')


        LOG.debug("Switch ID: %d", dpid)
        LOG.debug("Switch Host Ports: %s", host_ports)
        LOG.debug("Switch Switch Ports: %s", switch_ports)
        try:
            # Reinstall flows with updated variables
            # self.rdc_app.update_flows()
            pass
        except Exception as e:
            LOG.exception("Error while updating flows: %s", e)
            return Response(status=500, body='Internal Server Error while updating flows')

        return Response(status=200)

    @route('rdc', create_url_forwarding_table, methods=['POST'], requirements={'dpid': dpid_lib.DPID_PATTERN})
    def create_forwarding_table(self, req, **kwargs):
        LOG.info("Create Forwarding Table")

        return self.set_forwarding_table(self.rdc_app.CREATE, req, **kwargs)
    
    @route('rdc', update_url_forwarding_table, methods=['PUT'], requirements={'dpid': dpid_lib.DPID_PATTERN})
    def update_forwarding_table(self, req, **kwargs):
        LOG.info("Update Forwarding Table")
        return self.set_forwarding_table(self.rdc_app.UPDATE, req, **kwargs)

    @route('rdc', delete_url_forwarding_table, methods=['POST'], requirements={'dpid': dpid_lib.DPID_PATTERN})
    def delete_forwarding_table(self, req, **kwargs):
        LOG.info("Update Forwarding Table")
        return self.set_forwarding_table(self.rdc_app.DELETE, req, **kwargs)
    
    @route('rdc', '/rdc/traffic_matrix/{dpid}', methods=['GET'])
    def get_traffic_matrix(self, req, **kwargs):
        dpid_str = kwargs['dpid']
        dpid = int(dpid_str)

        if dpid not in self.rdc_app.traffic_matrix:
            return Response(status=404, body='Traffic matrix for dpid {} not found'.format(dpid))

        # Convert the traffic matrix to JSON
        src_dict = self.rdc_app.traffic_matrix[dpid]
        traffic_matrix_dict = {}
        for (src_ip, dst_ip), byte_count in src_dict.items():
            if src_ip not in traffic_matrix_dict:
                traffic_matrix_dict[src_ip] = {}
            traffic_matrix_dict[src_ip][dst_ip] = byte_count

        # Return JSON response
        body = json.dumps({dpid_str: traffic_matrix_dict})
        return Response(content_type='application/json', body=body)

    def set_forwarding_table(self, cmd, req, **kwargs):
        dpid_str = kwargs['dpid']
        dpid = dpid_lib.str_to_dpid(dpid_str)
        try:
            new_vars = req.json if req.body else {}
        except ValueError:
            return Response(status=400, body='Invalid JSON')

        entries = new_vars.get('entries')  # Expected to be a list of {'in_port': int, 'out_port': int, 'ip': str}

        if entries is None:
            return Response(status=400, body='Missing parameters')

        if not isinstance(entries, list):
            return Response(status=400, body='Entries must be a list')

        forwardingTable = {}

        for entry in entries:
            in_port = entry.get('in_port')
            out_port = entry.get('out_port')
            src_ip = entry.get('src_ip')
            dst_ip = entry.get('dst_ip')
            if in_port is None or out_port is None or src_ip is None or dst_ip is None:
                continue  # Skip invalid entries
            forwardingTable[(in_port, src_ip, dst_ip)] = out_port

        dp = self.rdc_app.dataPaths.get(dpid)
        if dp:
            try:
                self.rdc_app.build_packets(dp, dpid, forwardingTable, cmd)
                LOG.info("Successfully built packets for dpid %s", dpid_str)
            except Exception as e:
                LOG.exception("Error building packets for dpid %s: %s", dpid_str, e)
                return Response(status=500, body='Internal Server Error while building packets')
        else:
            LOG.error("Datapath %s not found", dpid_str)
            return Response(status=404, body='Datapath not found')
        
        LOG.debug("Forwarding Table with IP: %s", forwardingTable)

        return Response(status=200)
    

