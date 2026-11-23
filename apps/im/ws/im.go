package main

import (
	"flag"
	"fmt"
	"net/http"
	"time"

	"github.com/chanmineom/easy-chat/apps/im/session"
	"github.com/chanmineom/easy-chat/apps/im/ticket"
	"github.com/chanmineom/easy-chat/apps/im/ws/internal/config"
	"github.com/chanmineom/easy-chat/apps/im/ws/internal/gateway"
	wsgateway "github.com/chanmineom/easy-chat/apps/im/ws/internal/gateway"
	"github.com/chanmineom/easy-chat/apps/im/ws/internal/health"
	gatewaypb "github.com/chanmineom/easy-chat/apps/im/ws/rpc/gateway"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/proc"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
)

var configFile = flag.String("f", "etc/im.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())

	if c.NodeID == "" {
		panic("NodeID is required")
	}
	if c.RPCAdvertiseAddr == "" {
		panic("RPCAdvertiseAddr is required")
	}

	hub := gateway.NewHub(c.NodeID)

	chatPublisher, err := wsgateway.NewKafkaChatPublisher(c.Kafka.Brokers, c.Kafka.ChatCommandTopic)
	if err != nil {
		panic(err)
	}
	defer chatPublisher.Close()

	receiptPublisher, err := wsgateway.NewKafkaReceiptPusher(c.Kafka.Brokers, c.Kafka.ReceiptCommandTopic)
	if err != nil {
		panic(err)
	}
	defer receiptPublisher.Close()

	redisClient := redis.MustNewRedis(c.Redis)
	sessions := session.NewRedisRegistry(redisClient, 45*time.Second)

	readiness := health.NewReadiness(
		redisClient,
		[]string{
			c.Kafka.Brokers[0],
			c.WsRpc.Etcd.Hosts[0],
		},
	)

	proc.AddWrapUpListener(func() {
		readiness.MarkNotReady()
	})

	proc.AddShutdownListener(func() {
		logx.Infof(
			"closing websocket clients node=%s",
			c.NodeID,
		)
		hub.CloseAll("server shutting down")
	})

	tickets := ticket.NewRedisRegistry(
		redisClient,
		30*time.Second,
	)

	restServer := rest.MustNewServer(c.RestConf)
	restServer.AddRoutes([]rest.Route{
		{
			Method: http.MethodGet,
			Path: "/healthz",
			Handler: readiness.Liveness,
		},
		{
			Method: http.MethodGet,
			Path: "/readyz",
			Handler: readiness.Ready,
		},
		{
			Method: http.MethodGet,
			Path:   "/ws",
			Handler: wsgateway.NewConnectHandler(
				hub,
				c.JwtAuth.AccessSecret,
				c.WebSocket.AllowedOrigins,
				chatPublisher,
				receiptPublisher,
				sessions,
				c.NodeID,
				c.RPCAdvertiseAddr,
				tickets,
				gateway.Limits{
					MaxConnections:        c.WebSocket.MaxConnections,
					MaxConnectionsPerUser: c.WebSocket.MaxConnectionsPerUser,
					CommandRatePerSecond:  c.WebSocket.CommandRatePerSecond,
					CommandBurst:          c.WebSocket.CommandBurst,
				},
			).ServeHTTP,
		},
	})

	rpcServer := zrpc.MustNewServer(c.WsRpc, func(server *grpc.Server) {
		gatewaypb.RegisterGatewayServer(server, wsgateway.NewPushServer(hub))
	})

	group := service.NewServiceGroup()
	group.Add(restServer)
	group.Add(rpcServer)

	fmt.Printf("Starting websocket server at %s:%d...\n", c.Host, c.Port)
	fmt.Printf("Starting websocket internal grpc at %s...\n", c.WsRpc.ListenOn)
	group.Start()
}
