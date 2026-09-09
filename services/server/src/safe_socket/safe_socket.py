import socket

def recv_all(socket: socket.socket, size):
    left_to_read = size
    bytes_total = b''

    while left_to_read:
        bytes = socket.recv(left_to_read)

        if not bytes:
            break

        left_to_read -= len(bytes)
        bytes_total += bytes

    return bytes_total


def send_all(socket: socket.socket, bytes):
    written = 0

    while written < len(bytes):
        n = socket.send(bytes[written:])
        written += n

    return written
