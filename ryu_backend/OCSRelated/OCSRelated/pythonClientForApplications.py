import socket
import sys
from datetime import datetime
def main(paraString):
    # Create a TCP/IP socket
    sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    # Connect the socket to the port where the server is listening
    #server_address = ('10.224.92.152', 18201)
    server_address = ('168.7.139.226',18201)
    sock.connect(server_address)
    try:
        print 'before',datetime.now().strftime('%H:%M:%S.%f')
        sock.sendall(paraString)
        print 'after', datetime.now().strftime('%H:%M:%S.%f')
    finally:
        sock.close()

if __name__ == "__main__":
    paraString = sys.argv[1]+' '+sys.argv[2]+' '+sys.argv[3]
    #print paraString
    main(paraString)
