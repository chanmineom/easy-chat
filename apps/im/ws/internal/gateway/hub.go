package gateway

import (
	"errors"
	"sync"

	"github.com/chanmineom/easy-chat/apps/im/metrics"
	"github.com/gorilla/websocket"
)

var (
	ErrConnectionLimitExceeded     = errors.New("websocket connection limit exceeded")
	ErrUserConnectionLimitExceeded = errors.New("websocket user connection limit exceeded")
)

type Hub struct {
	mu     sync.RWMutex
	nodeID string
	users  map[string]map[*Client]struct{}
}

func NewHub(nodeID string) *Hub {
	return &Hub{
		nodeID: nodeID,
		users:  make(map[string]map[*Client]struct{}),
	}
}

func (h *Hub) CheckCapacity(userID string, maxConnections int, maxPerUser int) error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	total := 0
	for _, clients := range h.users {
		total += len(clients)
	}

	if maxConnections > 0 && total > maxConnections {
		return ErrConnectionLimitExceeded
	}

	if maxPerUser > 0 && len(h.users[userID]) >= maxPerUser {
		return ErrUserConnectionLimitExceeded
	}

	return nil
}

func (h *Hub) Add(client *Client, maxConnections int, maxPerUser int) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	total := 0
	for _, clients := range h.users {
		total += len(clients)
	}

	if maxConnections > 0 && total > maxConnections {
		return ErrConnectionLimitExceeded
	}

	current := h.users[client.UserID]
	if maxPerUser > 0 && len(current) >= maxPerUser {
		return ErrUserConnectionLimitExceeded
	}

	// if h.users[client.UserID] == nil {
	// 	h.users[client.UserID] = make(map[*Client]struct{})
	// }

	// h.users[client.UserID][client] = struct{}{}

	if current == nil {
		current = make(map[*Client]struct{})
		h.users[client.UserID] = current
	}

	current[client] = struct{}{}
	metrics.WSActiveConnections.Inc(h.nodeID)
	return nil
}

func (h *Hub) Remove(client *Client) {
	if client == nil {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	clients := h.users[client.UserID]
	if clients == nil {
		return
	}

	if _, exists := clients[client]; !exists {
		return
	}

	delete(clients, client)
	metrics.WSActiveConnections.Dec(h.nodeID)

	if len(clients) == 0 {
		delete(h.users, client.UserID)
	}
}

func (h *Hub) Connections(userID string) []*Client {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients := h.users[userID]
	result := make([]*Client, 0, len(clients))

	for client := range clients {
		result = append(result, client)
	}

	return result
}

func (h *Hub) ConnectionCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var count int
	for _, clients := range h.users {
		count += len(clients)
	}

	return count
}

func (h *Hub) SendToUsers(userIDs []string, envelope Envelope) (onlineUsers int, deliveredConnections int) {
	seenUsers := make(map[string]struct{}, len(userIDs))

	for _, userID := range userIDs {
		if userID == "" {
			continue
		}
		if _, exists := seenUsers[userID]; exists {
			continue
		}
		seenUsers[userID] = struct{}{}

		clients := h.Connections(userID)
		if len(clients) > 0 {
			onlineUsers++
		}

		for _, client := range clients {
			if err := client.Send(envelope); err != nil {
				client.Close()
				continue
			}
			deliveredConnections++
		}
	}

	return onlineUsers, deliveredConnections
}

func (h *Hub) CloseAll(reason string) {
	h.mu.Lock()

	clients := make([]*Client, 0)
	for _, userClients := range h.users {
		for client := range userClients {
			clients = append(clients, client)
		}
	}

	h.users = make(map[string]map[*Client]struct{})
	h.mu.Unlock()

	for _, client := range clients {
		client.CloseWithCode(
			websocket.CloseGoingAway,
			reason,
		)
	}

	metrics.WSActiveConnections.Set(0, h.nodeID)
}
