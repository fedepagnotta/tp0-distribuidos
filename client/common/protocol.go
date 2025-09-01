package common

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

const NewBetsOpCode byte = 0
const BetsRecvSuccessOpCode byte = 1
const BetsRecvFailOpCode byte = 2

// ProtocolError represents a protocol-layer validation or framing error.
// It includes (optionally) the opcode being processed when the error occurred.
type ProtocolError struct {
	Msg    string
	Opcode byte
}

func (e *ProtocolError) Error() string {
	return fmt.Sprintf("protocol error: %s (opcode=%d)", e.Msg, e.Opcode)
}

// Message is the minimal interface implemented by all protocol messages.
type Message interface {
	// GetOpCode returns the wire opcode for this message.
	GetOpCode() byte
}

// writeString writes a string as a length-prefixed UTF-8 sequence into buff,
// using the wire format: [int32 length][bytes]. It returns any I/O error
// encountered while writing to the buffer.
func writeString(buff *bytes.Buffer, s string) error {
	if err := binary.Write(buff, binary.LittleEndian, int32(len(s))); err != nil {
		return err
	}
	_, err := buff.WriteString(s)
	return err
}

// writePair writes a single (key,value) pair into buff, where each element
// is encoded with writeString (length-prefixed UTF-8).
func writePair(buff *bytes.Buffer, k string, v string) error {
	if err := writeString(buff, k); err != nil {
		return err
	}
	return writeString(buff, v)
}

// writeStringMap writes a [string map] into buff using the wire format:
// [int32 n] followed by n pairs <k><v>, each encoded with writeString.
func writeStringMap(buff *bytes.Buffer, body map[string]string) error {
	if err := binary.Write(buff, binary.LittleEndian, int32(len(body))); err != nil {
		return err
	}
	for k, v := range body {
		if err := writePair(buff, k, v); err != nil {
			return err
		}
	}
	return nil
}

// AddBetWithFlush serializes a single bet and appends it to the in-memory batch
// buffer `to`, incrementing betsCounter. If appending this bet would exceed the
// 8KB message cap (including protocol overhead: opcode + length + count) or the
// configured batchLimit, it first flushes the current batch to `finalOutput`
// using FlushBatch, then starts a new batch with this bet and resets
// betsCounter to 1.
//
// Returns any serialization or write error encountered. On success, either the
// bet is buffered into `to` (and betsCounter++) or a new batch is started with
// this bet (betsCounter=1).
func AddBetWithFlush(bet map[string]string, to *bytes.Buffer, finalOutput io.Writer, betsCounter *int32, batchLimit int32) error {
	var buff bytes.Buffer
	if err := writeStringMap(&buff, bet); err != nil {
		return err
	}
	if to.Len()+buff.Len()+1+4+4 <= 8*1024 && *betsCounter+1 <= batchLimit {
		_, err := io.Copy(to, &buff)
		if err != nil {
			return err
		}
		*betsCounter++
		return nil
	}
	if err := FlushBatch(to, finalOutput, *betsCounter); err != nil {
		return err
	}
	if err := writeStringMap(to, bet); err != nil {
		return err
	}
	*betsCounter = 1
	return nil
}

// FlushBatch writes one complete NEW_BETS frame to `out` using the wire format:
// [opcode][int32 length][int32 betsCount][body], where length = 4 + len(body)
// (the extra 4 accounts for betsCount). The `body` is taken from `batch`.
//
// The function uses io.Copy to avoid short writes for the variable-sized body.
// After a successful flush, the batch buffer is reset. Returns any I/O error.
func FlushBatch(batch *bytes.Buffer, out io.Writer, betsCounter int32) error {
	if err := binary.Write(out, binary.LittleEndian, NewBetsOpCode); err != nil {
		return err
	}
	if err := binary.Write(out, binary.LittleEndian, int32(4+batch.Len())); err != nil {
		return err
	}
	if err := binary.Write(out, binary.LittleEndian, betsCounter); err != nil {
		return err
	}
	if _, err := io.Copy(out, batch); err != nil {
		return err
	}
	batch.Reset()
	return nil
}

// Readable is implemented by inbound messages that know how to parse
// themselves from a framed stream. Implementations must consume exactly
// their body bytes (according to the protocol format) and return nil on
// success, or an error (ProtocolError / I/O) on failure.
type Readable interface {
	readFrom(reader *bufio.Reader) error
	Message
}

// BetsRecvSuccess represents the server's positive acknowledgment for a batch.
// Its body length is always 0 (no payload).
type BetsRecvSuccess struct{}

func (msg *BetsRecvSuccess) GetOpCode() byte {
	return BetsRecvSuccessOpCode
}

// readFrom parses the empty body of BetsRecvSuccess and validates that the
// length field is exactly 0. It returns nil on success, or an error if the
// length is invalid or the underlying read fails.
func (msg *BetsRecvSuccess) readFrom(reader *bufio.Reader) error {
	var length int32
	if err := binary.Read(reader, binary.LittleEndian, &length); err != nil {
		return err
	}
	if length != 0 {
		return &ProtocolError{"invalid body length", BetsRecvSuccessOpCode}
	}
	return nil
}

// BetsRecvFail represents the server's negative acknowledgment for a batch.
// Its body length is always 0 (no payload).
type BetsRecvFail struct{}

func (msg *BetsRecvFail) GetOpCode() byte {
	return BetsRecvFailOpCode
}

// readFrom parses the empty body of BetsRecvFail and validates that the
// length field is exactly 0. It returns nil on success, or an error if the
// length is invalid or the underlying read fails.
func (msg *BetsRecvFail) readFrom(reader *bufio.Reader) error {
	var length int32
	if err := binary.Read(reader, binary.LittleEndian, &length); err != nil {
		return err
	}
	if length != 0 {
		return &ProtocolError{"invalid body length", BetsRecvFailOpCode}
	}
	return nil
}

// ReadMessage reads one framed server response from reader. It first consumes
// the opcode byte, then dispatches to the specific message parser, which must
// validate and consume the entire body. On success it returns the parsed
// message (as a Readable). On failure, it returns a ProtocolError for invalid
// opcodes/lengths or any underlying I/O error (e.g., EOF).
func ReadMessage(reader *bufio.Reader) (Readable, error) {
	var opcode byte
	var err error
	if opcode, err = reader.ReadByte(); err != nil {
		return nil, err
	}
	switch opcode {
	case BetsRecvSuccessOpCode:
		{
			var msg BetsRecvSuccess
			if err := msg.readFrom(reader); err != nil {
				return nil, err
			}
			return &msg, nil
		}
	case BetsRecvFailOpCode:
		{
			var msg BetsRecvFail
			if err := msg.readFrom(reader); err != nil {
				return nil, err
			}
			return &msg, nil
		}
	default:
		return nil, &ProtocolError{"invalid opcode", opcode}
	}
}
