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

type Writeable interface {
	// writes contents to out following the package format (opcode, length, body)
	// returns the total length of the body, or error if the write failed
	WriteTo(out io.Writer) (int, error)
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

func writeMultiStringMap(buff *bytes.Buffer, body []map[string]string) error {
	for _, m := range body {
		if err := writeStringMap(buff, m); err != nil {
			return err
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

func (msg *NewBets) WriteTo(out io.Writer) (int, error) {
	var buff bytes.Buffer
	if err := buff.WriteByte(NewBetsOpCode); err != nil {
		return 0, err
	}
	var bodyBuff bytes.Buffer
	if err := binary.Write(&bodyBuff, binary.LittleEndian, int32(len(msg.Bets))); err != nil {
		return 0, err
	}
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

// Serializes the bet and adds it to the writer, incrementing the betsCounter.
// If the full NewBets package would exceed 8kB or the amount of bets would exceed the batchLimit, the bet is not
// added to the body, instead the full NewBets package is built (adding the opcode, body length, and `betsCounter`,
// that represents the amount of bets) and written to finalOutput. Finally, the body is empty and the bet is added,
// reseting the betsCounter to 1.
// Returns the error if some i/o operation failed.
func AddBetToBody(bet map[string]string, to *bytes.Buffer, finalOutput io.Writer, betsCounter *int32, batchLimit int32) error {
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
	if err := binary.Write(finalOutput, binary.LittleEndian, NewBetsOpCode); err != nil {
		return err
	}
	if err := binary.Write(finalOutput, binary.LittleEndian, int32(4+to.Len())); err != nil {
		return err
	}
	if err := binary.Write(finalOutput, binary.LittleEndian, *betsCounter); err != nil {
		return err
	}
	if _, err := io.Copy(finalOutput, to); err != nil {
		return err
	}
	to.Reset()
	if err := writeStringMap(to, bet); err != nil {
		return err
	}
	*betsCounter = 1
	return nil
}

type Readable interface {
	readFrom(reader *bufio.Reader) (Readable, error)
	Message
}

type BetsRecvSuccess struct{}

func (msg *BetsRecvSuccess) GetOpCode() byte {
	return BetsRecvSuccessOpCode
}

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

func (msg *BetsRecvFail) GetOpCode() byte {
	return BetsRecvFailOpCode
}

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
