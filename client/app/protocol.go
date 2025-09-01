package app

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

const NewBetsOpCode byte = 0
const BetsRecvSuccessOpCode byte = 1
const BetsRecvFailOpCode byte = 2

type ProtocolError struct {
	Msg    string
	Opcode byte
}

func (e *ProtocolError) Error() string {
	return fmt.Sprintf("protocol error: %s (opcode=%d)", e.Msg, e.Opcode)
}

type Message interface {
	GetOpCode() byte
}

// Writeable messages know how to write themselves to a net.Conn using the protocol framing:
// opcode (1 byte) | length (int32, little-endian) | body (length bytes).
// It returns the total body length written.
type Writeable interface {
	WriteTo(out net.Conn) (int, error)
}

func writeString(buff *bytes.Buffer, s string) error {
	if err := binary.Write(buff, binary.LittleEndian, int32(len(s))); err != nil {
		return err
	}
	_, err := buff.WriteString(s)
	return err
}

func writePair(buff *bytes.Buffer, k string, v string) error {
	if err := writeString(buff, k); err != nil {
		return err
	}
	return writeString(buff, v)
}

// writeMultiStringMap serializes a slice of map[string]string as a sequence of
// [string map], adding the amount of pairs <k><v> at the beginning.
func writeMultiStringMap(buff *bytes.Buffer, body []map[string]string) error {
	if err := binary.Write(buff, binary.LittleEndian, int32(len(body))); err != nil {
		return err
	}
	for _, m := range body {
		if err := binary.Write(buff, binary.LittleEndian, int32(len(m))); err != nil {
			return err
		}
		for k, v := range m {
			if err := writePair(buff, k, v); err != nil {
				return err
			}
		}
	}
	return nil
}

type NewBets struct {
	Bets []map[string]string
}

func (msg *NewBets) GetOpCode() byte {
	return NewBetsOpCode
}

// WriteTo serializes NewBets using the protocol framing. The body is composed of
// an int32 with the number of bets followed by that many [string map] entries.
// Returns the body length written.
func (msg *NewBets) WriteTo(out net.Conn) (int, error) {
	var buff bytes.Buffer
	if err := buff.WriteByte(NewBetsOpCode); err != nil {
		return 0, err
	}
	var bodyBuff bytes.Buffer
	if err := writeMultiStringMap(&bodyBuff, msg.Bets); err != nil {
		return 0, err
	}
	if err := binary.Write(&buff, binary.LittleEndian, int32(bodyBuff.Len())); err != nil {
		return 0, err
	}
	_, err := buff.Write(bodyBuff.Bytes())
	if err != nil {
		return 0, err
	}
	_, err = io.Copy(out, &buff)
	if err != nil {
		return 0, err
	}
	return bodyBuff.Len(), nil
}

// Readable messages know how to read themselves from a reader based in the protocol framing:
// opcode (1 byte) | length (int32, little-endian) | body (length bytes).
// It's also possible to retrieve the message's opcode and body length.
// Returns error if any operation fails.
type Readable interface {
	readFrom(reader *bufio.Reader) error
	Message
}

type BetsRecvSuccess struct{}

func (msg *BetsRecvSuccess) GetOpCode() byte {
	return BetsRecvSuccessOpCode
}

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

type BetsRecvFail struct{}

func (msg *BetsRecvFail) GetOpCode() byte {
	return BetsRecvFailOpCode
}

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

// ReadMessage reads an opcode from the stream and dispatches to the concrete message
// reader accordingly. It returns the parsed message or a transport/protocol error.
func ReadMessage(reader *bufio.Reader) (Readable, error) {
	var opcode byte
	var err error
	if opcode, err = reader.ReadByte(); err != nil {
		return nil, err
	}
	switch opcode {
	case BetsRecvSuccessOpCode:
		var msg BetsRecvSuccess
		if err := msg.readFrom(reader); err != nil {
			return nil, err
		}
		return &msg, nil
	case BetsRecvFailOpCode:
		var msg BetsRecvFail
		if err := msg.readFrom(reader); err != nil {
			return nil, err
		}
		return &msg, nil
	default:
		return nil, &ProtocolError{"invalid opcode", opcode}
	}
}
