import logging
import signal
import socket
import threading

from app import protocol, service


class Server:
    def __init__(self, port, listen_backlog, clients_amount):
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(("", port))
        self._server_socket.listen(listen_backlog)
        self._stop = threading.Event()
        self._finished = threading.Barrier(int(clients_amount))
        self._winners: dict[int, list[str]] = {}
        self._raffle_lock = threading.Lock()
        self._storage_lock = threading.Lock()
        self._threads: list[threading.Thread] = []
        self._raffle_done = threading.Event()

    def run(self):
        signal.signal(signal.SIGTERM, self.__handle_sigterm)
        while not self._stop.is_set():
            try:
                client_sock = self.__accept_new_connection()
                t = threading.Thread(
                    target=self.__handle_client_connection, args=(client_sock,)
                )
                self._threads.append(t)
                t.start()
            except OSError:
                if self._stop.is_set():
                    break
                raise
        for t in self._threads:
            t.join()
        logging.shutdown()

    def __accept_new_connection(self):
        """
        Accept new connections

        Function blocks until a connection to a client is made.
        Then connection created is printed and returned
        """
        logging.info("action: accept_connections | result: in_progress")
        c, addr = self._server_socket.accept()
        logging.info(f"action: accept_connections | result: success | ip: {addr[0]}")
        return c

    def __handle_client_connection(self, client_sock):
        while not self._stop.is_set():
            msg = None
            try:
                msg = protocol.recv_msg(client_sock)
                addr = client_sock.getpeername()
                logging.info(
                    "action: receive_message | result: success | ip: %s | opcode: %i",
                    addr[0],
                    msg.opcode,
                )
                if not self.__process_msg(msg, client_sock):
                    break
            except protocol.ProtocolError as e:
                logging.error("action: receive_message | result: fail | error: %s", e)
            except EOFError:
                break
            except OSError as e:
                logging.error("action: send_message | result: fail | error: %s", e)
                break
        client_sock.close()

    def __process_msg(self, msg, client_sock) -> bool:
        if msg.opcode == protocol.Opcodes.NEW_BETS:
            try:
                with self._storage_lock:
                    service.store_bets(msg.bets)
                    for bet in msg.bets:
                        logging.info(
                            "action: apuesta_almacenada | result: success | dni: %s | numero: %s",
                            bet.document,
                            bet.number,
                        )
            except Exception as e:
                protocol.BetsRecvFail().write_to(client_sock)
                logging.error(
                    "action: apuesta_recibida | result: fail | cantidad: %d", msg.amount
                )
                return True
            logging.info(
                "action: apuesta_recibida | result: success | cantidad: %d",
                msg.amount,
            )
            protocol.BetsRecvSuccess().write_to(client_sock)
            return True
        if msg.opcode == protocol.Opcodes.FINISHED:
            self._finished.wait()
            with self._raffle_lock:
                if not self._raffle_done.is_set():
                    self.__raffle()
            return True
        if msg.opcode == protocol.Opcodes.REQUEST_WINNERS:
            self._raffle_done.wait()
            self.__send_winners(msg.agency_id, client_sock)
            return False

    def __raffle(self):
        try:
            self._winners = service.compute_winners()
            logging.info("action: sorteo | result: success")
            self._raffle_done.set()
        except Exception as e:
            logging.error("action: sorteo | result: fail | error: %s", e)
            return

    def __send_winners(self, agency_id, sock):
        try:
            protocol.Winners(self._winners.get(agency_id, [])).write_to(sock)
            logging.info(
                "action: enviar_ganadores | result: success | agencia: %d", agency_id
            )
        except protocol.ProtocolError as e:
            logging.error(
                "action: enviar_ganadores | result: fail | agencia: %d | error: %s",
                agency_id,
                e,
            )

    def __handle_sigterm(self, *_):
        self._stop.set()
        self._server_socket.close()
