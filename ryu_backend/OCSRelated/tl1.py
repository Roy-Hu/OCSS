import re
import telnetlib
from datetime import datetime
import time
import log


class ExPortsDone(Exception):
    pass


class TL1:
    MATCHED_PROMPT_INDEX_MAIN = 0
    TIMEOUT = 600

    def __init__(self, options):
        self.ip = options.get_serverip()
        self.port = int(options.get_serverport())
        self.username = options.get_username()
        self.password = options.get_password()

        self.ports_list = {}
        self.prompt_list = ['<']
        self.done_re = re.compile('agent> ')
        self.completed_re = re.compile('COMPLD')
        self.deny_re = re.compile('DENY')
        self.telnet_obj = None
        self.swtype = None
        self.swserial = None

        self.mylog = log.Log("TL1")

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

    def closeLogs(self):
        self.mylog.closeLogs()

        # Prints to the Console

    def printMsg(self, msg):
        time = datetime.now().isoformat()
        print time + "," + msg

    def connect(self):
        msg = "Connecting to ... IP:%s Port:%s" % (self.ip, self.port)
        self.scriptMsg(msg)
        self.telnet_obj = telnetlib.Telnet(self.ip, int(self.port))
        self.telnet_obj.read_until('<')
        msg = "Connection to IP:%s Done" % self.ip
        self.scriptMsg(msg)

    def login(self):
        msg = "username is:\t" + self.username
        self.scriptMsg(msg)
        msg = "password is:\t" + self.password
        self.scriptMsg(msg)

        command_str = 'act-user::{}:1::{};'.format(self.username, self.password)
        self.scriptMsg(command_str)
        output = self.command(command_str)

        if self.completed_re.search(output) is None:  # None - No Match Found
            raise ValueError, 'Failed to login with username ' + self.username + ' and password ' + self.password

    def logout(self):
        command_str = 'canc-user:::{};'.format(self.username)
        self.scriptMsg(command_str)
        output = self.command(command_str)

        if self.completed_re.search(output) is None:  # None - No Match Found
            raise ValueError, 'command ' + command_str + ' failed.'

    def close(self):
        self.telnet_obj.close()
        self.closeLogs()

    # search for a pattern in output,
    # if match found return the value
    def pattern_value(self, output, searchstr):
        self.scriptMsg("Searching for:" + searchstr + " Output:" + output)
        for line in output.split('\n'):

            m = re.search(searchstr, line)
            if m:  # match found
                data = line.split(':')
                if len(data) == 2:
                    dataline = data[1]
                else:
                    dataline = data[0]

                for token in dataline.split(','):
                    key, val = token.split('=')
                    if (key == searchstr):
                        value = val.split('"')
                        return value[0]
        return None

    def sendline(self, command):
        self.command(command)

    def command(self, command, prompt=None, cr_lf=True, timeout=None):
        # config the prompt list
        if prompt is not None:
            prompt_list = [re.compile(prompt)]
        else:
            prompt_list = self.prompt_list

        # append ending mark to the command
        if cr_lf:
            command += '\n'
        self.telnet_obj.write(command)
        # print 'cmd:\t', command

        # reply timeout
        if timeout is None:
            timeout = TL1.TIMEOUT

        index, matchedindex, reply = self.telnet_obj.expect(prompt_list, timeout)
        if index == -1:
            raise ValueError, 'TELNET_ERROR: telnet to device ' + self.ip + ':' + str(self.port) + ' timed out. Data left in buffer was ' + str(reply) + '.'
        # print 'reply:\t', reply
        
        self.debugMsg(reply)
        return reply

    def command_eager_reply(self, command, prompt=None, cr_lf=True, timeout=None):

        prompt_list = []

        if timeout == None:
            timeout = TL1.TIMEOUT
        if prompt != None:  # None is no match found
            prompt_list = [re.compile(prompt)]
            msg = "prompt list is     " + 'prompt_list'
            self.scriptMsg(msg)
        else:
            prompt_list = self.prompt_list
            # msg = "prompt list is              " + `prompt_list`
            # self.scriptMsg(msg)
        # print prompt_list
        if cr_lf == True:
            self.telnet_obj.write(command + ';\n')
            # msg = "telnet object command   " + command
            # self.scriptMsg(msg)
        else:
            self.telnet_obj.write(command)

        return self.telnet_obj.read_very_eager()
        # self.scriptMsg(recv_buffer)
        #   if index == -1:
        #      raise ValueError, 'TELNET_ERROR: telnet to device ' + self.ip + ':' + str(self.port) + ' timed out. Data left in buffer was ' + str(recv_buffer) + '.'

        # return None

    def get_SwitchType(self):
        commandstr = 'rtrv-ne'
        switchtype_re = 'CHASSISTYPE'
        output = self.command(commandstr)
        swtype = self.pattern_value(output, switchtype_re)
        if swtype == None:
            raise ValueError, "Unable to find switch type:X320 or X272 "
        else:
            self.swtype = swtype[1:]  # X320,X272, remove the X
            msg = 'Switch Type:' + self.swtype
            self.scriptMsg(msg)
        return self.swtype

    def get_SwitchSerial(self):
        commandstr = 'rtrv-ne'
        switchnum_re = 'SERIALNUMBER'
        output = self.command(commandstr)
        swnum = self.pattern_value(output, switchnum_re)
        if type is None:
            raise ValueError, "Unable to find switch serial number "
        else:
            self.swserial = swnum[5:]  # C00000729, strip leading 5(max sw upto 1999)
        return self.swserial

    # if a port is valid x.y.z or npid
    def validate_port(self, port):
        portStr = str(port)

        try:
            fields = (portStr).split(".")

        except:
            print 'exception in split'
        else:
            if len(fields) >= 3:
                expr = '[1-6]\.[1-8]\.[1-8]'
                m = re.match(expr, portStr)
                if m:
                    return 3
                else:
                    msg = "Invalid Port:" + port + " has to be in this range [1-6].[1-8].[1-8]"
                    raise ValueError, msg
                    # return 0
            elif len(fields) == 1:
                portid = int(portStr)
                if (portid < 0 or portid >= 384):
                    msg = "Invalid Port:" + portStr + " has to be 1<= port <= 384 "
                    raise ValueError, msg
                    # return 0
                else:
                    return 1

    # conver the port to x.y.z or npid depending on the switchtype
    def convert_port(self, portStr):
        try:
            porttype = self.validate_port(portStr)
        except:
            raise ValueError, 'Invalid Port:' + str(portStr)
        else:
            if self.swtype == "320" and porttype == 3:  # port already in x.y.z format
                return portStr
            elif self.swtype == "320" and porttype == 1:  # convert
                port = int(portStr)
                shelf = (port / 64)
                rem = (port % 64)
                slot = (rem / 8)
                rem = (rem % 8)
                channel = rem
                portxyz = str(shelf + 1) + "." + str(slot + 1) + "." + str(channel + 1)
                return str(portxyz)
            elif self.swtype == "272" and porttype == 1:  # port already in npid format
                return portStr
            elif self.swtype == "272" and porttype == 3:  # its okay. on a 272 we allow x.y.z format
                return portStr

    # generate the port list based on the swtype.
    def auto_port_list(self):

        if (self.swtype == '272'):
            ports_list = {k: k for k in range(1, 272 + 1)}
        else:
            numports = 324
            port = 1
            try:
                for shelf in range(1, 7):
                    for slot in range(1, 9):
                        for channel in range(1, 9):

                            portid = str(shelf) + "." + str(slot) + "." + str(channel)
                            self.ports_list[port] = portid
                            port = port + 1

                            if (port > numports):  # to exit out of nested loops
                                raise ExPortsDone

            except ExPortsDone:
                pass

        return self.ports_list
