package gateway

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/time/rate"
)

const (
	sendBuffer = 64
	readLimit  = 64 * 1024
	readWait   = 75 * time.Second
	writeWait  = 10 * time.Second
)

var ErrSlowClient = errors.New("websocket client is too slow")

type Client struct {
	conn      *websocket.Conn
	hub       *Hub
	UserID    string
	DeviceID  string
	SessionID string

	send      chan []byte
	done      chan struct{}
	closeOnce sync.Once
	handler   func(*Client, Envelope)
	limiter   *rate.Limiter
}

func NewClient(conn *websocket.Conn, hub *Hub, userID string, deviceID string, handler func(*Client, Envelope), limiter *rate.Limiter) *Client {
	return &Client{
		conn:      conn,
		hub:       hub,
		UserID:    userID,
		DeviceID:  deviceID,
		SessionID: newSessionID(),
		send:      make(chan []byte, sendBuffer),
		done:      make(chan struct{}),
		handler:   handler,
		limiter:   limiter,
	}
}

func (c *Client) Send(value any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}

	select {
	case <-c.done:
		return net.ErrClosed
	case c.send <- payload:
		return nil
	default:
		return ErrSlowClient
	}
}

func (c *Client) Close() {
	c.closeOnce.Do(func() {
		close(c.done)
		_ = c.conn.Close()
	})
}

func (c *Client) ReadLoop() {
	defer func() {
		c.hub.Remove(c)
		c.Close()
	}()

	c.conn.SetReadLimit(readLimit)

	for {
		_ = c.conn.SetReadDeadline(time.Now().Add(readWait))

		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			return
		}

		if !c.limiter.Allow() {
			c.CloseWithPolicy("message rate limit exceeded")
			return
		}

		var message Envelope
		if err := json.Unmarshal(raw, &message); err != nil {
			c.sendError("", "invalid json envelope")
			continue
		}

		switch message.Type {
		case TypePing:
			if err := c.Send(Envelope{
				Type:      TypePong,
				RequestID: message.RequestID,
			}); err != nil {
				c.Close()
				return
			}
		default:
			if c.handler == nil {
				c.sendError(message.RequestID, "unsupported websocket message type")
				continue
			}
			c.handler(c, message)
		}
	}
}

func (c *Client) WriteLoop() {
	defer c.Close()

	for {
		select {
		case <-c.done:
			return
		case payload := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.TextMessage, payload); err != nil {
				return
			}
		}
	}
}

func (c *Client) sendError(requestID, text string) {
	_ = c.Send(Envelope{
		Type:      TypeError,
		RequestID: requestID,
		Error:     text,
	})
}

func (c *Client) CloseWithPolicy(reason string) {
	// deadline := time.Now().Add(writeWait)
	// _ = c.conn.WriteControl(
	// 	websocket.CloseMessage,
	// 	websocket.FormatCloseMessage(
	// 		websocket.ClosePolicyViolation,
	// 		reason,
	// 	),
	// 	deadline,
	// )
	// c.Close()
	
	c.CloseWithCode(
		websocket.ClosePolicyViolation,
		reason,
	)
}

func (c *Client) CloseWithCode(code int, reason string) {
	_ = c.conn.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(code, reason),
		time.Now().Add(writeWait),
	)
	c.Close()
}

func newSessionID() string {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		logx.Errorf("generate websocket session id: %v", err)
		return time.Now().UTC().Format("20060102150405.000000000")
	}

	return hex.EncodeToString(value[:])
}
