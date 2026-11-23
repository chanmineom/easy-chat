package health

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/zeromicro/go-zero/core/stores/redis"
)

type Readiness struct {
	redisClient *redis.Redis
	tcpTargets  []string
	ready       atomic.Bool
}

func NewReadiness(redisClient *redis.Redis, tcpTargets []string) *Readiness {
	result := &Readiness{
		redisClient: redisClient,
		tcpTargets:  tcpTargets,
	}

	result.ready.Store(true)

	return result
}

func (r *Readiness) MarkNotReady() {
	r.ready.Store(false)
}

func (r *Readiness) Liveness(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
	})
}

func (r *Readiness) Ready(w http.ResponseWriter, req *http.Request) {
	checks := make(map[string]string)

	if !r.ready.Load() {
		checks["service"] = "shutting down"
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"status": "not_ready",
			"checks": checks,
		})

		return

	}
	ctx, cancel := context.WithTimeout(
		req.Context(),
		2*time.Second,
	)
	defer cancel()

	if r.redisClient.PingCtx(ctx) {
		checks["redis"] = "ok"
	} else {
		checks["redis"] = "unavailable"
	}

	for _, target := range r.tcpTargets {
		conn, err := net.DialTimeout(
			"tcp",
			target,
			750*time.Millisecond,
		)
		if err != nil {
			checks[target] = err.Error()
			continue
		}

		checks[target] = "ok"
		_ = conn.Close()
	}

	statusCode := http.StatusOK
	status := "ready"

	for _, value := range checks {
		if value != "ok" {
			statusCode = http.StatusServiceUnavailable
			status = "not_ready"
			break
		}
	}

	writeJSON(w, statusCode, map[string]any{
		"status": status,
		"checks": checks,
	})
}

func writeJSON(w http.ResponseWriter, statusCode int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(value)
}
