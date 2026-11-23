package handler

import (
	"github.com/chanmineom/easy-chat/apps/task/mq/internal/config"
	"github.com/chanmineom/easy-chat/apps/task/mq/internal/handler/chatcommand"
	"github.com/chanmineom/easy-chat/apps/task/mq/internal/handler/messagecreated"
	"github.com/chanmineom/easy-chat/apps/task/mq/internal/handler/receiptcommand"
	taskservice "github.com/chanmineom/easy-chat/apps/task/mq/internal/service"
	"github.com/chanmineom/easy-chat/apps/task/mq/internal/svc"
	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/service"
)

type Listener struct {
	config config.Config
	svcCtx *svc.ServiceContext
}

func NewListener(c config.Config, svcCtx *svc.ServiceContext) *Listener {
	return &Listener{
		config: c,
		svcCtx: svcCtx,
	}
}

func (l *Listener) Services() []service.Service {
	chatGroup := l.config.Kafka.ConsumerGroup
	messageGroup := chatGroup + "-message-created"
	receiptGroup := chatGroup + "-receipt-command"

	return []service.Service{
		kq.MustNewQueue(
			l.config.ChatCommandQueue(),
			chatcommand.NewConsumer(
				l.svcCtx.Mongo,
				l.svcCtx.Gateway,
				l.svcCtx.Social,
				l.svcCtx.DeadLetter,
				l.config.Kafka.ChatCommandTopic,
				chatGroup,
			),
		),
		taskservice.NewOutboxService(l.svcCtx.OutboxWorker),
		kq.MustNewQueue(
			l.config.MessageCreatedQueue(),
			messagecreated.NewConsumer(
				l.svcCtx.Mongo,
				l.svcCtx.Gateway,
				l.svcCtx.DeadLetter,
				l.config.Kafka.MessageCreatedTopic,
				messageGroup,
			),
		),
		kq.MustNewQueue(
			l.config.ReceiptCommandQueue(),
			receiptcommand.NewConsumer(
				l.svcCtx.Mongo,
				l.svcCtx.DeadLetter,
				l.config.Kafka.ReceiptCommandTopic,
				receiptGroup,
			),
		),
		taskservice.NewMetricsService(l.svcCtx.Mongo),
	}
}
