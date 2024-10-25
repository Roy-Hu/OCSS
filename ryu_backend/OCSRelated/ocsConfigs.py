from flask import Flask

from .connections import GxcConnections
from .optionsForShareBackup import Options

fatTree = [(31,77), (32,78), (73,79), (74,80), #pod1 edge links
            (39,85), (40,86), (81,87), (82,88), #pod2 edge links
            (15,25), (26,65), (16,27), (28,66), #pod1 agg links
            (23,33), (24,35), (34,69), (36,70), #pod2 agg links
            (1,9), (3,10), (7,11), (12,61),     #pod1 core links
            (2,17), (4,18), (8,19), (20,62)]    #pod2 core links

f10     = [(31,77), (32,78), (73,79), (74,80), #pod1 edge links
            (39,85), (40,86), (81,87), (82,88), #pod2 edge links
            (15,25), (26,65), (16,27), (28,66), #pod1 agg links
            (23,33), (24,35), (34,69), (36,70), #pod2 agg links
            (1,9), (3,10), (7,11), (12,61),     #pod1 core links
            (2,17), (4,19), (8,18), (20,62)]    #pod2 core links

def create_initial_connections(connectionObj, low_power=[1, 2, 3, 4], high_power=[41, 42, 43, 45, 46, 47], splitter=101, fanout=4):
    connectionObj.ent_crs_fiber(low_power, low_power)
    spliiter_port = range(splitter, splitter + len(high_power) * fanout, fanout)
    connectionObj.ent_crs_fiber(high_power, spliiter_port)
    connectionObj.ent_crs_fiber(spliiter_port, high_power)


def createSelfLoop():
    app = Flask(__name__)
    connectionObj = GxcConnections(Options())
    connectionObj.dlt_crs_fiber_all()
    low_power_port_start = 1
    low_power_count = 40
    high_power_port_start = 41
    high_power_count = 20
    splitter_start = 89
    low_power_ports = range(low_power_port_start, low_power_port_start + low_power_count)
    low_power_secondHalf = range(61, 89)
    low_power_ports += low_power_secondHalf

    high_power_ports = range(high_power_port_start, high_power_port_start + high_power_count)
    high_power_ports.remove(52)
    high_power_ports.remove(56)
    high_power_ports.remove(60)
    create_initial_connections(connectionObj, low_power=low_power_ports, high_power=high_power_ports, splitter=splitter_start, fanout=4)
    connectionObj.ed_param(20)

