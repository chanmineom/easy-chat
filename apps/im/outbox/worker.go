package outbox

import (
	"context"
	"fmt"
	"time"

	"github.com/chanmineom/easy-chat/apps/im/repository"
	"github.com/zeromicro/go-zero/core/logx"
)

type Worker struct {
	store     *repository.MongoStore
	publisher EventPubliser

	pollInterval time.Duration // 轮询间隔
	lease        time.Duration // 租约时长
	baseBackoff  time.Duration // 基础退避时长
	maxBackoff   time.Duration // 最大退避时长
}

func NewWorker(store *repository.MongoStore, publiser EventPubliser) *Worker {
	return &Worker{
		store:        store,
		publisher:    publiser,
		pollInterval: 500 * time.Millisecond,
		lease:        30 * time.Second,
		baseBackoff:  time.Second,
		maxBackoff:   time.Minute,
	}
}

// PublishNext 至多处理一个事件。当当前没有可用事件时，返回 false。
func (w *Worker) PublishNext(ctx context.Context) (bool, error) {
	now := time.Now().UTC()

	event, found, err := w.store.ClaimNextOutboxEvent(ctx, now, w.lease)
	if err != nil {
		return false, err
	}
	if !found {
		return false, nil
	}

	if err := w.publisher.Publish(ctx, event); err != nil {
		retryDelay := w.retryDelay(event.Attempts)
		retryAt := time.Now().UTC().Add(retryDelay)

		if markErr := w.store.MarkOutboxFailed(ctx, event.EventID, retryAt, err); markErr != nil {
			return true, fmt.Errorf("publish failed: %v; mark failed: %w", err, markErr)
		}

		return true, fmt.Errorf("publish outbox event %s: %w", event.EventID, err)
	}

	if err := w.store.MarkOutboxSent(ctx, event.EventID, time.Now().UTC()); err != nil {
		return true, err
	}

	return true, nil
}

func (w *Worker) Run(ctx context.Context) error {
	for {
		processed, err := w.PublishNext(ctx)
		if err != nil {
			logx.WithContext(ctx).Errorf("outbox worker: %v", err)
		}
		if processed {
			continue
		}

		timer := time.NewTimer(w.pollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func (w *Worker) retryDelay(attempts int) time.Duration {
	if attempts < 1 {
		attempts = 1
	}

	delay := w.baseBackoff
	for i := 1; i < attempts && delay < w.maxBackoff; i++ {
		delay *= 2
	}
	if delay > w.maxBackoff {
		delay = w.maxBackoff
	}

	return delay
}
