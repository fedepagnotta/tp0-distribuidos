import logging
import signal
import socket
import threading

from app import protocol, service


class Server:
    def __init__(self, port, listen_backlog, clients_amount):
        """Initialize listening socket and concurrency primitives.

        - Creates and binds the TCP listening socket.
        - `_stop` is a process-wide shutdown flag (set by SIGTERM).
        - `_finished` is a Barrier with the expected number of clients/agencies;
          it is used to block FINISHED handlers until all are in.
        - `_winners` holds the computed winners grouped by agency.
        - `_raffle_done` is a latch Event set once the raffle is computed.
        - `_raffle_lock` ensures the raffle is computed exactly once.
        - `_storage_lock` serializes access to storage during batch persistence.
        - `_threads` keeps track of per-connection worker threads.
        """
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
        """Main server loop.

        Installs SIGTERM handler, accepts connections until `_stop` is set,
        and spawns one worker thread per client. On shutdown:
        - breaks the accept loop if the listening socket is closed,
        - joins all worker threads,
        - and calls `logging.shutdown()` to flush logs.
        """
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
        """Accept a single client connection.

        Blocks in `accept()` until a client connects, logs the remote IP,
        and returns the connected socket.
        """
        logging.info("action: accept_connections | result: in_progress")
        c, addr = self._server_socket.accept()
        logging.info(f"action: accept_connections | result: success | ip: {addr[0]}")
        return c

    def __handle_client_connection(self, client_sock):
        """Per-connection worker.

        Repeatedly receives framed messages (`protocol.recv_msg`), logs them,
        and delegates handling to `__process_msg`. The loop continues until
        `__process_msg` returns False (connection should close), `_stop` is set,
        EOF is reached, or a socket/protocol error occurs. Always closes the
        client socket on exit.
        """
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
        """Route a decoded message and apply the server-side semantics.

        Returns:
          True  -> keep reading more messages on this connection
          False -> stop the loop and close the connection

        Semantics:
        - NEW_BETS: persist the whole batch under `_storage_lock`. If every bet
          is stored successfully, reply BETS_RECV_SUCCESS and log
          'apuesta_recibida | success | cantidad'. On any exception, reply
          BETS_RECV_FAIL and log 'apuesta_recibida | fail | cantidad'.
        - FINISHED: wait on the `_finished` Barrier. The last thread crossing
          the barrier triggers the raffle (under `_raffle_lock`) if not done.
          Once the raffle is done, send the agency's winners.
        """
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
            self.__send_winners(msg.agency_id, client_sock)
            return False

    def __raffle(self):
        """Compute winners once and signal readiness.

        Calls `service.compute_winners()` (pure domain logic), stores the result
        into `_winners`, logs success, and sets `_raffle_done` so any waiting
        REQUEST_WINNERS handlers can proceed.
        """
        try:
            self._winners = service.compute_winners()
            logging.info("action: sorteo | result: success")
            self._raffle_done.set()
        except Exception as e:
            logging.error("action: sorteo | result: fail | error: %s", e)
            return

    def __send_winners(self, agency_id, sock):
        """Serialize and send a WINNERS response for a given agency.

        Writes the framed message using protocol helpers (which use sendall),
        and logs success or a protocol error if framing fails.
        """
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
        """SIGTERM handler.

        Sets the global stop flag and closes the listening socket to unblock
        `accept()`. Worker threads already running will drain naturally and
        be joined in `run()`.
        """
        self._stop.set()
        self._server_socket.close()
