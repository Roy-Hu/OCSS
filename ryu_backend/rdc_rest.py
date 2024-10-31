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
    url_variables = '/rdc/variables'
    url_set_switch_info = '/rdc/set_switch_info/{dpid}'
    url_set_ocs = '/rdc/set_ocs/{ip}'
    url_forwarding_table = '/rdc/forwardingtable/{dpid}'
    url_forwarding_table_with_ip = '/rdc/forwardingtablewithip/{dpid}'

    @route('rdc', url_variables, methods=['GET'])
    def get_variables(self, req, **kwargs):
        """
        Handle GET requests to retrieve current variables.
        """
        variables = {
            'switch_ids': list(self.rdc_app.switch_ids),
            'ocs_in_port': self.rdc_app.ocs_in_port,
            'ocs_out_port': self.rdc_app.ocs_out_port,
            'fowardingTable': self.rdc_app.fowardingTable,
            'forwardingTableWithIp': self.rdc_app.forwardingTableWithIp,
            'hostPorts': self.rdc_app.hostPorts,
            'switchPorts': self.rdc_app.switchPorts,
        }
        body = json.dumps(variables)
        return Response(content_type='application/json', body=body)

    @route('rdc', url_variables, methods=['POST'])
    def set_variables(self, req, **kwargs):
        """
        Handle POST requests to update variables.
        """
        try:
            new_vars = req.json if req.body else {}
        except ValueError:
            return Response(status=400, body='Invalid JSON')

        if 'switch_ids' in new_vars:
            self.rdc_app.switch_ids = set(new_vars['switch_ids'])
        if 'ocs_in_port' in new_vars:
            self.rdc_app.ocs_in_port = new_vars['ocs_in_port']
        if 'ocs_out_port' in new_vars:
            self.rdc_app.ocs_out_port = new_vars['ocs_out_port']
        if 'fowardingTable' in new_vars:
            self.rdc_app.fowardingTable = new_vars['fowardingTable']
        if 'forwardingTableWithIp' in new_vars:
            self.rdc_app.forwardingTableWithIp = new_vars['forwardingTableWithIp']
        if 'hostPorts' in new_vars:
            self.rdc_app.hostPorts = new_vars['hostPorts']
        if 'switchPorts' in new_vars:
            self.rdc_app.switchPorts = new_vars['switchPorts']

        # # Reinstall flows with updated variables
        # self.rdc_app.update_flows()
        return Response(status=200)

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
            self.rdc_app.ocs_in_port.extend(ocs_in_port + ocs_out_port)
            self.rdc_app.ocs_out_port.extend(ocs_out_port + ocs_in_port)
            self.rdc_app.OCS_create_initial_connections()
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
            # Update internal state
            self.rdc_app.switch_ids.add(dpid)
            self.rdc_app.hostPorts[dpid] = host_ports
            self.rdc_app.switchPorts[dpid] = switch_ports

            dp = self.rdc_app.dataPaths.get(dpid)
            self.rdc_app.init_switch(dp)
        except Exception as e:
            LOG.exception("Exception while setting switch info for dpid %d: %s", dpid, e)
            return Response(status=500, body='Internal Server Error while setting switch info')


        LOG.debug("Switch ID: %d", dpid)
        LOG.debug("Switch Host Ports: %s", self.rdc_app.hostPorts[dpid])
        LOG.debug("Switch Switch Ports: %s", self.rdc_app.switchPorts[dpid])
        try:
            # Reinstall flows with updated variables
            # self.rdc_app.update_flows()
            pass
        except Exception as e:
            LOG.exception("Error while updating flows: %s", e)
            return Response(status=500, body='Internal Server Error while updating flows')

        return Response(status=200)


    @route('rdc', url_forwarding_table, methods=['POST'], requirements={'dpid': dpid_lib.DPID_PATTERN})
    def set_forwarding_table(self, req, **kwargs):
        """
        Handle POST requests to update forwarding table.
        """
        dpid_str = kwargs['dpid']
        dpid = dpid_lib.str_to_dpid(dpid_str)
        try:
            new_vars = req.json if req.body else {}
        except ValueError:
            return Response(status=400, body='Invalid JSON')

        entries = new_vars.get('entries')  # Expected to be a list of [in_port, out_port]

        if entries is None:
            return Response(status=400, body='Missing parameters')

        if not isinstance(entries, list):
            return Response(status=400, body='Entries must be a list')

        self.rdc_app.fowardingTable[dpid] = entries
        LOG.debug("REST Forwarding Table: %s", self.rdc_app.fowardingTable)
        dp = self.rdc_app.dataPaths.get(dpid)
        if dp:
            try:
                self.rdc_app.build_packets(dp, dpid)
                LOG.debug("Successfully built packets for dpid %s", dpid_str)
            except Exception as e:
                LOG.exception("Error building packets for dpid %s: %s", dpid_str, e)
                return Response(status=500, body='Internal Server Error while building packets')
        else:
            LOG.error("Datapath %s not found", dpid_str)
            return Response(status=404, body='Datapath not found')
        
        LOG.debug("REST Forwarding Table: %s", self.rdc_app.fowardingTable[dpid])

        return Response(status=200)

    @route('rdc', url_forwarding_table_with_ip, methods=['POST'], requirements={'dpid': dpid_lib.DPID_PATTERN})
    def set_forwarding_table_with_ip(self, req, **kwargs):
        """
        Handle POST requests to update forwarding table with IP addresses.
        """
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

        self.rdc_app.forwardingTableWithIp[dpid] = {}

        for entry in entries:
            in_port = entry.get('in_port')
            out_port = entry.get('out_port')
            ip = entry.get('ip')
            if in_port is None or out_port is None or ip is None:
                continue  # Skip invalid entries
            self.rdc_app.forwardingTableWithIp[dpid][(in_port, out_port)] = ip

        dp = self.rdc_app.dataPaths.get(dpid)
        if dp:
            try:
                self.rdc_app.build_packets(dp, dpid)
                LOG.debug("Successfully built packets for dpid %s", dpid_str)
            except Exception as e:
                LOG.exception("Error building packets for dpid %s: %s", dpid_str, e)
                return Response(status=500, body='Internal Server Error while building packets')
        else:
            LOG.error("Datapath %s not found", dpid_str)
            return Response(status=404, body='Datapath not found')
        
        LOG.debug("Forwarding Table with IP: %s", self.rdc_app.forwardingTableWithIp[dpid])

        return Response(status=200)
