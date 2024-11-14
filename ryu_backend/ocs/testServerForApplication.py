import socket,time
from datetime import datetime

sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
sock.bind(("", 18201))
sock.listen(2)
while True:

    print datetime.now().strftime('%H:%M:%S.%f')
    conn, address = sock.accept()
    time.sleep(20)
    try:
        data = conn.recv(1024)
        paraList = str(data).strip().split(' ')
        print("==%s, received:%s from %s"%(datetime.now().strftime('%H:%M:%S.%f'), paraList, address))
    finally:
        conn.close()