import os
import shutil
from datetime import datetime
from time import strftime


class Log:
    def __init__(self, module_name=None, swnum=None):
        if not os.path.exists("./log"):
            os.makedirs("./log")
        if module_name == None:
            self.debuglog = None
            self.scriptlog = None
            self.outputlog = None
            self.stragglelog = None
        elif swnum != None:
            currtime = strftime("%Y-%m-%d-%H-%M-%S")
            module_debug = "./log/log_debug_" + str(swnum) + "_" + module_name + "_" + currtime + ".txt"
            module_script = "./log/log_script_" + str(swnum) + "_" + module_name + "_" + currtime + ".txt"
            module_output = "./log/log_logfile_" + str(swnum) + "_" + module_name + "_" + currtime + ".csv"
            module_straggle = "./log/log_straggle_" + str(swnum) + "_" + module_name + "_" + currtime + ".csv"

            self.debuglog = open(module_debug, "w")
            self.scriptlog = open(module_script, "w")
            self.outputlog = open(module_output, "w")
            self.stragglelog = open(module_straggle, "w")

        else:  # only module name specified no swnum
            currtime = strftime("%Y-%m-%d-%H-%M-%S")
            module_debug = "./log/log_debug_" + module_name + "_" + currtime + ".txt"
            module_script = "./log/log_script_" + module_name + "_" + currtime + ".txt"
            module_output = "./log/log_logfile_" + module_name + "_" + currtime + ".csv"
            module_straggle = "./log/log_straggle_" + module_name + "_" + currtime + ".csv"

            self.debuglog = open(module_debug, "w")
            self.scriptlog = open(module_script, "w")
            self.outputlog = open(module_output, "w")
            self.stragglelog = open(module_straggle, "w")

        self.swnum = None

        # Move all the old logs into the old folder.

    def archiveLogs(self):
        print "Moving all old logs into old/"
        source = os.listdir("./log/")
        destination = "./log/old/"
        for files in source:
            if files.endswith(".txt") or files.endswith(".csv"):
                logfile = './log/' + files
                shutil.move(logfile, destination)

    def setSwitchNum(self, swnum):
        self.swnum = swnum

    def setModule(self, module_name, swnum=None):
        # First close all the existing logs.
        if self.debuglog != None:
            self.debuglog.close()
        if self.scriptlog != None:
            self.scriptlog.close()
        if self.outputlog != None:
            self.outputlog.close()
        if self.stragglelog != None:
            self.stragglelog.close()

        if swnum == None:
            currtime = strftime("%Y-%m-%d-%H-%M-%S")
            module_debug = "./log/log_debug_" + module_name + "_" + currtime + ".txt"
            module_script = "./log/log_script_" + module_name + "_" + currtime + ".txt"
            module_output = "./log/log_logfile_" + module_name + "_" + currtime + ".csv"
            module_straggle = "./log/log_straggle_" + module_name + "_" + currtime + ".csv"

        else:
            #    swnum=self.swnum
            currtime = strftime("%Y-%m-%d-%H-%M-%S")
            module_debug = "./log/log_debug_" + str(swnum) + "_" + module_name + "_" + currtime + ".txt"
            module_script = "./log/log_script_" + str(swnum) + "_" + module_name + "_" + currtime + ".txt"
            module_output = "./log/log_logfile_" + str(swnum) + "_" + module_name + "_" + currtime + ".csv"
            module_straggle = "./log/log_straggle_" + str(swnum) + "_" + module_name + "_" + currtime + ".csv"

        self.debuglog = open(module_debug, "w")
        self.scriptlog = open(module_script, "w")
        self.outputlog = open(module_output, "w")
        self.stragglelog = open(module_straggle, "w")

    def logMsg(self, logtype, msg):
        if logtype == "debug" or logtype == "DEBUG":
            logObj = self.debuglog
        elif logtype == "script" or logtype == "SCRIPT":
            logObj = self.scriptlog
        elif logtype == "output" or logtype == "OUTPUT":
            logObj = self.outputlog
        elif logtype == "straggle" or logtype == "STRAGGLE":
            logObj = self.stragglelog
        else:
            raise ValueError, 'Invalid logtype:' + logtype

        line = datetime.now().isoformat() + ',' + msg
        logObj.write(line)
        logObj.write('\n')
        logObj.flush()

    def getdebuglog(self):
        return self.debuglog

    def getscriptlog(self):
        return self.scriptlog

    def getoutputlog(self):
        return self.outputlog

    def getstragglelog(self):
        return self.stragglelog

    def closeLogs(self):
        self.debuglog.close()
        self.scriptlog.close()
        self.outputlog.close()
        self.stragglelog.close()
