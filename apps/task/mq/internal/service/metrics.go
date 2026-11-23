package service

import (
	"context"
	"time"

	"github.com/chanmineom/easy-chat/apps/im/metrics"
	"github.com/chanmineom/easy-chat/apps/im/repository"
	"github.com/zeromicro/go-zero/core/logx"
)

type MetricsService struct {
	store  *repository.MongoStore
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
}

func NewMetricsService(store *repository.MongoStore) *MetricsService {
	ctx, cancel := context.WithCancel(context.Background())

	return &MetricsService{
		store:  store,
		ctx:    ctx,
		cancel: cancel,
		done:   make(chan struct{}),
	}
}

func (s *MetricsService) Start() {
	defer close(s.done)

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		s.collect()

		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *MetricsService) Stop() {
	s.cancel()
	<-s.done
}

func (s *MetricsService) collect() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	count, err := s.store.CountPendingOutbox(ctx)
	if err != nil {
		logx.WithContext(ctx).Errorf(
			"count pending outbox: %v",
			err,
		)
		return
	}

	metrics.OutboxPending.Set(float64(count), "im")
}
