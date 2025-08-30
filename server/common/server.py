import logging
import signal
import socket

from common import communication, utils


class Server:
    def __init__(self, port, listen_backlog):
        # Initialize server socket
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(("", port))
        self._server_socket.listen(listen_backlog)
        self._running = False
        self._finished: set[int] = set()
        self._winners: dict[int, list[str]] = {}
        self._raffle_done: bool = False

    def run(self):
        """
        Dummy Server loop

        Server that accept a new connections and establishes a
        communication with a client. After client with communucation
        finishes, servers starts to accept new connections again
        """
        self._running = True
        signal.signal(signal.SIGTERM, self.__stop_running)
        while self._running:
            try:
                client_sock = self.__accept_new_connection()
                self.__handle_client_connection(client_sock)
            except OSError:
                if not self._running:
                    break
                raise
        logging.shutdown()

    def __raffle(self):
        try:
            bets = utils.load_bets()
            winners = [(b.agency, b.document) for b in bets if utils.has_won(b)]
            for w in winners:
                self._winners.setdefault(w[0], []).append(w[1])
            logging.info("action: sorteo | result: success")
            self._raffle_done = True
        except Exception as e:
            logging.error("action: sorteo | result: fail | error: %s", e)
            return

    def __send_winners(self, agency_id, sock):
        try:
            communication.Winners(self._winners.get(agency_id, [])).write_to(sock)
            logging.info(
                "action: enviar_ganadores | result: success | agencia: %d", agency_id
            )
        except communication.ProtocolError as e:
            logging.error(
                "action: enviar_ganadores | result: fail | agencia: %d | error: %s",
                agency_id,
                e,
            )

    def __process_msg(self, msg, client_sock: socket.socket) -> bool:
        if msg.opcode == communication.Opcodes.NEW_BETS:
            try:
                utils.store_bets(msg.bets)
                for bet in msg.bets:
                    logging.info(
                        "action: apuesta_almacenada | result: success | dni: %s | numero: %s",
                        bet.document,
                        bet.number,
                    )
            except Exception as e:
                communication.BetsRecvFail().write_to(client_sock)
                logging.error(
                    "action: apuesta_recibida | result: fail | cantidad: %d", msg.amount
                )
                return True
            logging.info(
                "action: apuesta_recibida | result: success | cantidad: %d",
                msg.amount,
            )
            communication.BetsRecvSuccess().write_to(client_sock)
            return True
        if msg.opcode == communication.Opcodes.FINISHED:
            self._finished.add(msg.agency_id)
            if len(self._finished) == 5 and not self._raffle_done:
                self.__raffle()
            return False
        if msg.opcode == communication.Opcodes.REQUEST_WINNERS:
            if self._raffle_done and msg.agency_id in self._finished:
                self.__send_winners(msg.agency_id, client_sock)
            return False

    def __handle_client_connection(self, client_sock):
        while True:
            msg = None
            try:
                msg = communication.recv_msg(client_sock)
                addr = client_sock.getpeername()
                logging.info(
                    "action: receive_message | result: success | ip: %s | opcode: %i",
                    addr[0],
                    msg.opcode,
                )
                if not self.__process_msg(msg, client_sock):
                    break
            except communication.ProtocolError as e:
                logging.error("action: receive_message | result: fail | error: %s", e)
            except EOFError:
                break
            except OSError as e:
                logging.error("action: send_message | result: fail | error: %s", e)
                break
        client_sock.close()

    def __accept_new_connection(self):
        """
        Accept new connections

        Function blocks until a connection to a client is made.
        Then connection created is printed and returned
        """
        # Connection arrived
        logging.info("action: accept_connections | result: in_progress")
        c, addr = self._server_socket.accept()
        logging.info(f"action: accept_connections | result: success | ip: {addr[0]}")
        return c

    def __stop_running(self, _signum, _frame):
        self._running = False
        self._server_socket.close()
