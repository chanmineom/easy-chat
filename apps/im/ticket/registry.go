package ticket

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	gzredis "github.com/zeromicro/go-zero/core/stores/redis"
)

const ticketKeyPrefix = "im:ws:ticket:"

var ErrTicketInvalid = errors.New("websocket ticket is invalid")

type Claim struct {
	UserID    string    `json:"user_id"`
	DeviceID  string    `json:"device_id"`
	ExpiresAt time.Time `json:"expires_at"`
}

type Issued struct {
	Ticket    string
	ExpiresAt time.Time
}

type Registry interface {
	Issue(ctx context.Context, userID string, deviceID string) (Issued, error)
	Consume(ctx context.Context, ticket string) (Claim, error)
}

type RedisRegistry struct {
	client *gzredis.Redis
	ttl    time.Duration
}

func NewRedisRegistry(client *gzredis.Redis, ttl time.Duration) *RedisRegistry {
	if ttl <= 0 {
		ttl = 30 * time.Second
	}

	return &RedisRegistry{
		client: client,
		ttl:    ttl,
	}
}

func (r *RedisRegistry) Issue(ctx context.Context, userID string, deviceID string) (Issued, error) {
	userID = strings.TrimSpace(userID)
	deviceID = strings.TrimSpace(deviceID)

	if userID == "" || deviceID == "" {
		return Issued{}, fmt.Errorf("user and device are required")
	}

	value, err := randomTicket()
	if err != nil {
		return Issued{}, fmt.Errorf("generate ticket: %w", err)
	}

	expiresAt := time.Now().UTC().Add(r.ttl)
	payload, err := json.Marshal(Claim{
		UserID:    userID,
		DeviceID:  deviceID,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return Issued{}, fmt.Errorf("encode ticket: %w", err)
	}

	err = r.client.SetexCtx(
		ctx,
		ticketKey(value),
		string(payload),
		int(r.ttl.Seconds()),
	)
	if err != nil {
		return Issued{}, fmt.Errorf("store ticket: %w", err)
	}

	return Issued{
		Ticket:    value,
		ExpiresAt: expiresAt,
	}, nil
}

const consumeScript = `
local value = redis.call("GET", KEYS[1])
if not value then
	return nil
end
redis.call("DEL", KEYS[1])
return value
`

func (r *RedisRegistry) Consume(ctx context.Context, value string) (Claim, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return Claim{}, ErrTicketInvalid
	}

	result, err := r.client.EvalCtx(
		ctx,
		consumeScript,
		[]string{ticketKey(value)},
	)
	if err != nil {
		return Claim{}, fmt.Errorf("consume ticket: %w", err)
	}
	if result == nil {
		return Claim{}, ErrTicketInvalid
	}

	payload, ok := result.(string)
	if !ok {
		return Claim{}, ErrTicketInvalid
	}

	var claim Claim
	if err := json.Unmarshal([]byte(payload), &claim); err != nil {
		return Claim{}, ErrTicketInvalid
	}
	if claim.UserID == "" || claim.DeviceID == "" || time.Now().UTC().After(claim.ExpiresAt) {
		return Claim{}, ErrTicketInvalid
	}

	return claim, nil
}

func randomTicket() (string, error) {
	var value [32]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(value[:]), nil
}

func ticketKey(value string) string {
	return ticketKeyPrefix + value
}
