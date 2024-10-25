import socket
import sys
def main(sid, port):
    # Create a TCP/IP socket
    sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    # Connect the socket to the port where the server is listening
    server_address = ('10.224.92.152', 18201)
    print('connecting to %s port %s' % server_address)
    sock.connect(server_address)
    try:
        message = str(sid) + ' ' + str(port)
        sock.sendall(message)
    finally:
        print('closing socket')
        sock.close()

if __name__ == "__main__":
    while True:
        input = raw_input("input switchId and phy port, separated by space:")
        paraList = input.strip().split(' ')
        main(paraList[0], paraList[1])
