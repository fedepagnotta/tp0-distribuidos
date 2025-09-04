import logging
import signal
import socket
from collections import deque
from enum import Enum, auto
from typing import Deque, Tuple

from app import protocol, service


class HandleConnAction(Enum):
    """Directive for how the caller should proceed with a client connection.

    Values:
      - CLOSE: stop handling this connection and close the socket now.
      - EXIT:  stop reading, but don't close the socket (the server will reply
               later, e.g., after FINISHED, and will close it at that time).
      - KEEP:  keep the connection open and continue reading more messages.
    """

    CLOSE = auto()
    EXIT = auto()
    KEEP = auto()


class Server:
    def __init__(self, port, listen_backlog, clients_amount):
        """Initialize listening socket and server state.

        Binds and listens on the given port. Tracks:
        - _running: main-loop flag toggled by SIGTERM handler.
        - _finished: queue of agencies (id and connection) that already sent FINISHED.
        - _winners: winners grouped by agency after the raffle.
        - _clients_amount: total agencies expected to finish.
        """
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(("", port))
        self._server_socket.listen(listen_backlog)
        self._running = False
        self._finished: Deque[Tuple[int, socket.socket]] = deque()
        self._winners: dict[int, list[str]] = {}
        self._clients_amount = int(clients_amount)

    def run(self):
        """Main server loop.

        Installs SIGTERM handler, accepts connections sequentially,
        and handles each client to completion. On SIGTERM the listening
        socket is closed to unblock accept(), the loop drains, and
        logging.shutdown() is called.
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
        """Compute winners once all agencies finished.

        Delegates to service.compute_winners() and stores the result.
        Logs success/failure. Intended to run exactly once.
        """
        try:
            self._winners = service.compute_winners()
            logging.info("action: sorteo | result: success")
        except Exception as e:
            logging.error("action: sorteo | result: fail | error: %s", e)
            return

    def __send_winners(self, agency_id, sock):
        """Serialize and send WINNERS for the given agency.

        Frames the response via protocol helpers and logs the outcome.
        Any ProtocolError during framing is logged.
        """
        try:
            protocol.Winners(self._winners.get(agency_id, [])).write_to(sock)
            logging.info(
                "action: enviar_ganadores | result: success | agencia: %d", agency_id
            )
        except (protocol.ProtocolError, OSError) as e:
            logging.error(
                "action: enviar_ganadores | result: fail | agencia: %d | error: %s",
                agency_id,
                e,
            )

    def __process_msg(self, msg, client_sock: socket.socket) -> HandleConnAction:
        """Route a decoded message and apply server semantics.

        Returns:
          - HandleConnAction.KEEP  -> keep reading on this connection.
          - HandleConnAction.CLOSE -> reply (if applicable) and close now.
          - HandleConnAction.EXIT  -> stop reading but DO NOT close here
                                      (socket is queued to be answered later).

        Semantics:
          - NEW_BETS:
              * Persist all bets; on success, send BETS_RECV_SUCCESS, log the
                stored count, and return KEEP.
              * On any error, send BETS_RECV_FAIL, log the failure, and return CLOSE.
          - FINISHED:
              * Enqueue (agency_id, socket). If all agencies have finished and the
                raffle is not done, run the raffle and push WINNERS to every queued
                socket (closing each).
              * Return EXIT so the caller stops reading this connection without
                closing it here (the socket will be closed after winners are sent).
        """
        if msg.opcode == protocol.Opcodes.NEW_BETS:
            try:
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
                return HandleConnAction.CLOSE
            logging.info(
                "action: apuesta_recibida | result: success | cantidad: %d",
                msg.amount,
            )
            protocol.BetsRecvSuccess().write_to(client_sock)
            return HandleConnAction.KEEP

        if msg.opcode == protocol.Opcodes.FINISHED:
            self._finished.append((msg.agency_id, client_sock))
            if len(self._finished) == self._clients_amount:
                self.__raffle()
                while self._finished:
                    (ag_id, sock) = self._finished.popleft()
                    self.__send_winners(ag_id, sock)
                    sock.close()
            return HandleConnAction.EXIT

    def __handle_client_connection(self, client_sock):
        """Handle a single client synchronously.

        Repeatedly receives a framed message (protocol.recv_msg), logs it,
        and delegates to __process_msg. Closes the client socket at exit if
        __process_msg returns CLOSE action.
        """
        curr_handle_conn_action = HandleConnAction.CLOSE
        while True:
            msg = None
            try:
                msg = protocol.recv_msg(client_sock)
                addr = client_sock.getpeername()
                logging.info(
                    "action: receive_message | result: success | ip: %s | opcode: %i",
                    addr[0],
                    msg.opcode,
                )
                curr_handle_conn_action = self.__process_msg(msg, client_sock)
                if curr_handle_conn_action in (
                    HandleConnAction.CLOSE,
                    HandleConnAction.EXIT,
                ):
                    break
            except protocol.ProtocolError as e:
                logging.error("action: receive_message | result: fail | error: %s", e)
            except EOFError:
                break
            except OSError as e:
                logging.error("action: send_message | result: fail | error: %s", e)
                break
        if curr_handle_conn_action == HandleConnAction.CLOSE:
            client_sock.close()

    def __accept_new_connection(self):
        """Accept a new client connection.

        Blocks in accept(), logs the remote address, and returns
        the connected socket.
        """
        logging.info("action: accept_connections | result: in_progress")
        c, addr = self._server_socket.accept()
        logging.info(f"action: accept_connections | result: success | ip: {addr[0]}")
        return c

    def __stop_running(self, _signum, _frame):
        """SIGTERM handler.

        Marks the server as stopping and closes the listening socket
        to wake up accept().
        """
        self._running = False
        self._server_socket.close()
        while self._finished:
            _ag, sock = self._finished.popleft()
            try:
                sock.close()
            except OSError:
                pass
