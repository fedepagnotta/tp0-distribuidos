import logging
import signal
import socket


class Server:
    def __init__(self, port, listen_backlog):
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
        """
        Read message from a specific client socket and closes the socket

        If a problem arises in the communication with the client, the
        client socket will also be closed
        """
        try:
            rf = client_sock.makefile("rb")
            line = rf.readline(64 * 1024)
            if line == b"":
                raise EOFError("peer closed connection")
            msg = line.rstrip(b"\r\n").decode("utf-8")
            addr = client_sock.getpeername()
            logging.info(
                "action: receive_message | result: success | ip: %s | msg: %s",
                addr[0],
                msg,
            )
            client_sock.sendall((msg + "\n").encode("utf-8"))
        except (UnicodeDecodeError, EOFError, OSError) as e:
            logging.error("action: receive_message | result: fail | error: %s", e)
        finally:
            client_sock.close()

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

    def __stop_running(self, signum, frame):
        self._running = False
        self._server_socket.close()
