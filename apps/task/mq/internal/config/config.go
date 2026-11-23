package config

import (
	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	service.ServiceConf

	Mongo struct {
		Url string
		Db  string
	}

	Redis redis.RedisConf

	Kafka struct {
		Brokers             []string
		ChatCommandTopic    string
		MessageCreatedTopic string
		ReceiptCommandTopic string
		ConsumerGroup       string
	}

	SocialRpc zrpc.RpcClientConf

	WsDiscovery struct {
		Hosts          []string
		WsRpcKeyPrefix string
		App            string
		Token          string
	}

	DeadLetter struct {
		Topic string
	}
}

func (c Config) ChatCommandQueue() kq.KqConf {
	return kq.KqConf{
		Brokers:       c.Kafka.Brokers,
		Group:         c.Kafka.ConsumerGroup,
		Topic:         c.Kafka.ChatCommandTopic,
		Offset:        "last",
		Conns:         1,
		Consumers:     1,
		Processors:    1,
		CommitInOrder: true,
		ForceCommit:   false,
	}
}

func (c Config) MessageCreatedQueue() kq.KqConf {
	return kq.KqConf{
		Brokers:       c.Kafka.Brokers,
		Group:         c.Kafka.ConsumerGroup + "-message-created",
		Topic:         c.Kafka.MessageCreatedTopic,
		Offset:        "last",
		Conns:         1,
		Consumers:     1,
		Processors:    1,
		CommitInOrder: true,
		ForceCommit:   false,
	}
}

func (c Config) ReceiptCommandQueue() kq.KqConf {
	return kq.KqConf{
		Brokers:       c.Kafka.Brokers,
		Group:         c.Kafka.ConsumerGroup + "-receipt-command",
		Topic:         c.Kafka.ReceiptCommandTopic,
		Offset:        "last",
		Conns:         1,
		Consumers:     1,
		Processors:    1,
		CommitInOrder: true,
		ForceCommit:   false,
	}
}
