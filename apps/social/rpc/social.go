package main

import (
	"flag"
	"fmt"

	"github.com/chanmineom/easy-chat/apps/social/rpc/internal/config"
	"github.com/chanmineom/easy-chat/apps/social/rpc/internal/server"
	socialservice "github.com/chanmineom/easy-chat/apps/social/rpc/internal/service"
	"github.com/chanmineom/easy-chat/apps/social/rpc/internal/svc"
	"github.com/chanmineom/easy-chat/apps/social/rpc/social"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/social.yaml", "the config file")

func main() {
	flag.Parse()

	sqlx.DisableStmtLog()

	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())
	svcCtx := svc.NewServiceContext(c)

	rpcServer := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		social.RegisterSocialServer(grpcServer, server.NewSocialServer(svcCtx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})

	group := service.NewServiceGroup()
	group.Add(rpcServer)
	group.Add(
		socialservice.NewOutboxService(svcCtx.OutboxWorker),
	)

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	fmt.Println("Starting social outbox worker...")

	group.Start()
}
