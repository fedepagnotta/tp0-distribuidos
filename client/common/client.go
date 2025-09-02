package common

import (
	"bufio"
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"io"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("log")

// ClientConfig Configuration used by the client
type ClientConfig struct {
	ID            string
	ServerAddress string
	BetsFilePath  string
	BatchLimit    int32
}

// Client Entity that encapsulates how
type Client struct {
	config ClientConfig
	conn   net.Conn
}

// NewClient Initializes a new client receiving the configuration
// as a parameter
func NewClient(config ClientConfig) *Client {
	client := &Client{
		config: config,
	}
	return client
}

func (c *Client) processNextBet(betsReader *csv.Reader, batchBuff *bytes.Buffer, betsCounter *int32) error {
	betFields, err := betsReader.Read()
	if err != nil {
		return err
	}
	bet := map[string]string{
		"AGENCIA":    c.config.ID,
		"NOMBRE":     betFields[0],
		"APELLIDO":   betFields[1],
		"DOCUMENTO":  betFields[2],
		"NACIMIENTO": betFields[3],
		"NUMERO":     betFields[4],
	}
	if err := AddBetWithFlush(bet, batchBuff, c.conn, betsCounter, c.config.BatchLimit); err != nil {
		return err
	}
	return nil
}

func (c *Client) buildAndSendBatches(ctx context.Context, betsReader *csv.Reader) error {
	var batchBuff bytes.Buffer
	var betsCounter int32 = 0
	for {
		select {
		case <-ctx.Done():
			if betsCounter > 0 {
				if err := FlushBatch(&batchBuff, c.conn, betsCounter); err != nil {
					return err
				}
				betsCounter = 0
			}
			return ctx.Err()
		default:
		}
		if err := c.processNextBet(betsReader, &batchBuff, &betsCounter); err != nil {
			if errors.Is(err, io.EOF) {
				if betsCounter > 0 {
					if err := FlushBatch(&batchBuff, c.conn, betsCounter); err != nil {
						return err
					}
				}
				break
			}
			return err
		}
	}
	return nil
}

// CreateClientSocket Initializes client socket. In case of
// failure, error is printed in stdout/stderr and exit 1
// is returned
func (c *Client) createClientSocket() error {
	conn, err := net.Dial("tcp", c.config.ServerAddress)
	if err != nil {
		log.Criticalf(
			"action: connect | result: fail | client_id: %v | error: %v",
			c.config.ID,
			err,
		)
		return err
	}
	c.conn = conn
	return nil
}

// SendBet Sends bet with the received parameters to the server, and waits for a response (success or fail)
func (c *Client) SendBets() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	defer stop()

	betsFile, err := os.Open(c.config.BetsFilePath)
	if err != nil {
		log.Criticalf("action: read_bets | result: fail | error: %v", err)
		return
	}
	defer betsFile.Close()

	betsReader := csv.NewReader(betsFile)
	betsReader.Comma = ','
	betsReader.FieldsPerRecord = 5

	if err := c.createClientSocket(); err != nil {
		return
	}
	defer c.conn.Close()

	writeDone := make(chan error, 1)
	go func() {
		writeDone <- c.buildAndSendBatches(ctx, betsReader)
	}()

	conn := c.conn
	readDone := make(chan struct{})
	readResponse(conn, readDone)

	if err = <-writeDone; err != nil && !errors.Is(err, context.Canceled) {
		log.Errorf("action: send_bets | result: fail | error: %v", err)
		return
	}

	if err == nil {
		c.sendFinishedAndAskForWinners(ctx)
	}

	select {
	case <-ctx.Done():
		_ = c.conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		<-readDone
		return
	case <-readDone:
		if tcp, ok := c.conn.(*net.TCPConn); ok {
			_ = tcp.CloseWrite()
		}
	}
}

func readResponse(conn net.Conn, readDone chan struct{}) {
	reader := bufio.NewReader(conn)
	go func() {
		for {
			msg, err := ReadMessage(reader)
			if err != nil {
				if !errors.Is(err, io.EOF) {
					log.Errorf("action: leer_respuesta | result: fail | err: %v", err)
				}
				break
			}
			switch msg.GetOpCode() {
			case BetsRecvSuccessOpCode:
				log.Info("action: bets_enviadas | result: success")
			case BetsRecvFailOpCode:
				log.Error("action: bets_enviadas | result: fail")
			}
		}
		close(readDone)
	}()
}

func (c *Client) sendFinishedAndAskForWinners(ctx context.Context) {
	agencyId, err := strconv.Atoi(c.config.ID)
	if err != nil {
		log.Errorf("action: send_finished | result: fail | error: %v", err)
		return
	}

	finishedMsg := Finished{int32(agencyId)}
	if _, err := finishedMsg.WriteTo(c.conn); err != nil {
		log.Errorf("action: send_finished | result: fail | error: %v", err)
		return
	}

	log.Infof("action: send_finished | result: success | agencyId: %d", int32(agencyId))

	for {
		if err := c.createClientSocket(); err != nil {
			return
		}
		conn := c.conn
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))

		reqMsg := RequestWinners{int32(agencyId)}
		if _, err := reqMsg.WriteTo(conn); err != nil {
			conn.Close()
			log.Errorf("action: send_request_winners | result: fail | error: %v", err)
			return
		}
		log.Infof("action: send_request_winners | result: success | agencyId: %d", int32(agencyId))

		reader := bufio.NewReader(conn)
		msg, err := ReadMessage(reader)
		conn.Close()

		if err == nil && msg.GetOpCode() == WinnersOpCode {
			log.Infof("action: consulta_ganadores | result: success | cant_ganadores: %d",
				len(msg.(*Winners).List))
			break
		}
		if errors.Is(err, io.EOF) {
			select {
			case <-ctx.Done():
				return
			case <-time.After(500 * time.Millisecond):
			}
			continue
		}
		log.Errorf("action: leer_respuesta | result: fail | err: %v", err)
	}
}
