package config

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	rest.RestConf

	NodeID           string
	RPCAdvertiseAddr string

	JwtAuth struct {
		AccessSecret string
	}

	WebSocket struct {
		AllowedOrigins       []string
		MaxConnections       int
		MaxConnectionsPerUser int
		CommandRatePerSecond int
		CommandBurst         int
	}

	WsRpc zrpc.RpcServerConf

	Redis redis.RedisConf

	Kafka struct {
		Brokers             []string
		ChatCommandTopic    string
		ReceiptCommandTopic string
	}
}
