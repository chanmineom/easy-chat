package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/chanmineom/easy-chat/apps/task/mq/internal/config"
	"github.com/chanmineom/easy-chat/apps/task/mq/internal/handler"
	"github.com/chanmineom/easy-chat/apps/task/mq/internal/svc"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
)

var configFile = flag.String("f", "etc/task.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())

	if err := c.SetUp(); err != nil {
		panic(err)
	}

	svcCtx, err := svc.NewServiceContext(context.Background(), c)
	if err != nil {
		panic(err)
	}
	defer svcCtx.Close(context.Background())

	listener := handler.NewListener(c, svcCtx)
	group := service.NewServiceGroup()
	for _, item := range listener.Services() {
		group.Add(item)
	}

	fmt.Printf("Starting server...\n")
	group.Start()
}
