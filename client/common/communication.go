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
const FinishedOpCode byte = 3
const RequestWinnersOpCode byte = 4
const WinnersOpCode byte = 5

type ProtocolError struct {
	Msg    string
	Opcode byte
}

func (e *ProtocolError) Error() string {
	return fmt.Sprintf("protocol error: %s (opcode=%d)", e.Msg, e.Opcode)
}

type Message interface {
	GetOpCode() byte
	GetLength() int32
}

type Writeable interface {
	WriteTo(out io.Writer) (int32, error)
}

type Finished struct {
	AgencyId int32
}

func (msg *Finished) GetOpCode() byte {
	return FinishedOpCode
}

func (msg *Finished) GetLength() int32 {
	return 4
}

func (msg *Finished) WriteTo(out io.Writer) (int32, error) {
	if err := binary.Write(out, binary.LittleEndian, msg.GetOpCode()); err != nil {
		return 0, err
	}
	if err := binary.Write(out, binary.LittleEndian, msg.GetLength()); err != nil {
		return 0, err
	}
	if err := binary.Write(out, binary.LittleEndian, msg.AgencyId); err != nil {
		return 0, err
	}
	return 5 + msg.GetLength(), nil
}

type RequestWinners struct {
	AgencyId int32
}

func (msg *RequestWinners) GetOpCode() byte {
	return RequestWinnersOpCode
}

func (msg *RequestWinners) GetLength() int32 {
	return 4
}

func (msg *RequestWinners) WriteTo(out io.Writer) (int32, error) {
	if err := binary.Write(out, binary.LittleEndian, msg.GetOpCode()); err != nil {
		return 0, err
	}
	if err := binary.Write(out, binary.LittleEndian, msg.GetLength()); err != nil {
		return 0, err
	}
	if err := binary.Write(out, binary.LittleEndian, msg.AgencyId); err != nil {
		return 0, err
	}
	return 5 + msg.GetLength(), nil
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

// Serializes the bet and adds it to the writer, incrementing the betsCounter.
// If the full NewBets package would exceed 8kB or the amount of bets would exceed the batchLimit, the bet is not
// added to the body, instead the full NewBets package is built (adding the opcode, body length, and `betsCounter`,
// that represents the amount of bets) and written to finalOutput. Finally, the body is empty and the bet is added,
// reseting the betsCounter to 1.
// Returns the error if some i/o operation failed.
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

type Readable interface {
	readFrom(reader *bufio.Reader) error
	Message
}

type BetsRecvSuccess struct{}

func (msg *BetsRecvSuccess) GetOpCode() byte {
	return BetsRecvSuccessOpCode
}

func (msg *BetsRecvSuccess) GetLength() int32 {
	return 0
}

func (msg *BetsRecvSuccess) readFrom(reader *bufio.Reader) error {
	var length int32
	if err := binary.Read(reader, binary.LittleEndian, &length); err != nil {
		return err
	}
	if length != msg.GetLength() {
		return &ProtocolError{"invalid body length", BetsRecvSuccessOpCode}
	}
	return nil
}

type BetsRecvFail struct{}

func (msg *BetsRecvFail) GetOpCode() byte {
	return BetsRecvFailOpCode
}

func (msg *BetsRecvFail) GetLength() int32 {
	return 0
}

func (msg *BetsRecvFail) readFrom(reader *bufio.Reader) error {
	var length int32
	if err := binary.Read(reader, binary.LittleEndian, &length); err != nil {
		return err
	}
	if length != msg.GetLength() {
		return &ProtocolError{"invalid body length", BetsRecvFailOpCode}
	}
	return nil
}

type Winners struct {
	List []string
}

func (msg *Winners) GetOpCode() byte {
	return WinnersOpCode
}

func (msg *Winners) GetLength() int32 {
	var totalLen int32 = 4
	for _, doc := range msg.List {
		totalLen += 4 + int32(len(doc))
	}
	return totalLen
}

func (msg *Winners) readFrom(reader *bufio.Reader) error {
	var remaining int32
	if err := binary.Read(reader, binary.LittleEndian, &remaining); err != nil {
		return err
	}
	if remaining < 4 {
		return &ProtocolError{"invalid body length", msg.GetOpCode()}
	}
	var nWinners int32
	if err := binary.Read(reader, binary.LittleEndian, &nWinners); err != nil {
		return err
	}
	if nWinners < 0 {
		return &ProtocolError{"invalid body", msg.GetOpCode()}
	}
	remaining -= 4
	for i := int32(0); i < nWinners; i++ {
		if remaining < 4 {
			return &ProtocolError{"invalid body length", msg.GetOpCode()}
		}
		var strLen int32
		if err := binary.Read(reader, binary.LittleEndian, &strLen); err != nil {
			return err
		}
		if strLen < 0 {
			return &ProtocolError{"invalid body", msg.GetOpCode()}
		}
		remaining -= 4
		if remaining < strLen {
			return &ProtocolError{"invalid body length", msg.GetOpCode()}
		}
		buf := make([]byte, strLen)
		if _, err := io.ReadFull(reader, buf); err != nil {
			return err
		}
		remaining -= strLen
		if remaining != 0 {
			return &ProtocolError{"invalid body length", msg.GetOpCode()}
		}

		msg.List = append(msg.List, string(buf))
	}
	return nil
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
			err := msg.readFrom(reader)
			return &msg, err
		}
	case BetsRecvFailOpCode:
		{
			var msg BetsRecvFail
			err := msg.readFrom(reader)
			return &msg, err
		}
	case WinnersOpCode:
		{
			var msg Winners
			err := msg.readFrom(reader)
			return &msg, err
		}
	default:
		return nil, &ProtocolError{"invalid opcode", opcode}
	}
}
