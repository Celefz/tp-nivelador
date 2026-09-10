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

        self._shutdown_event = threading.Event()
        self._server_socket = None
        self._client_threads = []
        self._client_sockets = set()
        self._clients_lock = threading.Lock()

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
            while (
                len(self._completed_agencies) < self.agency_quorum_min
                and not self._shutdown_event.is_set()
            ):
                self._quorum_condition.wait(timeout=0.1)

    def _handle_client(self, client_socket):
        action = "handle-client"

        with self._clients_lock:
            self._client_sockets.add(client_socket)

        try:
            logger.info(action, logger.LogResult.in_progress)

            while True:
                if self._shutdown_event.is_set():
                    return

                packet = _recv_packet(client_socket)
                if not packet:
                    return

                if protocol.is_end_packet(packet):
                    agency_id = packet[1]
                    break

                agency_id = packet[1]
                bets = protocol.deserialize_batch(packet)

                with self._storage_lock:
                    self.lottery.store_bets(bets)

                safe_socket.send_all(client_socket, protocol.serialize_ack(agency_id))

            self._register_agency(agency_id)
            self._wait_for_agency_quorum()

            if self._shutdown_event.is_set():
                return

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

            with self._clients_lock:
                self._client_sockets.discard(client_socket)

    def shutdown(self):
        logger.info("shutdown", logger.LogResult.in_progress)

        self._shutdown_event.set()

        with self._quorum_condition:
            self._quorum_condition.notify_all()

        if self._server_socket is not None:
            self._server_socket.close()

        with self._clients_lock:
            client_sockets = list(self._client_sockets)
            client_threads = list(self._client_threads)

        for client_socket in client_sockets:
            client_socket.close()

        for client_thread in client_threads:
            client_thread.join()

        logger.info("shutdown", logger.LogResult.success)

    def run(self):
        action = "accept-connection"
        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as server_socket:
            self._server_socket = server_socket
            server_socket.bind((self.server_host, self.server_port))
            server_socket.listen()

            while not self._shutdown_event.is_set():
                try:
                    logger.info(action, logger.LogResult.in_progress)
                    client_socket, _ = server_socket.accept()

                except OSError:
                    if self._shutdown_event.is_set():
                        break

                    logger.error(action, logger.LogResult.fail)
                    raise

                logger.info(action, logger.LogResult.success)
                client_thread = threading.Thread(
                    target=self._handle_client,
                    args=(client_socket,),
                )

                with self._clients_lock:
                    self._client_threads.append(client_thread)

                client_thread.start()

            self._server_socket = None