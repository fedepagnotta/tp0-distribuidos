package common

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

type Writeable interface {
	// writes contents to out following the package format (opcode, length, body)
	// returns the total length of the body, or error if the write failed
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

func writeMultiStringMap(buff *bytes.Buffer, body []map[string]string) error {
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

func (b NewBets) WriteTo(out net.Conn) (int, error) {
	var buff bytes.Buffer
	if err := buff.WriteByte(NewBetsOpCode); err != nil {
		return 0, err
	}
	var bodyBuff bytes.Buffer
	if err := binary.Write(&bodyBuff, binary.LittleEndian, int32(len(b.Bets))); err != nil {
		return 0, err
	}
	if err := writeMultiStringMap(&bodyBuff, b.Bets); err != nil {
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

type Readable interface {
	readFrom(reader *bufio.Reader) (Readable, error)
}

type BetsRecvSuccess struct{}

func (msg *BetsRecvSuccess) readFrom(reader *bufio.Reader) (Readable, error) {
	var length int32
	if err := binary.Read(reader, binary.LittleEndian, &length); err != nil {
		return nil, err
	}
	if length != 0 {
		return nil, &ProtocolError{"invalid body length", BetsRecvSuccessOpCode}
	}
	return msg, nil
}

type BetsRecvFail struct{}

func (msg *BetsRecvFail) readFrom(reader *bufio.Reader) (Readable, error) {
	var length int32
	if err := binary.Read(reader, binary.LittleEndian, &length); err != nil {
		return nil, err
	}
	if length != 0 {
		return nil, &ProtocolError{"invalid body length", BetsRecvFailOpCode}
	}
	return msg, nil
}

// reads contents from reader and returns the parsed package
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
			return msg.readFrom(reader)
		}
	case BetsRecvFailOpCode:
		{
			var msg BetsRecvFail
			return msg.readFrom(reader)
		}
	default:
		return nil, &ProtocolError{"invalid opcode", opcode}
	}
}
