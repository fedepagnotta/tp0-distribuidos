package common

import (
	"bytes"
	"encoding/binary"
	"io"
	"net"
)

type Writeable interface {
	// writes contents to out following the package format (opcode, length, body)
	// returns the total length of the body, or error if the write failed
	WriteTo(out net.Conn) (int, error)
}

type NewBets struct {
	Bets []map[string]string
}

const NewBetsOpCode byte = 0

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

func writeMultiStringMap(buff *bytes.Buffer, bets NewBets) error {
	for _, bet := range bets.Bets {
		if err := binary.Write(buff, binary.LittleEndian, int32(len(bet))); err != nil {
			return err
		}
		for k, v := range bet {
			if err := writePair(buff, k, v); err != nil {
				return err
			}
		}
	}
	return nil
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
	if err := writeMultiStringMap(&bodyBuff, b); err != nil {
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
