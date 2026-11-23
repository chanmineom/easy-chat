package svc

import (
	"context"
	"time"

	"github.com/chanmineom/easy-chat/apps/im/outbox"
	"github.com/chanmineom/easy-chat/apps/im/repository"
	"github.com/chanmineom/easy-chat/apps/im/session"
	"github.com/chanmineom/easy-chat/apps/social/rpc/socialclient"
	"github.com/chanmineom/easy-chat/apps/task/mq/internal/config"
	"github.com/chanmineom/easy-chat/apps/task/mq/internal/deadletter"
	"github.com/chanmineom/easy-chat/apps/task/mq/internal/gatewayport"
	"github.com/chanmineom/easy-chat/apps/task/mq/internal/gatewayrouter"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config       config.Config
	Mongo        *repository.MongoStore
	Publiser     outbox.EventPubliser
	OutboxWorker *outbox.Worker
	Router       *gatewayrouter.Router
	Gateway      gatewayport.Gateway
	Social       socialclient.Social
	DeadLetter   deadletter.Publisher
}

func NewServiceContext(ctx context.Context, c config.Config) (*ServiceContext, error) {
	mongoStore, err := repository.NewMongoStore(ctx, c.Mongo.Url, c.Mongo.Db)
	if err != nil {
		return nil, err
	}

	publisher, err := outbox.NewKafkaPublisher(c.Kafka.Brokers, c.Kafka.MessageCreatedTopic)
	if err != nil {
		_ = mongoStore.Close(context.Background())
		return nil, err
	}

	redisClient := redis.MustNewRedis(c.Redis)
	sessions := session.NewRedisRegistry(
		redisClient,
		45*time.Second,
	)

	router, err := gatewayrouter.New(sessions, c.WsDiscovery.Hosts, c.WsDiscovery.WsRpcKeyPrefix, c.WsDiscovery.App, c.WsDiscovery.Token)
	if err != nil {
		_ = publisher.Close()
		_ = mongoStore.Close(context.Background())
		return nil, err
	}

	deadLetterPublisher, err := deadletter.NewKafkaPublisher(
		c.Kafka.Brokers,
		c.DeadLetter.Topic,
	)
	if err != nil {
		_ = publisher.Close()
		_ = mongoStore.Close(context.Background())
		return nil, err
	}

	return &ServiceContext{
		Config:       c,
		Mongo:        mongoStore,
		Publiser:     publisher,
		OutboxWorker: outbox.NewWorker(mongoStore, publisher),
		Router:       router,
		Gateway:      router,
		Social:       socialclient.NewSocial(zrpc.MustNewClient(c.SocialRpc)),
		DeadLetter:   deadLetterPublisher,
	}, nil
}

func (s *ServiceContext) Close(ctx context.Context) error {
	routerErr := s.Router.Close()
	publiserErr := s.Publiser.Close()
	deadLetterErr := s.DeadLetter.Close()
	mongoErr := s.Mongo.Close(ctx)
	

	if routerErr != nil {
		return routerErr
	}
	if publiserErr != nil {
		return publiserErr
	}
	if deadLetterErr != nil {
		return deadLetterErr
	}
	return mongoErr
}
