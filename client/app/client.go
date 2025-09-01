package app

import (
	"bufio"
	"context"
	"net"
	"os/signal"
	"syscall"
	"time"

	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("log")

// ClientConfig Configuration used by the client
type ClientConfig struct {
	ID            string
	ServerAddress string
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

// createClientSocket establishes a TCP connection to ServerAddress.
// On failure it logs the error and returns it without leaving a live connection behind.
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

// SendBet opens a connection, serializes and sends a single bet (NewBets with 1 entry),
// then waits for the server ack (BETS_RECV_SUCCESS or BETS_RECV_FAIL) and logs the result.
// Supports graceful shutdown for SIGTERM.
func (c *Client) SendBet(name string, lastName string, dni string, birthDate string, number string) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	defer stop()

	select {
	case <-ctx.Done():
		return
	default:
	}

	if err := c.createClientSocket(); err != nil {
		return
	}

	reader := bufio.NewReader(c.conn)

	bets := NewBets{
		Bets: []map[string]string{
			{
				"AGENCIA":    c.config.ID,
				"NOMBRE":     name,
				"APELLIDO":   lastName,
				"DOCUMENTO":  dni,
				"NACIMIENTO": birthDate,
				"NUMERO":     number,
			},
		},
	}
	if _, err := bets.WriteTo(c.conn); err != nil {
		log.Errorf("action: apuesta_enviada | result: fail | dni: %s | numero: %s", dni, number)
		_ = c.conn.Close()
		return
	}

	readDone := make(chan struct{})
	var msg Readable
	var err error
	go func() {
		msg, err = ReadMessage(reader)
		close(readDone)
	}()

	select {
	case <-ctx.Done():
		_ = c.conn.SetReadDeadline(time.Now())
		<-readDone
		c.conn.Close()
		return
	case <-readDone:
		c.conn.Close()
		if err != nil || msg.GetOpCode() == BetsRecvFailOpCode {
			log.Errorf("action: apuesta_enviada | result: fail | dni: %s | numero: %s", dni, number)
			return
		}
		log.Infof("action: apuesta_enviada | result: success | dni: %s | numero: %s", dni, number)
	}
}
