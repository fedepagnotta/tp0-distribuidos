import socket
import struct
from typing import Tuple


class ProtocolError(Exception):
    def __init__(self, message, opcode=None):
        super().__init__(message)
        self.opcode = opcode


class Opcodes:
    NEW_BETS = 0
    BETS_RECV_SUCCESS = 1
    BETS_RECV_FAIL = 2


class RawBet:
    def __init__(
        self,
        agency: str,
        first_name: str,
        last_name: str,
        document: str,
        birthdate: str,
        number: str,
    ):
        self.agency = agency
        self.first_name = first_name
        self.last_name = last_name
        self.document = document
        self.birthdate = birthdate
        self.number = number


class NewBets:
    def __init__(self):
        self.bets: list[RawBet] = []
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
        for _ in range(0, n_pairs):
            (k, v, remaining) = self.__read_pair(sock, remaining)
            curr_bet[k] = v
        if [k for k in self.required if k not in curr_bet]:
            raise ProtocolError("invalid body", self.opcode)
        self.bets.append(
            RawBet(
                curr_bet["AGENCIA"],
                curr_bet["NOMBRE"],
                curr_bet["APELLIDO"],
                curr_bet["DOCUMENTO"],
                curr_bet["NACIMIENTO"],
                curr_bet["NUMERO"],
            )
        )
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


def recv_exactly(sock: socket.socket, n: int) -> bytes:
    """
    Reads exactly n bytes from the socket (retrying as needed) or raises
    EOFError if the peer closes first.
    Converts timeouts/OS errors into ProtocolError. Prevents short reads.
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
    Reads the exact number of bytes required by 'fmt' (struct.calcsize),
    then unpacks them using little-endian formats.
    """
    size = struct.calcsize(fmt)
    buf = recv_exactly(sock, size)
    return struct.unpack(fmt, buf)


def read_u8(sock: socket.socket) -> int:
    return recv_exactly(sock, 1)[0]


def read_i32(sock: socket.socket, remaining: int, opcode: int) -> tuple[int, int]:
    if remaining < 4:
        raise ProtocolError("indicated length doesn't match body length", opcode)
    remaining -= 4
    val = int.from_bytes(recv_exactly(sock, 4), byteorder="little", signed=True)
    return val, remaining


def read_string(sock: socket.socket, remaining: int, opcode: int) -> (str, int):
    """
    Reads a protocol [string]: int32 length (validated) followed
    by UTF-8 bytes.
    Returns the decoded string and the updated 'remaining' counter.
    """
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
    """
    Reads opcode (u8) and length (i32 LE), then dispatches to the appropriate
    message class (currently only NEW_BETS). Validates 'length' and raises
    ProtocolError for invalid opcodes.
    """
    opcode = read_u8(sock)
    (length, _) = read_i32(sock, 4, -1)
    if length < 0:
        raise ProtocolError("invalid length")
    if opcode == Opcodes.NEW_BETS:
        new_bets = NewBets()
        new_bets.read_from(sock, length)
        return new_bets
    else:
        raise ProtocolError(f"invalid opcode: {opcode}")


def write_u8(sock, value: int) -> None:
    if not 0 <= value <= 255:
        raise ValueError("u8 out of range")
    sock.sendall(bytes([value]))


def write_i32(sock: socket.socket, value: int) -> None:
    sock.sendall(int(value).to_bytes(4, byteorder="little", signed=True))


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
