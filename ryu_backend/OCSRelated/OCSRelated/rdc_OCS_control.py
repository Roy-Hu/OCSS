from flask import Flask

from .connections import GxcConnections
from .optionsForShareBackup import Options

def create_initial_connections():
    serverPorts = xrange(33, 41) #[33-40] are connected to server
    torPorts = xrange(61, 69)
    connectionObj = GxcConnections(Options())
    connectionObj.ent_crs_fiber(serverPorts+torPorts, torPorts+serverPorts)

