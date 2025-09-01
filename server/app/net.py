import logging
import signal
import socket

from app import protocol, service


class Server:
    def __init__(self, port, listen_backlog):
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(("", port))
        self._server_socket.listen(listen_backlog)
        self._running = False

    def run(self):
        """
        Main server loop.

        Accepts connections sequentially until SIGTERM is received.
        On SIGTERM, closes the listening socket to unblock accept(),
        drains the loop, and finally calls logging.shutdown().
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

    def __handle_client_connection(self, client_sock):
        """
        Handles a single connection.

        Reads one NEW_BETS message. If bets are persisted successfully,
        replies BETS_RECV_SUCCESS and logs:
          action: apuesta_recibida | result: success | cantidad: <amount>
        Also logs one 'apuesta_almacenada' per bet. On any protocol/storage
        error, replies BETS_RECV_FAIL (best effort) and logs 'fail'. Closes
        the socket before returning.
        """
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
                try:
                    service.store_bets(msg.bets)
                    for b in msg.bets:
                        logging.info(
                            "action: apuesta_almacenada | result: success | dni: %s | numero: %s",
                            b.document,
                            b.number,
                        )
                except OSError as e:
                    logging.error("action: process_bets | result: fail | error: %s", e)
                    protocol.BetsRecvFail().write_to(client_sock)
                    logging.error(
                        "action: apuesta_recibida | result: fail | cantidad: %i",
                        getattr(msg, "amount", 0),
                    )
                    continue

                protocol.BetsRecvSuccess().write_to(client_sock)
                logging.info(
                    "action: apuesta_recibida | result: success | cantidad: %i",
                    msg.amount,
                )

            except protocol.ProtocolError as e:
                try:
                    protocol.BetsRecvFail().write_to(client_sock)
                except Exception as e1:
                    logging.error("action: send_message | result: fail | error: %s", e1)
                logging.error("action: apuesta_recibida | result: fail")
                logging.error("action: receive_message | result: fail | error: %s", e)
            except EOFError:
                client_sock.close()
                break
            except OSError as e:
                logging.error("action: send_message | result: fail | error: %s", e)
                break

        client_sock.close()

    def __accept_new_connection(self):
        """
        Accept new connections.

        Blocks until a client connects.
        Logs the client IP and returns the accepted socket.
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
