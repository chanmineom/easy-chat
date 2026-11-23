package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	gzredis "github.com/zeromicro/go-zero/core/stores/redis"
)

const (
	defaultTTLSeconds = 45

	sessionKeyPrefix = "im:session:"
	userSetPrefix    = "im:user:sessions:"
)

const registerScript = `
redis.call("SET", KEYS[1], ARGV[1], "EX", ARGV[2])
redis.call("SADD", KEYS[2], ARGV[3])
redis.call("EXPIRE", KEYS[2], ARGV[2])
return 1
`

const touchScript = `
if redis.call("EXISTS", KEYS[1]) == 0 then
	return 0
end
redis.call("EXPIRE", KEYS[1], ARGV[1])
redis.call("EXPIRE", KEYS[2], ARGV[1])
return 1
`

const unregisterScript = `
redis.call("DEL", KEYS[1])
redis.call("SREM", KEYS[2], ARGV[1])
if redis.call("SCARD", KEYS[2]) == 0 then
	redis.call("DEL", KEYS[2])
end
return 1
`

type RedisRegistry struct {
	client *gzredis.Redis
	ttl    int
}

func NewRedisRegistry(client *gzredis.Redis, ttl time.Duration) *RedisRegistry {
	seconds := int(ttl.Seconds())
	if seconds <= 0 {
		seconds = defaultTTLSeconds
	}

	return &RedisRegistry{
		client: client,
		ttl:    seconds,
	}
}

func (r *RedisRegistry) TTL() int {
	return r.ttl
}

func (r *RedisRegistry) Register(ctx context.Context, session Session) error {
	// if session.SessionID == "" ||
	// 	session.UserID == "" ||
	// 	session.NodeID == "" ||
	// 	session.RPCAddr == "" {
	// 	return fmt.Errorf("session fields are incomplete")
	// }
	if session.SessionID == "" || session.NodeID == "" {
		return fmt.Errorf("session fields are incomplete")
	}

	payload, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("encode session: %w", err)
	}

	_, err = r.client.EvalCtx(ctx, registerScript, []string{
		sessionKey(session.SessionID),
		userSessionsKey(session.UserID),
	},
		string(payload),
		r.ttl,
		session.SessionID,
	)
	if err != nil {
		return fmt.Errorf("register redis session: %w", err)
	}

	return nil
}

func (r *RedisRegistry) Touch(ctx context.Context, session Session) (bool, error) {
	result, err := r.client.EvalCtx(
		ctx,
		touchScript,
		[]string{
			sessionKey(session.SessionID),
			userSessionsKey(session.UserID),
		},
		r.ttl,
	)
	if err != nil {
		return false, fmt.Errorf("touch redis session: %w", err)
	}

	value, ok := result.(int64)
	return ok && value == 1, nil
}

func (r *RedisRegistry) Unregister(ctx context.Context, userID string, sessionID string) error {
	_, err := r.client.EvalCtx(
		ctx,
		unregisterScript,
		[]string{
			sessionKey(sessionID),
			userSessionsKey(userID),
		},
		sessionID,
	)
	if err != nil {
		return fmt.Errorf("unregister redis session: %w", err)
	}

	return nil
}

func (r *RedisRegistry) Sessions(ctx context.Context, userID string) ([]Session, error) {
	sessionIDs, err := r.client.SmembersCtx(
		ctx,
		userSessionsKey(userID),
	)
	if err != nil {
		return nil, fmt.Errorf("list user sessions: %w", err)
	}

	result := make([]Session, 0, len(sessionIDs))
	for _, sessionID := range sessionIDs {
		value, err := r.client.GetCtx(ctx, sessionKey(sessionID))
		if errors.Is(err, gzredis.Nil) {
			_, _ = r.client.SremCtx(ctx, userSessionsKey(userID), sessionID)
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("get redis session: %w", err)
		}

		var session Session
		if err := json.Unmarshal([]byte(value), &session); err != nil {
			return nil, fmt.Errorf("decode redis session: %w", err)
		}
		if session.UserID != userID {
			continue
		}

		result = append(result, session)
	}

	return result, nil
}

func sessionKey(sessionID string) string {
	return sessionKeyPrefix + sessionID
}

func userSessionsKey(userID string) string {
	return userSetPrefix + userID
}
