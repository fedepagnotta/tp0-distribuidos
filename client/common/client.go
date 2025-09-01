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
	"syscall"
	"time"

	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("log")

// ClientConfig holds the runtime configuration for a client instance.
// - ID: agency identifier used inside each bet payload.
// - ServerAddress: TCP address of the server (host:port).
// - BetsFilePath: path to the CSV file with bets for this agency.
// - BatchLimit: maximum number of bets per batch (upper-bounded also by 8KB framing on the wire).
type ClientConfig struct {
	ID            string
	ServerAddress string
	BetsFilePath  string
	BatchLimit    int32
}

// Client encapsulates the client application state: configuration and the
// currently open TCP connection (if any).
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

// processNextBet reads exactly one CSV record from betsReader, builds the
// corresponding map[string]string (including the agency ID), and attempts to
// add it to the current batch buffer. If adding the bet would exceed either the
// 8KB maximum message size or the configured BatchLimit, the current batch is
// flushed to the server and a new batch is started with this bet.
// Returns io.EOF when the CSV is exhausted, or any I/O / serialization error.
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

// buildAndSendBatches streams the CSV file, building batches into batchBuff and
// flushing them to the server connection as needed. On context cancellation, it
// flushes any partially built batch and returns context.Canceled.
// On normal EOF, it flushes the final batch (if any) and returns nil.
// Any error serializing or writing a batch is returned immediately.
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

// createClientSocket dials the server and stores the resulting TCP connection
// in the client. On failure, it logs at Critical level and returns the error.
// The caller is responsible for closing the connection.
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

// SendBets orchestrates the batch sending workflow end-to-end:
//  1. Opens the CSV, dials the server, and starts a goroutine that builds and
//     flushes batches to the server until EOF or context cancellation.
//  2. After all batches are sent, half-closes the socket (CloseWrite) to signal
//     the end of the request stream.
//  3. Reads server responses until EOF (or error). If the last message indicates
//     failure, or a non-EOF read error occurs, logs a fail; otherwise logs success.
//  4. Supports graceful shutdown: on SIGTERM, cancels the context, forces a read
//     deadline to unblock the reader goroutine, and exits cleanly.
//
// Short-writes are avoided by using FlushBatch/AddBetWithFlush implementations
// that rely on io.Copy / send-all semantics. Short-reads are avoided by reading
// framed messages with ReadMessage, which consumes exact sizes per frame.
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

	if err = <-writeDone; err != nil && !errors.Is(err, context.Canceled) {
		log.Errorf("action: send_bets | result: fail | error: %v", err)
		return
	}

	if tcp, ok := c.conn.(*net.TCPConn); ok {
		_ = tcp.CloseWrite()
	}

	reader := bufio.NewReader(c.conn)
	readDone := make(chan struct{})
	var msg Readable
	var rerr error
	go func() {
		for {
			msg, rerr = ReadMessage(reader)
			if rerr != nil {
				break
			}
		}
		close(readDone)
	}()

	select {
	case <-ctx.Done():
		_ = c.conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		<-readDone
		return
	case <-readDone:
		if (rerr != nil && !errors.Is(rerr, io.EOF)) || (msg != nil && msg.GetOpCode() == BetsRecvFailOpCode) {
			log.Error("action: bets_enviadas | result: fail")
			return
		}
		log.Info("action: bets_enviadas | result: success")
	}
}
