import re

import sys
import time
from datetime import datetime

from tl1 import TL1


class GxcConnections(TL1):
    def __init__(self, options):
        TL1.__init__(self, options)
        self.connect()
        self.login()

    def logging(self, module):
        self.mylog.setModule(module, self.swserial)

    def debugMsg(self, msg):
        self.mylog.logMsg("DEBUG", msg)

    def outputMsg(self, msg):
        self.mylog.logMsg("OUTPUT", msg)

    def scriptMsg(self, msg):
        self.mylog.logMsg("SCRIPT", msg)

    def straggleMsg(self, msg):
        self.mylog.logMsg("STRAGGLE", msg)

    def errorMsg(self, msg):
        self.mylog.logMsg("DEBUG", msg)

    def closeLogs(self):
        self.mylog.closeLogs()

        # Prints to the Console

    def printMsg(self, msg):

        time = datetime.now().isoformat()
        print (time + "," + msg)

    #####################
    #####################
    # save_recall_ready
    # checks if the previous save recall operation is complete
    # after it is complete, fetch the powers
    #####################

    def save_recall_ready(self):
        # keeps polling for the save recall status and tries 20 times.
        # if does not find it as completed !!!...aborts..

        pending_re = re.compile('progress')
        success_re = re.compile('Completed')
        failed_re = re.compile('Failed')
        conn_failed_re = re.compile('connections failed')
        retry = 0

        while (retry < 25):
            # self.debuglog.flush();
            output = self.command('rtrv-saverecall')

            if pending_re.search(output) != None:  # pending match found
                self.debugMsg("\n *** In Progress ***\n")
                time.sleep(1)
                retry = retry + 1
                continue
            elif success_re.search(output) != None:  # completed match found
                self.debugMsg("\n === Completed === \n")
                return True
            elif failed_re.search(output) != None:  # failed
                self.errorMsg("\n #### Failed #### \n" + "Output:" + output + "\n")
                self.debugMsg("\n #### Failed #### \n" + "Output:" + output + "\n")
                self.debugMsg(output)
                return False
            elif conn_failed_re.search(output) != None:  # some connections failed
                self.errorMsg("\n ### CONNS Failed ###\n" + "Output:" + output + "\n")
                self.debugMsg("\n ### CONNS Failed ###\n" + "Output:" + output + "\n")
                self.debugMsg(output)
                return False

        self.errorMsg("Status is not complete...Aborting\n")
        self.debugMsg("Status is not complete...Aborting\n")
        return False

    def ent_crs_fiber(self, inports, outports, is_async=False):
        # convert the ports from human readable ports to machine readable ports
        self.debugMsg("Inport In: " + str(inports))
        self.debugMsg("Outport In: " + str(outports))
        
        inport_list = [str(i % 10000 + 10000) for i in inports]
        outport_list = [str(o % 20000 + 20000) for o in outports]

        # convert list of ports to message format
        inport_str = "&".join(inport_list)
        outport_str = "&".join(outport_list)

        self.debugMsg("Inport Str: " + inport_str)
        self.debugMsg("Outport Str: " + outport_str)
        
        # send the message
        command_str_fmt = 'ent-crs-fiber::{},{}:1%s;'.format(inport_str, outport_str)
        command_str = command_str_fmt % (":::opmode=async" if is_async else "")

        self.debugMsg("Command: " + command_str)

        return self.command(command_str)

    def ent_crs_fiber_all(self):
        command_str = 'ent-crs-fiber::{},{}:1;'.format("all", "all")
        return self.command(command_str)

    def dlt_crs_fiber(self, inports, outports, is_async=False):
        # convert the ports from human readable ports to machine readable ports
        inport_list = [str(i % 10000 + 10000) for i in inports]
        outport_list = [str(o % 20000 + 20000) for o in outports]

        # convert list of ports to message format
        inport_str = "&".join(inport_list)
        outport_str = "&".join(outport_list)

        # send the message
        command_str_fmt = 'dlt-crs-fiber::{},{}:1%s;'.format(inport_str, outport_str)
        command_str = command_str_fmt % (":::opmode=async" if is_async else "")

        return self.command(command_str)

    def dlt_crs_fiber_all(self, is_async=False):
        command_str_fmt = 'dlt-crs-fiber::all:1%s;'
        command_str = command_str_fmt % (":::opmode=async" if is_async else "")
        return self.command(command_str)

    def ed_param(self, val):
        command_str = 'ed-param:::1:::NAME=PowerMonitoringPeriod,VAL=%d;' % val
        return self.command(command_str)

    #####################
    # loopback
    # performs a loopback, deletes existing connections .
    #
    #####################
    def loopback(self):
        self.debugMsg("proceeding to do loopback")
        #        self.logging("loopback");

        self.command('dlt-crs-all')

        self.command('opr-crsset-lpbk')
        rc = self.save_recall_ready()

        if rc == False:  # save recall Fail
            print ("Loopback command Failed")
            self.errorMsg("Loopback failed")
            sys.exit(1)

    ################
    # bulkconnect
    #
    # performs a bulk connect using the connStr
    #
    ###############

    def bulkConnect(self, connStr):
        self.debugMsg("Proceeding to do bulkconnect")
        #      self.logging("bulkconnect")

        bulkcommand = "ent-crs-bulk::" + connStr
        self.command(bulkcommand)

        rc = self.save_recall_ready()

        if rc == False:
            print ("Bulk Connect Failed")
            self.errorMsg("Bulk connect failed")
            sys.exit(1)

    ###################
    # find_crs
    ###################

    def find_crs(self, source, dest, group_name=None, conn_name=None):
        crs_list = self.get_crs_list()
        # print"crs list  "
        # print crs_list
        for crs in crs_list:
            print("cross list is     ")
            print(crs)
            print(crs['SRCPORT'])
            print(str(source))
            print(crs['DSTPORT'])
            print(str(dest))
            # '''
            if crs['SRCPORT'] == str(source) and crs['DSTPORT'] == str(dest):

                if group_name != None:
                    if crs['GRPNAME'] != str(group_name):
                        continue

                if conn_name != None:
                    if crs['CONNNAME'] != str(conn_name):
                        continue

            elif crs['SRCPORT'] == str(source):
                # found the source in a connection
                return crs

            elif crs['DSTPORT'] == str(dest):
                return crs

                # return crs

    # generic get list implementation
    # for different commands which returns a list
    # rtrv-crs, rtrv-port-sum::0, rtrv-crs-sum;
    # a dictionary based list is returned.
    def get_list(self, listcommand):
        output = self.command(listcommand)
        self.scriptMsg(output)

        if self.completed_re.search(output) == None:
            raise (ValueError, 'Failed to get list.')

        out_list = {}
        # 'rtrv-crs' 'rtrv-port-sum'  output is spanned across multiple pages
        # split each page by 'COMPLD'
        for pages in output.split('COMPLD'):
            # strip off any blank lines etc.
            block = pages.strip()
            # within each page, split on newlines.
            items = block.split('\r\n')

            try:
                items.remove('')
            except:
                pass

            for line in items:
                line = line.strip()

                if line.startswith('"'):
                    # strip off the last '"'
                    line = line.rstrip('"')
                    title, fields = line.split(':', 1)
                    parts = fields.split(',')

                    try:
                        parts.remove('')
                    except:
                        pass

                    fields_dictionary = {}
                    connid = (title.strip()).strip('\"')
                    fields_dictionary['title'] = (title.strip()).strip('\"')

                    for part in parts:
                        if not part.strip():
                            continue
                        name, value = part.split('=', 1)
                        fields_dictionary[name] = value

                    try:
                        src = fields_dictionary['SRCPORT']
                    except KeyError:
                        src = fields_dictionary['title']
                        pass
                        # we save the data in associative array format.
                        # {'PS': 'UPR', 'GRPNAME': 'SYSTEM', 'CONNNAME': 'LOOPBACK_001', 'OC': 'OK', 'title': '1>1'
                        #  , 'SRCPORT': '1', 'AS': 'UMA', 'AL': 'CL"', 'CONNTYPE': '1WAY', 'DSTPORT': '1', 'OS': 'RDY'}
                    # out_list.append(fields_dictionary)
                    out_list[src] = fields_dictionary
        # print out_list
        return out_list

    ###################
    # get_crs_list
    ###################

    def get_crs_list(self):

        crs_list = self.get_list('rtrv-crs')
        return crs_list

    ######
    #  uses rtrv-port-sum::0
    #######


    def get_power_list(self):

        power_list = self.get_list('rtrv-port-sum::0')
        return power_list

    ######
    #
    # provides power information for each port.
    #
    def get_crs_power_list(self):

        crs_list = self.get_list('rtrv-crs')

        port_power_list = self.get_list('rtrv-port-sum::0')

        # add the power information from rtrv-port-sum
        # to the crs list.
        for crs, crsitems in crs_list.items():
            connid = crsitems['title']
            print ("Now processing:" + connid)
            conntype = crsitems['CONNTYPE']

            if (conntype == '1WAY'):
                # uni-direction
                ports = connid.split(">")
            else:
                # bi-direction
                ports = connid.split("-")

            srcport = ports[0]
            dstport = ports[1]

            # find the inpower from the port_power_list
            inpoweritems = port_power_list[srcport]
            outpoweritems = port_power_list[dstport]
            inpwr = inpoweritems['INPWR']
            if (inpwr == 'NONE'):
                inpower = float(-90)
            else:
                inpower = float(inpwr)

            outpwr = outpoweritems['OUTPWR']
            if (outpwr == 'NONE'):
                outpower = float(-90)
            else:
                outpower = float(outpwr)

            loss = inpower - outpower

            crsitems['F_INPWR'] = inpwr
            crsitems['F_OUTPWR'] = outpwr
            crsitems['F_LOSS'] = str(loss)

            if (conntype == '2WAY'):
                # populate the reverse powers.
                srcport = ports[1]
                dstport = ports[0]
                inpoweritems = port_power_list[srcport]
                outpoweritems = port_power_list[dstport]
                inpwr = inpoweritems['INPWR']
                outpwr = outpoweritems['OUTPWR']

                if (inpwr == 'NONE'):
                    inpower = float(-90)
                else:
                    inpower = float(inpwr)

                if (outpwr == 'NONE'):
                    outpower = float(-90)
                else:
                    outpower = float(outpwr)

                loss = inpower - outpower

                crsitems['R_INPWR'] = inpwr
                crsitems['R_OUTPWR'] = outpwr
                crsitems['R_LOSS'] = str(loss)

        return crs_list

    ################
    # delete_connection
    ################

    def delete_connection(self, source, dest, conn_id):
        command_string = 'dlt-crs::' + source + ',' + dest + ':::' + conn_id

        output = self.command(command_string)

        if self.completed_re.search(output) == None:
            raise (ValueError, 'command ' + command_string + ' failed.')

        if self.find_crs(source, dest) != None:
            raise (ValueError, 'could not delete cross connect between ' + source + ' and ' + dest)

    ########################
    # loopback
    ################

    def make_connection(self, source, dest, group_name=None, conn_name=None):

        src = str(source)
        dst = str(dest)
        if group_name is not None and conn_name is not None:
            command_string = 'ent-crs::' + src + ',' + dst + ':::' + group_name + ',1WAY,' + conn_name + ',,'
        elif group_name is not None:
            command_string = 'ent-crs::' + src + ',' + dst + ':::' + group_name + ',1WAY,'
        elif conn_name is not None:
            command_string = 'ent-crs::' + src + ',' + dst + ':::,1WAY,' + conn_name
        else:
            command_string = 'ent-crs::' + src + ',' + dst + ':::,1WAY,'

        output = self.command(command_string)

        if self.completed_re.search(output) is None:
            crs = self.find_crs(source, dest)
        if crs is not None:
            raise (ValueError, 'Port already in Connection: ' + crs['SRCPORT'] + '>' + crs['DSTPORT'])
        else:
            raise (ValueError, 'Unable to create a connection , cmd=' + command_string)


            ######################
            # delete loopback connections
            ######################

    # get port power from
    def getAllPower(self):
        inpower = -90
        outpower = -90
        power = []
        command_str = 'rtrv-port-sum::0'  # get power for all ports.
        output = self.command(command_str)
        lines = output.split('\r\n')
        data = ""

        for line in lines:
            data = ""
            tokens = line.split(',')
            for token in tokens:
                port_value = token.split(':')
                if len(port_value) > 1:  # port:alias=none,
                    portid = port_value
                    data += portid
                    data += ','

                    # now search for INPWR
                name_value = token.split('=')
                if len(name_value) > 1:
                    if name_value[0] == 'INPWR':
                        if name_value[1] == 'NONE':
                            inpower = -90
                        else:
                            inpower = float(name_value[1])
                            data += str(inpower)
                            data += ','

                            # now search for OUTPWR
                name_value = token.split('=')
                if len(name_value) > 1:
                    if name_value[0] == 'OUTPWR':
                        if name_value[1] == 'NONE':
                            outpower = -90
                        else:
                            outpower = float(name_value[1])
                            data += str(outpower)
                            data += ','

        power.append(data)

        return power

    # gets power for 'a input' and 'a output' port
    def getPower(self, inport, outport):
        inpower = -90
        outpower = -90
        power = []

        command_str = 'rtrv-port-sum::' + str(inport)
        output = self.command(command_str)
        lines = output.split('\n')

        for line in lines:
            line = line.strip()
            if line.startswith('"'):
                tokens = line.split(',')
                for token in tokens:
                    name_value = token.split('=')
                    if len(name_value) > 1:
                        if name_value[0] == 'INPWR':
                            if name_value[1] == 'NONE':
                                inpower = -90
                            else:
                                inpower = float(name_value[1])
                            power.append(inpower)


                            # Get output port power
        command_str = 'rtrv-port-sum::' + str(outport)
        output = self.command(command_str)
        lines = output.split('\n')
        for line in lines:
            line = line.strip()
            if line.startswith('"'):
                tokens = line.split(',')
                for token in tokens:
                    name_value = token.split('=')
                    if len(name_value) > 1:
                        if name_value[0] == 'OUTPWR':
                            if name_value[1] == 'NONE':
                                outpower = -90
                            else:
                                outpower = float(name_value[1])
                            power.append(outpower)
        print (power)
        return power
