import logging
import socket
import struct
from typing import Tuple

from common import utils


class ProtocolError(Exception):
    def __init__(self, message, opcode=None):
        super().__init__(message)
        self.opcode = opcode


class Opcodes:
    NEW_BETS = 0
    BETS_RECV_SUCCESS = 1
    BETS_RECV_FAIL = 2
    FINISHED = 3
    REQUEST_WINNERS = 4
    WINNERS = 5


class NewBets:
    def __init__(self):
        self.bets: list[utils.Bet] = []
        self.opcode: int = Opcodes.NEW_BETS
        self.required = (
            "AGENCIA",
            "NOMBRE",
            "APELLIDO",
            "DOCUMENTO",
            "NACIMIENTO",
            "NUMERO",
        )
        self.amount: int = 0

    def __read_pair(self, sock: socket.socket, remaining: int) -> tuple[str, str, int]:
        (key, remaining) = read_string(sock, remaining, self.opcode)
        (value, remaining) = read_string(sock, remaining, self.opcode)
        return (key, value, remaining)

    def __read_bet(self, sock: socket.socket, remaining: int) -> int:
        curr_bet: dict[str, str] = {}
        (n_pairs, remaining) = read_i32(sock, remaining, self.opcode)
        if n_pairs != 6:
            raise ProtocolError("invalid body", self.opcode)

        for _ in range(n_pairs):
            (k, v, remaining) = self.__read_pair(sock, remaining)
            curr_bet[k] = v

        if [k for k in self.required if k not in curr_bet]:
            raise ProtocolError("invalid body", self.opcode)

        try:
            bet = utils.Bet(
                curr_bet["AGENCIA"],
                curr_bet["NOMBRE"],
                curr_bet["APELLIDO"],
                curr_bet["DOCUMENTO"],
                curr_bet["NACIMIENTO"],
                curr_bet["NUMERO"],
            )
        except (ValueError, TypeError) as e:
            raise ProtocolError("invalid body", self.opcode) from e

        self.bets.append(bet)
        return remaining

    def read_from(self, sock, length: int):
        remaining = length
        try:
            n_bets, remaining = read_i32(sock, remaining, self.opcode)
            self.amount = n_bets
            for _ in range(n_bets):
                remaining = self.__read_bet(sock, remaining)
            if remaining != 0:
                raise ProtocolError(
                    "indicated length doesn't match body length", self.opcode
                )
        except ProtocolError:
            if remaining > 0:
                _ = recv_exactly(sock, remaining)
            raise


class Finished:
    def __init__(self):
        self.opcode = Opcodes.FINISHED
        self.agency_id = None
        self._length = 4

    def read_from(self, sock: socket.socket, length: int):
        if length != self._length:
            raise ProtocolError("invalid length", self.opcode)
        (agency_id, _) = read_i32(sock, length, self.opcode)
        self.agency_id = agency_id


class RequestWinners:
    def __init__(self):
        self.opcode = Opcodes.REQUEST_WINNERS
        self.agency_id = None
        self._length = 4

    def read_from(self, sock: socket.socket, length: int):
        if length != self._length:
            raise ProtocolError("invalid length", self.opcode)
        (agency_id, _) = read_i32(sock, length, self.opcode)
        self.agency_id = agency_id


def recv_exactly(sock: socket.socket, n: int) -> bytes:
    """
    Reads exactly n bytes or throws EOFError if peer closed before finishing.
    """
    if n < 0:
        raise ProtocolError("invalid body")
    data = bytearray(n)
    view = memoryview(data)
    read = 0
    while read < n:
        try:
            nrecv = sock.recv_into(view[read:], n - read)
        except socket.timeout as e:
            raise ProtocolError("recv timeout") from e
        except InterruptedError:
            continue
        except OSError as e:
            raise ProtocolError(f"recv failed: {e}") from e
        if nrecv == 0:
            raise EOFError("peer closed connection")
        read += nrecv
    return bytes(data)


def read_struct(sock: socket.socket, fmt: str) -> Tuple:
    """
    Reads struct with endianness and types defined by fmt.
    """
    size = struct.calcsize(fmt)
    buf = recv_exactly(sock, size)
    return struct.unpack(fmt, buf)


def read_u8(sock: socket.socket) -> int:
    return read_struct(sock, "<B")[0]


def read_i32(sock: socket.socket, remaining: int, opcode: int) -> (int, int):
    if remaining < 4:
        raise ProtocolError("indicated length doesn't match body length", opcode)
    remaining -= 4
    return (read_struct(sock, "<i")[0], remaining)


def read_string(sock: socket.socket, remaining: int, opcode: int) -> (str, int):
    (key_len, remaining) = read_i32(sock, remaining, opcode)
    if key_len <= 0:
        raise ProtocolError("invalid body", opcode)
    if remaining < key_len:
        raise ProtocolError("indicated length doesn't match body length", opcode)
    try:
        s = recv_exactly(sock, key_len).decode("utf-8")
    except UnicodeDecodeError as e:
        raise ProtocolError("invalid body", opcode) from e
    remaining -= key_len
    return (s, remaining)


def recv_msg(sock: socket.socket):
    opcode = read_u8(sock)
    (length, _) = read_i32(sock, 4, -1)
    if length < 0:
        raise ProtocolError("invalid length")
    if opcode == Opcodes.NEW_BETS:
        msg = NewBets()
        msg.read_from(sock, length)
        return msg
    if opcode == Opcodes.FINISHED:
        msg = Finished()
        msg.read_from(sock, length)
        return msg
    if opcode == Opcodes.REQUEST_WINNERS:
        msg = RequestWinners()
        msg.read_from(sock, length)
        return msg
    raise ProtocolError(f"invalid opcode: {opcode}")


def write_struct(sock: socket.socket, fmt: str, *values) -> None:
    data = struct.pack(fmt, *values)
    sock.sendall(data)


def write_u8(sock: socket.socket, value: int) -> None:
    write_struct(sock, "<B", value)


def write_i32(sock: socket.socket, value: int) -> None:
    write_struct(sock, "<i", value)


def write_string(sock: socket.socket, s: str) -> None:
    b = s.encode("utf-8")
    n = len(b)
    write_i32(sock, n)
    sock.sendall(b)


class BetsRecvSuccess:
    def __init__(self):
        self.opcode = Opcodes.BETS_RECV_SUCCESS

    def write_to(self, sock: socket.socket):
        write_u8(sock, self.opcode)
        write_i32(sock, 0)


class BetsRecvFail:
    def __init__(self):
        self.opcode = Opcodes.BETS_RECV_FAIL

    def write_to(self, sock: socket.socket):
        write_u8(sock, self.opcode)
        write_i32(sock, 0)


class Winners:
    def __init__(self, winners: list[str]):
        self.opcode = Opcodes.WINNERS
        self.list = winners

    def write_to(self, sock: socket.socket):
        body_length = 4
        for document in self.list:
            body_length += 4 + len(document)
        write_u8(sock, self.opcode)
        write_i32(sock, body_length)
        write_i32(sock, len(self.list))
        for document in self.list:
            write_string(sock, document)
