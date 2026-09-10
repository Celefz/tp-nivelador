import socket
import logger
import safe_socket
import protocol
from lottery import Lottery

_PACKET_LEN_SIZE = 2


def _recv_packet(client_socket):
    header = safe_socket.recv_all(client_socket, _PACKET_LEN_SIZE)
    packet_size = int.from_bytes(header, byteorder="big")
    packet = safe_socket.recv_all(client_socket, packet_size)
    return packet


class Server:
    def __init__(
        self,
        server_host: str,
        server_port: int,
        storage_path: str = "/tmp/lottery_bets.csv",
    ) -> None:
        self.server_host = server_host
        self.server_port = server_port
        with open(storage_path, "a"):
            pass
        self.lottery = Lottery(storage_path)

    def _handle_client(self, client_socket):
        action = "handle-client"

        try:
            logger.info(action, logger.LogResult.in_progress)
            while True:
                packet = _recv_packet(client_socket)
                if protocol.is_end_packet(packet):
                    agency_id = packet[1]
                    break

                bet = protocol.deserialize_bet(packet)
                self.lottery.store_bets([bet])

            for stored_bet in self.lottery.load_bets():
                if self.lottery.has_won(stored_bet) and stored_bet.agency_id == agency_id:
                    safe_socket.send_all(client_socket, protocol.serialize_bet(stored_bet))

            safe_socket.send_all(client_socket, protocol.serialize_end())

            logger.info(action, logger.LogResult.success)

        except Exception as e:
            logger.error(
                action, logger.LogResult.fail, "err", e
            )

        finally:
            client_socket.close()

    def run(self):
        action = "accept-connection"
        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as server_socket:
            server_socket.bind((self.server_host, self.server_port))
            server_socket.listen()
            while True:
                try:
                    logger.info(action, logger.LogResult.in_progress)
                    client_socket, _ = server_socket.accept()
                except Exception as e:
                    logger.error(action, logger.LogResult.fail)
                    raise e
                logger.info(action, logger.LogResult.success)

                self._handle_client(client_socket)
