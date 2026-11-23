package gateway

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/chanmineom/easy-chat/apps/im/metrics"
	"github.com/chanmineom/easy-chat/apps/im/model"
	"github.com/chanmineom/easy-chat/apps/im/session"
	"github.com/chanmineom/easy-chat/apps/im/ticket"
	"github.com/golang-jwt/jwt/v4"
	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/token"
	"golang.org/x/time/rate"
)

type ConnectHandler struct {
	hub              *Hub
	parser           *token.TokenParser
	secret           string
	upgrader         websocket.Upgrader
	chatPublisher    ChatPublisher
	receiptPublisher ReceiptPublisher
	sessions         session.Registry
	nodeID           string
	rpcAdvertiseAddr string
	tickets          ticket.Registry
	limits           Limits
}

func NewConnectHandler(hub *Hub, secret string, allowedOrigins []string, chatPublisher ChatPublisher, receiptPublisher ReceiptPublisher, sessions session.Registry, nodeID string, rpcAdvertiseAddr string, tickets ticket.Registry, limits Limits) *ConnectHandler {
	origins := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		origins[origin] = struct{}{}
	}

	return &ConnectHandler{
		hub:    hub,
		parser: token.NewTokenParser(),
		secret: secret,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				if origin == "" {
					return true
				}
				if _, ok := origins["*"]; ok {
					return true
				}
				_, ok := origins[origin]
				return ok
			},
		},
		chatPublisher:    chatPublisher,
		receiptPublisher: receiptPublisher,
		sessions:         sessions,
		nodeID:           nodeID,
		rpcAdvertiseAddr: rpcAdvertiseAddr,
		tickets:          tickets,
		limits:           limits.WithDefaults(),
	}
}

func (h *ConnectHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// tok, err := h.parser.ParseToken(r, h.secret, "")
	// if err != nil || !tok.Valid {
	// 	http.Error(w, "unauthorized", http.StatusUnauthorized)
	// 	return
	// }

	// claims, ok := tok.Claims.(jwt.MapClaims)
	// if !ok {
	// 	http.Error(w, "invalid token claims", http.StatusUnauthorized)
	// 	return
	// }

	// userID, ok := claims[IdentityClaim].(string)
	// if !ok || userID == "" {
	// 	http.Error(w, "missing user identity", http.StatusUnauthorized)
	// 	return
	// }

	// deviceID := r.URL.Query().Get("device_id")
	// if deviceID == "" {
	// 	http.Error(w, "device_id is required", http.StatusBadRequest)
	// 	return
	// }

	identity, err := h.authenticate(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	userID := identity.UserID
	deviceID := identity.DeviceID

	if err := h.hub.CheckCapacity(
		userID,
		h.limits.MaxConnections,
		h.limits.MaxConnectionsPerUser,
	); err != nil {
		http.Error(
			w,
			err.Error(),
			http.StatusTooManyRequests,
		)
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logx.Errorf("upgrade websocket user=%s: %v", userID, err)
		return
	}

	limiter := rate.NewLimiter(
		rate.Limit(h.limits.CommandRatePerSecond),
		h.limits.CommandBurst,
	)

	client := NewClient(conn, h.hub, userID, deviceID, h.handleClientMessage, limiter)

	wsSession := session.Session{
		SessionID:   client.SessionID,
		UserID:      client.UserID,
		DeviceID:    client.DeviceID,
		NodeID:      h.nodeID,
		RPCAddr:     h.rpcAdvertiseAddr,
		ConnectedAt: time.Now().UnixMilli(),
	}

	// h.hub.Add(client)
	if err := h.hub.Add(
		client,
		h.limits.MaxConnections,
		h.limits.MaxConnectionsPerUser,
	); err != nil {
		client.CloseWithPolicy(err.Error())
		return
	}

	if err := h.sessions.Register(r.Context(), wsSession); err != nil {
		h.hub.Remove(client)
		client.Close()
		logx.Errorf(
			"register websocket session user=%s: %v",
			client.UserID,
			err,
		)
		return
	}

	defer h.sessions.Unregister(
		context.Background(),
		client.UserID,
		client.SessionID,
	)

	go h.keepSessionAlive(wsSession, client)

	logx.Infof(
		"websocket connected user=%s device=%s session=%s node=%s",
		client.UserID,
		client.DeviceID,
		client.SessionID,
		wsSession.NodeID,
	)

	go client.WriteLoop()

	ready, err := NewDataEnvelope(
		TypeConnectionReady,
		map[string]any{
			"session_id":  client.SessionID,
			"node_id":     h.nodeID,
			"server_time": time.Now().UnixMilli(),
		},
	)
	if err != nil {
		client.Close()
		return
	}

	if err := client.Send(ready); err != nil {
		client.Close()
		return
	}

	client.ReadLoop()
}

func (h *ConnectHandler) handleClientMessage(client *Client, message Envelope) {
	switch message.Type {
	case TypeMessageSend:
		metrics.WSMessagesTotal.Inc(
			h.nodeID,
			"message.send",
			"received",
		)
		h.handleSendMessage(client, message)

	case TypeMessageDelivered:
		metrics.WSMessagesTotal.Inc(
			h.nodeID,
			"message.delivered",
			"received",
		)
		h.handleReceipt(client, message, model.ReceiptDelivered)

	case TypeMessageRead:
		metrics.WSMessagesTotal.Inc(
			h.nodeID,
			"message.read",
			"received",
		)
		h.handleReceipt(client, message, model.ReceiptRead)
	default:
		metrics.WSMessagesTotal.Inc(
			h.nodeID,
			"unknown",
			"rejected",
		)
		client.sendError(message.RequestID, "unsupported websocket message type")
	}
}

func (h *ConnectHandler) keepSessionAlive(wsSession session.Session, client *Client) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-client.done:
			return
		case <-ticker.C:
			found, err := h.sessions.Touch(
				context.Background(),
				wsSession,
			)
			if err != nil {
				logx.Errorf(
					"touch websocket session user=%s: %v",
					client.UserID,
					err,
				)
				continue
			}
			if found {
				continue
			}

			if err := h.sessions.Register(context.Background(), wsSession); err != nil {
				logx.Errorf(
					"restore websocket session user=%s: %v",
					client.UserID,
					err,
				)
			}
		}
	}
}

type connectionIdentity struct {
	UserID   string
	DeviceID string
}

func (h *ConnectHandler) authenticate(r *http.Request) (connectionIdentity, error) {
	ticketValue := r.URL.Query().Get("ticket")
	authorization := r.Header.Get("Authorization")

	if ticketValue != "" && authorization != "" {
		return connectionIdentity{}, errors.New(
			"use either ticket or authorization, not both",
		)
	}

	if ticketValue != "" {
		claim, err := h.tickets.Consume(r.Context(), ticketValue)
		if err != nil {
			return connectionIdentity{}, err
		}

		return connectionIdentity{
			UserID:   claim.UserID,
			DeviceID: claim.DeviceID,
		}, nil
	}

	tok, err := h.parser.ParseToken(r, h.secret, "")
	if err != nil || !tok.Valid {
		return connectionIdentity{}, errors.New("invalid token")
	}

	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return connectionIdentity{}, errors.New("invalid claims")
	}

	userID, ok := claims[IdentityClaim].(string)
	if !ok || userID == "" {
		return connectionIdentity{}, errors.New("missing identity")
	}

	deviceID := r.URL.Query().Get("device_id")
	if deviceID == "" {
		return connectionIdentity{}, errors.New("device_id is required")
	}

	return connectionIdentity{
		UserID:   userID,
		DeviceID: deviceID,
	}, nil
}
