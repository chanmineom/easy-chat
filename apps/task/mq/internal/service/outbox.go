package service

import (
	"context"
	"errors"

	"github.com/chanmineom/easy-chat/apps/im/outbox"
	"github.com/zeromicro/go-zero/core/logx"
)

type OutboxService struct {
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
	worker *outbox.Worker
}

func NewOutboxService(worker *outbox.Worker) *OutboxService {
	ctx, cancel := context.WithCancel(context.Background())
	return &OutboxService{
		ctx:    ctx,
		cancel: cancel,
		done:   make(chan struct{}),
		worker: worker,
	}
}

func (s *OutboxService) Start() {
	defer close(s.done)

	if err := s.worker.Run(s.ctx); err != nil && !errors.Is(err, context.Canceled) {
		logx.Errorf("outbox service stopped unexpectedly: %v", err)
	}
}

func (s *OutboxService) Stop() {
	s.cancel()
	<-s.done
}
