import socket
import logger
import safe_socket
import protocol
from lottery import Lottery
import threading

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
        agency_quorum_min: int = 1,
    ) -> None:
        self.server_host = server_host
        self.server_port = server_port
        self.agency_quorum_min = max(1, agency_quorum_min)
        self._storage_lock = threading.Lock()
        self._quorum_condition = threading.Condition()
        self._completed_agencies = set()

        with open(storage_path, "a"):
            pass

        self.lottery = Lottery(storage_path)

    def _register_agency(self, agency_id):
        with self._quorum_condition:
            self._completed_agencies.add(agency_id)
            if len(self._completed_agencies) >= self.agency_quorum_min:
                self._quorum_condition.notify_all()

    def _wait_for_agency_quorum(self):
        with self._quorum_condition:
            while len(self._completed_agencies) < self.agency_quorum_min:
                self._quorum_condition.wait()

    def _handle_client(self, client_socket):
        action = "handle-client"

        try:
            logger.info(action, logger.LogResult.in_progress)
            while True:
                packet = _recv_packet(client_socket)

                if protocol.is_end_packet(packet):
                    agency_id = packet[1]
                    break

                bets = protocol.deserialize_batch(packet)

                with self._storage_lock:
                    self.lottery.store_bets(bets)

            self._register_agency(agency_id)
            self._wait_for_agency_quorum()

            with self._storage_lock:
                stored_bets =  self.lottery.load_bets()
                for stored_bet in stored_bets:
                    if self.lottery.has_won(stored_bet) and stored_bet.agency_id == agency_id:
                        safe_socket.send_all(
                            client_socket,
                            protocol.serialize_batch([stored_bet], agency_id),
                        )

                safe_socket.send_all(client_socket, protocol.serialize_end())

            logger.info(action, logger.LogResult.success)

        except Exception as e:
            logger.error(action, logger.LogResult.fail, "err", e)

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
                client_thread = threading.Thread(
                    target=self._handle_client,
                    args=(client_socket,),
                    daemon=True,
                )
                client_thread.start()
