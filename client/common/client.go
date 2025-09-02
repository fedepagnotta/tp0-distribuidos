package common

import (
	"bufio"
	"context"
	"fmt"
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
	LoopAmount    int
	LoopPeriod    time.Duration
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

func writeFull(conn net.Conn, b []byte) error {
	for len(b) > 0 {
		n, err := conn.Write(b)
		if err != nil {
			return err
		}
		b = b[n:]
	}
	return nil
}

// StartClientLoop Send messages to the client until some time threshold is met
func (c *Client) StartClientLoop() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	defer stop()

	// There is an autoincremental msgID to identify every message sent
	// Messages if the message amount threshold has not been surpassed
	for msgID := 1; msgID <= c.config.LoopAmount; msgID++ {
		select {
		case <-ctx.Done():
			return
		default:
		}
		conn, derr := net.Dial("tcp", c.config.ServerAddress)
		if derr != nil {
			log.Criticalf(
				"action: connect | result: fail | client_id: %v | error: %v",
				c.config.ID,
				derr,
			)
			return
		}
		c.conn = conn
		defer conn.Close()

		wMsg := fmt.Sprintf("[CLIENT %v] Message N°%v\n", c.config.ID, msgID)

		if err := writeFull(c.conn, []byte(wMsg)); err != nil {
			log.Errorf("action: send | result: fail | client_id: %v | error: %v", c.config.ID, err)
			return
		}

		readDone := make(chan struct{})
		var msg string
		var err error
		go func() {
			msg, err = bufio.NewReader(c.conn).ReadString('\n')
			close(readDone)
		}()

		timer := time.NewTimer(c.config.LoopPeriod)
		select {
		case <-ctx.Done():
			_ = c.conn.SetReadDeadline(time.Now())
			<-readDone
			timer.Stop()
			c.conn.Close()
			return
		case <-timer.C:
			select {
			case <-ctx.Done():
				_ = c.conn.SetReadDeadline(time.Now())
				<-readDone
				timer.Stop()
				c.conn.Close()
				return
			case <-readDone:
				c.conn.Close()
				if err != nil {
					log.Errorf("action: receive_message | result: fail | client_id: %v | error: %v",
						c.config.ID,
						err,
					)
					return
				}
				log.Infof("action: receive_message | result: success | client_id: %v | msg: %v",
					c.config.ID,
					msg,
				)
			}
		}
	}
	log.Infof("action: loop_finished | result: success | client_id: %v", c.config.ID)
}
