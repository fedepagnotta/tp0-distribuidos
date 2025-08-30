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

                try:
                    msg.process()
                except Exception as e:
                    logging.error("action: process_bets | result: fail | error: %s", e)
                    communication.BetsRecvFail().write_to(client_sock)
                    logging.error(
                        "action: apuesta_recibida | result: fail | cantidad: %i",
                        getattr(msg, "amount", 0),
                    )
                    continue

                communication.BetsRecvSuccess().write_to(client_sock)
                logging.info(
                    "action: apuesta_recibida | result: success | cantidad: %i",
                    msg.amount,
                )

            except communication.ProtocolError as e:
                try:
                    communication.BetsRecvFail().write_to(client_sock)
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
